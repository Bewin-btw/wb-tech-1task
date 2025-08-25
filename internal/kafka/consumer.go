package kafka

import (
	"context"
	"encoding/json"
	"errors"
	kafka "github.com/segmentio/kafka-go"
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
				log.Printf("Failed to fetch message: %v", err)
				continue
			}
			if err := c.processMessage(ctx, msg); err != nil {
				log.Printf("Failed to process message: %v", err)
				c.sendToDeadLetter(ctx, msg)
			} else {
				c.reader.CommitMessages(ctx, msg)
			}
		}
	}
}

func (c *Consumer) processMessage(ctx context.Context, msg kafka.Message) error {
	var order models.Order
	if err := json.Unmarshal(msg.Value, &order); err != nil {
		return err
	}

	if err := validateOrder(&order); err != nil {
		return err
	}

	return c.service.SaveOrder(ctx, &order)
}

func (c *Consumer) sendToDeadLetter(ctx context.Context, msg kafka.Message) {
	err := c.deadLetterWriter.WriteMessages(ctx, kafka.Message{
		Value: msg.Value,
		Headers: append(msg.Headers, kafka.Header{
			Key:   "error",
			Value: []byte("processing_failed"),
		}),
	})

	if err != nil {
		log.Printf("Failed to send to dead letter queue: %v", err)
	}
}

func validateOrder(order *models.Order) error {
	if order.OrderUID == "" {
		return errors.New("order_uid is required")
	}
	if order.TrackNumber == "" {
		return errors.New("track_number is required")
	}
	if order.Entry == "" {
		return errors.New("entry is required")
	}
	if order.CustomerID == "" {
		return errors.New("customer_id is required")
	}
	if order.DeliveryService == "" {
		return errors.New("delivery_service is required")
	}

	if order.Delivery.Name == "" {
		return errors.New("delivery.name is required")
	}
	if order.Payment.Transaction == "" {
		return errors.New("payment.transaction is required")
	}
	if len(order.Items) == 0 {
		return errors.New("items cannot be empty")
	}

	return nil
}

func (c *Consumer) Close() error {
	if err := c.reader.Close(); err != nil {
		return err
	}

	return c.deadLetterWriter.Close()
}
