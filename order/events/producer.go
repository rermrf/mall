package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/IBM/sarama"
)

type Producer interface {
	ProduceCloseDelay(ctx context.Context, orderNo string) error
	ProduceCancelled(ctx context.Context, evt OrderCancelledEvent) error
	ProduceCompleted(ctx context.Context, evt OrderCompletedEvent) error
}

type SaramaProducer struct {
	producer sarama.SyncProducer
}

func NewSaramaProducer(producer sarama.SyncProducer) Producer {
	return &SaramaProducer{producer: producer}
}

func (p *SaramaProducer) ProduceCloseDelay(ctx context.Context, orderNo string) error {
	msg := DelayMessage{
		Biz:       "order",
		Key:       orderNo,
		BizTopic:  TopicOrderCloseDelay,
		ExecuteAt: time.Now().Add(30 * time.Minute).Unix(),
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, _, err = p.producer.SendMessage(&sarama.ProducerMessage{
		Topic: TopicDelayMessage,
		Key:   sarama.StringEncoder(fmt.Sprintf("order:%s", orderNo)),
		Value: sarama.ByteEncoder(data),
	})
	return err
}

func (p *SaramaProducer) ProduceCancelled(ctx context.Context, evt OrderCancelledEvent) error {
	data, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	_, _, err = p.producer.SendMessage(&sarama.ProducerMessage{
		Topic: TopicOrderCancelled,
		Key:   sarama.StringEncoder(evt.OrderNo),
		Value: sarama.ByteEncoder(data),
	})
	return err
}

func (p *SaramaProducer) ProduceCompleted(ctx context.Context, evt OrderCompletedEvent) error {
	data, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	_, _, err = p.producer.SendMessage(&sarama.ProducerMessage{
		Topic: TopicOrderCompleted,
		Key:   sarama.StringEncoder(evt.OrderNo),
		Value: sarama.ByteEncoder(data),
	})
	return err
}
