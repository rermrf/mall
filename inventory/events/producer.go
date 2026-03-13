package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/IBM/sarama"
)

type Producer interface {
	ProduceDelay(ctx context.Context, orderId int64) error
	ProduceAlert(ctx context.Context, evt InventoryAlertEvent) error
}

type SaramaProducer struct {
	producer sarama.SyncProducer
}

func NewSaramaProducer(producer sarama.SyncProducer) Producer {
	return &SaramaProducer{producer: producer}
}

// ProduceDelay 发送延迟消息到 go-delay（30 分钟后投递到 inventory_deduct_expire）
func (p *SaramaProducer) ProduceDelay(ctx context.Context, orderId int64) error {
	msg := DelayMessage{
		Biz:       "inventory",
		Key:       fmt.Sprintf("%d", orderId),
		BizTopic:  TopicDeductExpire,
		ExecuteAt: time.Now().Add(30 * time.Minute).Unix(),
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, _, err = p.producer.SendMessage(&sarama.ProducerMessage{
		Topic: TopicDelayMessage,
		Key:   sarama.StringEncoder(fmt.Sprintf("inventory:%d", orderId)),
		Value: sarama.ByteEncoder(data),
	})
	return err
}

func (p *SaramaProducer) ProduceAlert(ctx context.Context, evt InventoryAlertEvent) error {
	data, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	_, _, err = p.producer.SendMessage(&sarama.ProducerMessage{
		Topic: TopicInventoryAlert,
		Value: sarama.ByteEncoder(data),
	})
	return err
}
