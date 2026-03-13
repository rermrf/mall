package events

import (
	"context"
	"encoding/json"

	"github.com/IBM/sarama"
)

const (
	TopicTenantApproved    = "tenant_approved"
	TopicTenantPlanChanged = "tenant_plan_changed"
)

type Producer interface {
	ProduceTenantApproved(ctx context.Context, evt TenantApprovedEvent) error
	ProduceTenantPlanChanged(ctx context.Context, evt TenantPlanChangedEvent) error
}

type SaramaProducer struct {
	producer sarama.SyncProducer
}

func NewSaramaProducer(producer sarama.SyncProducer) Producer {
	return &SaramaProducer{producer: producer}
}

func (p *SaramaProducer) ProduceTenantApproved(ctx context.Context, evt TenantApprovedEvent) error {
	data, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	_, _, err = p.producer.SendMessage(&sarama.ProducerMessage{
		Topic: TopicTenantApproved,
		Value: sarama.ByteEncoder(data),
	})
	return err
}

func (p *SaramaProducer) ProduceTenantPlanChanged(ctx context.Context, evt TenantPlanChangedEvent) error {
	data, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	_, _, err = p.producer.SendMessage(&sarama.ProducerMessage{
		Topic: TopicTenantPlanChanged,
		Value: sarama.ByteEncoder(data),
	})
	return err
}
