package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"testing"
	"time"

	kafka "github.com/segmentio/kafka-go"
	"wb-tech-1task/internal/models"
)

type fakeReader struct {
	msgs        []kafka.Message
	fetchCalled int
	committed   []kafka.Message
	closed      bool
}

func (r *fakeReader) FetchMessage(ctx context.Context) (kafka.Message, error) {
	if r.fetchCalled >= len(r.msgs) {
		return kafka.Message{}, io.EOF
	}
	m := r.msgs[r.fetchCalled]
	r.fetchCalled++
	return m, nil
}

func (r *fakeReader) CommitMessages(ctx context.Context, msgs ...kafka.Message) error {
	r.committed = append(r.committed, msgs...)
	return nil
}

func (r *fakeReader) Close() error {
	r.closed = true
	return nil
}

type fakeWriter struct {
	written []kafka.Message
	closed  bool
	err     error
}

func (w *fakeWriter) WriteMessages(ctx context.Context, msgs ...kafka.Message) error {
	if w.err != nil {
		return w.err
	}
	w.written = append(w.written, msgs...)
	return nil
}

func (w *fakeWriter) Close() error {
	w.closed = true
	return nil
}

type dummyService struct {
	saved *models.Order
	err   error
}

func (d *dummyService) SaveOrder(ctx context.Context, order *models.Order) error {
	if d.err != nil {
		return d.err
	}
	d.saved = order
	return nil
}

func sampleOrderJSON() []byte {
	order := models.Order{
		OrderUID:          "test123",
		TrackNumber:       "TRACK123",
		Entry:             "WBIL",
		Locale:            "en",
		InternalSignature: "",
		CustomerID:        "test_customer",
		DeliveryService:   "meest",
		Shardkey:          "9",
		SmID:              99,
		DateCreated:       time.Now(),
		OofShard:          "1",
		Delivery: models.Delivery{
			Name:    "Test Testov",
			Phone:   "+9720000000",
			Zip:     "2639809",
			City:    "Kiryat Mozkin",
			Address: "Ploshad Mira 15",
			Region:  "Kraiot",
			Email:   "test@gmail.com",
		},
		Payment: models.Payment{
			Transaction:  "test123",
			RequestID:    "",
			Currency:     "USD",
			Provider:     "wbpay",
			Amount:       1817,
			PaymentDt:    1637900000,
			Bank:         "alpha",
			DeliveryCost: 1500,
			GoodsTotal:   317,
			CustomFee:    0,
		},
		Items: []models.Item{
			{
				ChrtID:      9934930,
				TrackNumber: "TRACK123",
				Price:       453,
				Rid:         "ab4219087a764ae0btest",
				Name:        "Mascaras",
				Sale:        30,
				Size:        "0",
				TotalPrice:  317,
				NmID:        2389212,
				Brand:       "Vivienne Sabo",
				Status:      202,
			},
		},
	}

	data, _ := json.Marshal(order)
	return data
}

func hasNonZeroField(ptr interface{}) bool {
	if ptr == nil {
		return false
	}
	v := reflect.ValueOf(ptr)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return false
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return false
	}
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		switch f.Kind() {
		case reflect.String:
			if f.String() != "" {
				return true
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if f.Int() != 0 {
				return true
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if f.Uint() != 0 {
				return true
			}
		case reflect.Bool:
			if f.Bool() {
				return true
			}
		case reflect.Slice, reflect.Map, reflect.Interface, reflect.Ptr:
			if !f.IsNil() {
				return true
			}
		}
	}
	return false
}

func TestProcessMessage_Success(t *testing.T) {
	svc := &dummyService{}
	c := &Consumer{
		service: svc,
	}

	msg := kafka.Message{
		Key:   []byte("k1"),
		Value: sampleOrderJSON(),
	}

	if err := c.processMessage(context.Background(), msg); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if svc.saved == nil {
		t.Fatalf("expected service.SaveOrder to be called and save order")
	}
	if !hasNonZeroField(svc.saved) {
		t.Fatalf("saved order appears to have all zero fields (bad sample JSON or model mismatch)")
	}
}

func TestProcessMessage_InvalidJSON(t *testing.T) {
	svc := &dummyService{}
	c := &Consumer{service: svc}

	msg := kafka.Message{
		Value: []byte("{invalid-json"),
	}

	if err := c.processMessage(context.Background(), msg); err == nil {
		t.Fatalf("expected json unmarshal error, got nil")
	}
}

func TestProcessMessage_SaveOrderError(t *testing.T) {
	svc := &dummyService{err: errors.New("save failed")}
	c := &Consumer{service: svc}

	msg := kafka.Message{Value: sampleOrderJSON()}

	err := c.processMessage(context.Background(), msg)
	if err == nil {
		t.Fatalf("expected SaveOrder error, got nil")
	}
	if err.Error() != "save failed" {
		t.Fatalf("expected error 'save failed', got %v", err)
	}
}

func TestSendToDeadLetter_WritesMessageWithErrorHeader(t *testing.T) {
	fw := &fakeWriter{}
	c := &Consumer{deadLetterWriter: fw}

	orig := kafka.Message{
		Key:   []byte("k"),
		Value: []byte("v"),
		Headers: []kafka.Header{
			{Key: "h1", Value: []byte("v1")},
		},
	}

	procErr := errors.New("proc fail")
	if err := c.sendToDeadLetter(context.Background(), orig, procErr); err != nil {
		t.Fatalf("sendToDeadLetter returned error: %v", err)
	}

	if len(fw.written) != 1 {
		t.Fatalf("expected 1 written message, got %d", len(fw.written))
	}

	w := fw.written[0]
	if len(w.Headers) != 2 {
		t.Fatalf("expected 2 headers, got %d", len(w.Headers))
	}
	last := w.Headers[len(w.Headers)-1]
	if last.Key != "error" || string(last.Value) != procErr.Error() {
		t.Fatalf("expected error header with value %q, got %q/%q", procErr.Error(), last.Key, string(last.Value))
	}
	if string(w.Value) != string(orig.Value) {
		t.Fatalf("value mismatch: expected %s, got %s", string(orig.Value), string(w.Value))
	}
}

func TestRun_ProcessAndCommitThenEOFStops(t *testing.T) {
	msg := kafka.Message{Key: []byte("k"), Value: sampleOrderJSON(), Offset: 123}

	fr := &fakeReader{msgs: []kafka.Message{msg}}
	fw := &fakeWriter{}
	svc := &dummyService{}

	c := &Consumer{
		reader:           fr,
		deadLetterWriter: fw,
		service:          svc,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- c.Run(ctx)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Fatalf("Run did not finish in time")
	}

	if svc.saved == nil {
		t.Fatalf("expected service.SaveOrder to be called")
	}
	if len(fr.committed) != 1 {
		t.Fatalf("expected commit called once, got %d", len(fr.committed))
	}
	if fr.committed[0].Offset != msg.Offset {
		t.Fatalf("committed offset mismatch: expected %d got %d", msg.Offset, fr.committed[0].Offset)
	}
}

func TestClose_ClosesReaderAndWriter(t *testing.T) {
	fr := &fakeReader{}
	fw := &fakeWriter{}
	c := &Consumer{reader: fr, deadLetterWriter: fw}

	if err := c.Close(); err != nil {
		t.Fatalf("Close returned unexpected error: %v", err)
	}
	if !fr.closed {
		t.Fatalf("reader not closed")
	}
	if !fw.closed {
		t.Fatalf("writer not closed")
	}
}
