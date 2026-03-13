package events

import (
	"context"
	"encoding/json"

	"github.com/IBM/sarama"
)

const (
	TopicProductStatusChanged = "product_status_changed"
	TopicProductUpdated       = "product_updated"
)

type Producer interface {
	ProduceProductStatusChanged(ctx context.Context, evt ProductStatusChangedEvent) error
	ProduceProductUpdated(ctx context.Context, evt ProductUpdatedEvent) error
}

type SaramaProducer struct {
	producer sarama.SyncProducer
}

func NewSaramaProducer(producer sarama.SyncProducer) Producer {
	return &SaramaProducer{producer: producer}
}

func (p *SaramaProducer) ProduceProductStatusChanged(ctx context.Context, evt ProductStatusChangedEvent) error {
	data, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	_, _, err = p.producer.SendMessage(&sarama.ProducerMessage{
		Topic: TopicProductStatusChanged,
		Value: sarama.ByteEncoder(data),
	})
	return err
}

func (p *SaramaProducer) ProduceProductUpdated(ctx context.Context, evt ProductUpdatedEvent) error {
	data, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	_, _, err = p.producer.SendMessage(&sarama.ProducerMessage{
		Topic: TopicProductUpdated,
		Value: sarama.ByteEncoder(data),
	})
	return err
}
