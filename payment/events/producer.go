package events

import (
	"context"
	"encoding/json"

	"github.com/IBM/sarama"
)

type Producer interface {
	ProduceOrderPaid(ctx context.Context, evt OrderPaidEvent) error
}

type SaramaProducer struct {
	producer sarama.SyncProducer
}

func NewSaramaProducer(producer sarama.SyncProducer) Producer {
	return &SaramaProducer{producer: producer}
}

func (p *SaramaProducer) ProduceOrderPaid(ctx context.Context, evt OrderPaidEvent) error {
	data, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	_, _, err = p.producer.SendMessage(&sarama.ProducerMessage{
		Topic: TopicOrderPaid,
		Key:   sarama.StringEncoder(evt.OrderNo),
		Value: sarama.ByteEncoder(data),
	})
	return err
}
