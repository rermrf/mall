package events

import (
	"context"
	"encoding/json"

	"github.com/IBM/sarama"
)

const TopicUserRegistered = "user_registered"

type Producer interface {
	ProduceUserRegistered(ctx context.Context, evt UserRegisteredEvent) error
}

type SaramaProducer struct {
	producer sarama.SyncProducer
}

func NewSaramaProducer(producer sarama.SyncProducer) Producer {
	return &SaramaProducer{producer: producer}
}

func (p *SaramaProducer) ProduceUserRegistered(ctx context.Context, evt UserRegisteredEvent) error {
	data, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	_, _, err = p.producer.SendMessage(&sarama.ProducerMessage{
		Topic: TopicUserRegistered,
		Value: sarama.ByteEncoder(data),
	})
	return err
}
