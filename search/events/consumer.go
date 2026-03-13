package events

import (
	"context"
	"time"

	"github.com/IBM/sarama"
	"github.com/rermrf/emo/logger"
	productv1 "github.com/rermrf/mall/api/proto/gen/product/v1"
	"github.com/rermrf/mall/pkg/saramax"
	"github.com/rermrf/mall/search/domain"
	"github.com/rermrf/mall/search/service"
)

// ProductUpdatedConsumer 消费 product_updated 事件
type ProductUpdatedConsumer struct {
	client        sarama.ConsumerGroup
	l             logger.Logger
	productClient productv1.ProductServiceClient
	svc           service.SearchService
}

func NewProductUpdatedConsumer(
	client sarama.ConsumerGroup,
	l logger.Logger,
	productClient productv1.ProductServiceClient,
	svc service.SearchService,
) *ProductUpdatedConsumer {
	return &ProductUpdatedConsumer{
		client:        client,
		l:             l,
		productClient: productClient,
		svc:           svc,
	}
}

func (c *ProductUpdatedConsumer) Start() error {
	h := saramax.NewHandler[ProductUpdatedEvent](c.l, c.Consume)
	go func() {
		for {
			err := c.client.Consume(context.Background(), []string{TopicProductUpdated}, h)
			if err != nil {
				c.l.Error("消费 product_updated 事件出错", logger.Error(err))
				time.Sleep(time.Second)
			}
		}
	}()
	return nil
}

func (c *ProductUpdatedConsumer) Consume(msg *sarama.ConsumerMessage, evt ProductUpdatedEvent) error {
	ctx := context.Background()
	resp, err := c.productClient.GetProduct(ctx, &productv1.GetProductRequest{Id: evt.ProductId})
	if err != nil {
		return err
	}
	p := resp.GetProduct()
	if p.GetStatus() != 2 {
		return c.svc.DeleteProduct(ctx, evt.ProductId)
	}
	return c.svc.SyncProduct(ctx, productToDocument(p))
}

// ProductStatusChangedConsumer 消费 product_status_changed 事件
type ProductStatusChangedConsumer struct {
	client        sarama.ConsumerGroup
	l             logger.Logger
	productClient productv1.ProductServiceClient
	svc           service.SearchService
}

func NewProductStatusChangedConsumer(
	client sarama.ConsumerGroup,
	l logger.Logger,
	productClient productv1.ProductServiceClient,
	svc service.SearchService,
) *ProductStatusChangedConsumer {
	return &ProductStatusChangedConsumer{
		client:        client,
		l:             l,
		productClient: productClient,
		svc:           svc,
	}
}

func (c *ProductStatusChangedConsumer) Start() error {
	h := saramax.NewHandler[ProductStatusChangedEvent](c.l, c.Consume)
	go func() {
		for {
			err := c.client.Consume(context.Background(), []string{TopicProductStatusChanged}, h)
			if err != nil {
				c.l.Error("消费 product_status_changed 事件出错", logger.Error(err))
				time.Sleep(time.Second)
			}
		}
	}()
	return nil
}

func (c *ProductStatusChangedConsumer) Consume(msg *sarama.ConsumerMessage, evt ProductStatusChangedEvent) error {
	ctx := context.Background()
	if evt.NewStatus != 2 {
		return c.svc.DeleteProduct(ctx, evt.ProductId)
	}
	resp, err := c.productClient.GetProduct(ctx, &productv1.GetProductRequest{Id: evt.ProductId})
	if err != nil {
		return err
	}
	return c.svc.SyncProduct(ctx, productToDocument(resp.GetProduct()))
}

func productToDocument(p *productv1.Product) domain.ProductDocument {
	var minPrice int64
	if len(p.GetSkus()) > 0 {
		minPrice = p.GetSkus()[0].GetPrice()
		for _, sku := range p.GetSkus()[1:] {
			if sku.GetPrice() < minPrice {
				minPrice = sku.GetPrice()
			}
		}
	}
	return domain.ProductDocument{
		ID:         p.GetId(),
		TenantID:   p.GetTenantId(),
		Name:       p.GetName(),
		Subtitle:   p.GetSubtitle(),
		CategoryID: p.GetCategoryId(),
		Price:      minPrice,
		Sales:      p.GetSales(),
		MainImage:  p.GetMainImage(),
		Status:     p.GetStatus(),
	}
}
