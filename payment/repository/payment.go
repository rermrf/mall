package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/rermrf/mall/payment/domain"
	"github.com/rermrf/mall/payment/repository/cache"
	"github.com/rermrf/mall/payment/repository/dao"
)

type PaymentRepository interface {
	CreatePayment(ctx context.Context, payment domain.PaymentOrder) (domain.PaymentOrder, error)
	FindByPaymentNo(ctx context.Context, paymentNo string) (domain.PaymentOrder, error)
	FindByOrderNo(ctx context.Context, orderNo string) (domain.PaymentOrder, error)
	UpdateStatus(ctx context.Context, paymentNo string, oldStatus, newStatus domain.PaymentStatus, updates map[string]any) error
	ListPayments(ctx context.Context, tenantId int64, status int32, page, pageSize int32) ([]domain.PaymentOrder, int64, error)
	CreateRefund(ctx context.Context, refund domain.RefundRecord) error
	FindRefundByNo(ctx context.Context, refundNo string) (domain.RefundRecord, error)
	UpdateRefundStatus(ctx context.Context, refundNo string, status domain.RefundStatus, updates map[string]any) error
}

type paymentRepository struct {
	dao   dao.PaymentDAO
	cache cache.PaymentCache
}

func NewPaymentRepository(d dao.PaymentDAO, c cache.PaymentCache) PaymentRepository {
	return &paymentRepository{dao: d, cache: c}
}

func (r *paymentRepository) CreatePayment(ctx context.Context, payment domain.PaymentOrder) (domain.PaymentOrder, error) {
	model := r.toModel(payment)
	model, err := r.dao.CreatePayment(ctx, model)
	if err != nil {
		return domain.PaymentOrder{}, err
	}
	payment.ID = model.ID
	payment.Ctime = time.UnixMilli(model.Ctime)
	payment.Utime = time.UnixMilli(model.Utime)
	return payment, nil
}

func (r *paymentRepository) FindByPaymentNo(ctx context.Context, paymentNo string) (domain.PaymentOrder, error) {
	data, err := r.cache.GetPayment(ctx, paymentNo)
	if err == nil {
		var payment domain.PaymentOrder
		if json.Unmarshal(data, &payment) == nil {
			return payment, nil
		}
	}
	if err != nil && err != redis.Nil {
		// Redis 错误只记录，不阻塞
	}
	model, err := r.dao.FindByPaymentNo(ctx, paymentNo)
	if err != nil {
		return domain.PaymentOrder{}, err
	}
	payment := r.toDomain(model)
	r.setCache(ctx, paymentNo, payment)
	return payment, nil
}

func (r *paymentRepository) FindByOrderNo(ctx context.Context, orderNo string) (domain.PaymentOrder, error) {
	model, err := r.dao.FindByOrderNo(ctx, orderNo)
	if err != nil {
		return domain.PaymentOrder{}, err
	}
	return r.toDomain(model), nil
}

func (r *paymentRepository) UpdateStatus(ctx context.Context, paymentNo string, oldStatus, newStatus domain.PaymentStatus, updates map[string]any) error {
	err := r.dao.UpdateStatus(ctx, paymentNo, int32(oldStatus), int32(newStatus), updates)
	if err != nil {
		return err
	}
	_ = r.cache.DeletePayment(ctx, paymentNo)
	return nil
}

func (r *paymentRepository) ListPayments(ctx context.Context, tenantId int64, status int32, page, pageSize int32) ([]domain.PaymentOrder, int64, error) {
	offset := int((page - 1) * pageSize)
	models, total, err := r.dao.ListPayments(ctx, tenantId, status, offset, int(pageSize))
	if err != nil {
		return nil, 0, err
	}
	payments := make([]domain.PaymentOrder, 0, len(models))
	for _, m := range models {
		payments = append(payments, r.toDomain(m))
	}
	return payments, total, nil
}

func (r *paymentRepository) CreateRefund(ctx context.Context, refund domain.RefundRecord) error {
	return r.dao.CreateRefund(ctx, dao.RefundRecordModel{
		TenantId:  refund.TenantID,
		PaymentNo: refund.PaymentNo,
		RefundNo:  refund.RefundNo,
		Channel:   refund.Channel,
		Amount:    refund.Amount,
		Status:    int32(refund.Status),
	})
}

func (r *paymentRepository) FindRefundByNo(ctx context.Context, refundNo string) (domain.RefundRecord, error) {
	model, err := r.dao.FindRefundByNo(ctx, refundNo)
	if err != nil {
		return domain.RefundRecord{}, err
	}
	return r.toDomainRefund(model), nil
}

func (r *paymentRepository) UpdateRefundStatus(ctx context.Context, refundNo string, status domain.RefundStatus, updates map[string]any) error {
	return r.dao.UpdateRefundStatus(ctx, refundNo, int32(status), updates)
}

func (r *paymentRepository) setCache(ctx context.Context, paymentNo string, payment domain.PaymentOrder) {
	data, _ := json.Marshal(payment)
	_ = r.cache.SetPayment(ctx, paymentNo, data)
}

func (r *paymentRepository) toModel(p domain.PaymentOrder) dao.PaymentOrderModel {
	return dao.PaymentOrderModel{
		TenantId:       p.TenantID,
		PaymentNo:      p.PaymentNo,
		OrderId:        p.OrderID,
		OrderNo:        p.OrderNo,
		Channel:        p.Channel,
		Amount:         p.Amount,
		Status:         int32(p.Status),
		ChannelTradeNo: p.ChannelTradeNo,
		PayTime:        p.PayTime,
		ExpireTime:     p.ExpireTime,
		NotifyUrl:      p.NotifyUrl,
	}
}

func (r *paymentRepository) toDomain(m dao.PaymentOrderModel) domain.PaymentOrder {
	return domain.PaymentOrder{
		ID:             m.ID,
		TenantID:       m.TenantId,
		PaymentNo:      m.PaymentNo,
		OrderID:        m.OrderId,
		OrderNo:        m.OrderNo,
		Channel:        m.Channel,
		Amount:         m.Amount,
		Status:         domain.PaymentStatus(m.Status),
		ChannelTradeNo: m.ChannelTradeNo,
		PayTime:        m.PayTime,
		ExpireTime:     m.ExpireTime,
		NotifyUrl:      m.NotifyUrl,
		Ctime:          time.UnixMilli(m.Ctime),
		Utime:          time.UnixMilli(m.Utime),
	}
}

func (r *paymentRepository) toDomainRefund(m dao.RefundRecordModel) domain.RefundRecord {
	return domain.RefundRecord{
		ID:              m.ID,
		TenantID:        m.TenantId,
		PaymentNo:       m.PaymentNo,
		RefundNo:        m.RefundNo,
		Channel:         m.Channel,
		Amount:          m.Amount,
		Status:          domain.RefundStatus(m.Status),
		ChannelRefundNo: m.ChannelRefundNo,
		Ctime:           time.UnixMilli(m.Ctime),
		Utime:           time.UnixMilli(m.Utime),
	}
}
