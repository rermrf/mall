package events

import (
	"context"
	"encoding/json"

	"github.com/IBM/sarama"
)

type Producer interface {
	ProduceSeckillSuccess(ctx context.Context, evt SeckillSuccessEvent) error
}

type SaramaProducer struct {
	producer sarama.SyncProducer
}

func NewSaramaProducer(producer sarama.SyncProducer) Producer {
	return &SaramaProducer{producer: producer}
}

func (p *SaramaProducer) ProduceSeckillSuccess(ctx context.Context, evt SeckillSuccessEvent) error {
	data, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	_, _, err = p.producer.SendMessage(&sarama.ProducerMessage{
		Topic: TopicSeckillSuccess,
		Value: sarama.ByteEncoder(data),
	})
	return err
}
