package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/segmentio/kafka-go"
	"io"
	"log"
	"time"
	"wb-tech-1task/internal/models"
	"wb-tech-1task/internal/service"
)

type Consumer struct {
	reader           *kafka.Reader
	service          *service.OrderService
	deadLetterWriter *kafka.Writer
}

func NewConsumer(brokers []string, topic, groupID string, service *service.OrderService) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 10e3,
		MaxBytes: 10e6,
		MaxWait:  time.Second,
	})

	deadLetterWriter := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Topic:    topic + "_dead_letter",
		Balancer: &kafka.LeastBytes{},
	}

	return &Consumer{
		reader:           reader,
		service:          service,
		deadLetterWriter: deadLetterWriter}
}

func (c *Consumer) Start(ctx context.Context) {
	go c.consumeMessages(ctx)
}

func (c *Consumer) consumeMessages(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			msg, err := c.reader.FetchMessage(ctx)
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return
				}
				if err == io.EOF {
					return
				}
				log.Printf("Failed to fetch message: %v", err)
				continue
			}
			if err := c.processMessage(ctx, msg); err != nil {
				log.Printf("Failed to process message: %v", err)
				if derr := c.sendToDeadLetter(ctx, msg, err); derr != nil {
					log.Printf("Failed to send to dead letter queue: %v", derr)
				}
				continue
			} else {
				if err := c.reader.CommitMessages(ctx, msg); err != nil {
					log.Printf("failed to commit message offset=%d: %v", msg.Offset, err)
				}
			}
		}
	}
}

func (c *Consumer) processMessage(ctx context.Context, msg kafka.Message) error {
	var order models.Order
	if err := json.Unmarshal(msg.Value, &order); err != nil {
		return err
	}

	if err := order.Validate(); err != nil {
		return err
	}

	if err := c.service.SaveOrder(ctx, &order); err != nil {
		return err
	}
	return nil
}

func (c *Consumer) sendToDeadLetter(ctx context.Context, msg kafka.Message, procErr error) error {
	headers := append(msg.Headers, kafka.Header{
		Key:   "error",
		Value: []byte(procErr.Error()),
	})
	return c.deadLetterWriter.WriteMessages(ctx, kafka.Message{
		Key:     msg.Key,
		Value:   msg.Value,
		Headers: headers,
		Time:    time.Now(),
	})
}

func (c *Consumer) Close() error {
	var firstErr error
	if err := c.reader.Close(); err != nil {
		firstErr = err
	}
	if err := c.deadLetterWriter.Close(); err != nil && firstErr == nil {
		firstErr = err
	}
	return firstErr
}
