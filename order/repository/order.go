package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/rermrf/mall/order/domain"
	"github.com/rermrf/mall/order/repository/cache"
	"github.com/rermrf/mall/order/repository/dao"
)

type OrderRepository interface {
	CreateOrder(ctx context.Context, order domain.Order) (domain.Order, error)
	FindByID(ctx context.Context, id int64) (domain.Order, error)
	FindByOrderNo(ctx context.Context, orderNo string) (domain.Order, error)
	FindByBuyerIdAndHash(ctx context.Context, buyerId int64, hash string) (domain.Order, error)
	ListOrders(ctx context.Context, buyerId, tenantId int64, status int32, page, pageSize int32) ([]domain.Order, int64, error)
	UpdateStatus(ctx context.Context, orderNo string, oldStatus, newStatus domain.OrderStatus, updates map[string]any) error
	UpdatePaymentNo(ctx context.Context, orderNo string, paymentNo string) error
	AddRefundedAmount(ctx context.Context, orderNo string, amount int64) error
	InsertStatusLog(ctx context.Context, log domain.OrderStatusLog) error
	CreateRefund(ctx context.Context, refund domain.RefundOrder) error
	FindRefundByNo(ctx context.Context, refundNo string) (domain.RefundOrder, error)
	FindLatestActiveRefundByOrderID(ctx context.Context, orderId int64, amount int64) (domain.RefundOrder, error)
	UpdateRefundStatus(ctx context.Context, refundNo string, status domain.RefundStatus, updates map[string]any) error
	ListRefunds(ctx context.Context, tenantId, buyerId int64, status int32, page, pageSize int32) ([]domain.RefundOrder, int64, error)
}

type orderRepository struct {
	dao   dao.OrderDAO
	cache cache.OrderCache
}

func NewOrderRepository(d dao.OrderDAO, c cache.OrderCache) OrderRepository {
	return &orderRepository{dao: d, cache: c}
}

func (r *orderRepository) CreateOrder(ctx context.Context, order domain.Order) (domain.Order, error) {
	orderModel := r.toOrderModel(order)
	itemModels := r.toItemModels(order.Items)
	db := r.dao.GetDB()
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := r.dao.CreateOrder(ctx, tx, orderModel, itemModels); err != nil {
			return err
		}
		// 写状态日志
		return r.dao.InsertStatusLog(ctx, tx, dao.OrderStatusLogModel{
			OrderId:      orderModel.ID,
			FromStatus:   0,
			ToStatus:     int32(domain.OrderStatusPending),
			OperatorId:   order.BuyerID,
			OperatorType: 1,
			Remark:       "创建订单",
		})
	})
	if err != nil {
		return domain.Order{}, err
	}
	order.ID = orderModel.ID
	return order, nil
}

func (r *orderRepository) FindByID(ctx context.Context, id int64) (domain.Order, error) {
	model, err := r.dao.FindByID(ctx, id)
	if err != nil {
		return domain.Order{}, err
	}
	return r.toDomainOrder(model), nil
}

func (r *orderRepository) FindByOrderNo(ctx context.Context, orderNo string) (domain.Order, error) {
	// 尝试缓存
	data, err := r.cache.GetOrder(ctx, orderNo)
	if err == nil {
		var cd cache.OrderCacheData
		if json.Unmarshal(data, &cd) == nil {
			var order domain.Order
			var items []domain.OrderItem
			if json.Unmarshal(cd.Order, &order) == nil && json.Unmarshal(cd.Items, &items) == nil {
				order.Items = items
				return order, nil
			}
		}
	}
	if err != nil && err != redis.Nil {
		// Redis 错误只记录，不阻塞
	}
	// 查 MySQL
	orderModel, err := r.dao.FindByOrderNo(ctx, orderNo)
	if err != nil {
		return domain.Order{}, err
	}
	itemModels, err := r.dao.FindItemsByOrderId(ctx, orderModel.ID)
	if err != nil {
		return domain.Order{}, err
	}
	order := r.toDomainOrder(orderModel)
	order.Items = r.toDomainItems(itemModels)
	// 回填缓存
	r.setCache(ctx, orderNo, order)
	return order, nil
}

func (r *orderRepository) FindByBuyerIdAndHash(ctx context.Context, buyerId int64, hash string) (domain.Order, error) {
	model, err := r.dao.FindByBuyerIdAndHash(ctx, buyerId, hash)
	if err != nil {
		return domain.Order{}, err
	}
	return r.toDomainOrder(model), nil
}

func (r *orderRepository) ListOrders(ctx context.Context, buyerId, tenantId int64, status int32, page, pageSize int32) ([]domain.Order, int64, error) {
	offset := int((page - 1) * pageSize)
	models, total, err := r.dao.ListOrders(ctx, buyerId, tenantId, status, offset, int(pageSize))
	if err != nil {
		return nil, 0, err
	}
	orders := make([]domain.Order, 0, len(models))
	for _, m := range models {
		orders = append(orders, r.toDomainOrder(m))
	}
	return orders, total, nil
}

func (r *orderRepository) UpdateStatus(ctx context.Context, orderNo string, oldStatus, newStatus domain.OrderStatus, updates map[string]any) error {
	err := r.dao.UpdateStatus(ctx, orderNo, int32(oldStatus), int32(newStatus), updates)
	if err != nil {
		return err
	}
	_ = r.cache.DeleteOrder(ctx, orderNo)
	return nil
}

func (r *orderRepository) UpdatePaymentNo(ctx context.Context, orderNo string, paymentNo string) error {
	return r.dao.UpdatePaymentNo(ctx, orderNo, paymentNo)
}

func (r *orderRepository) AddRefundedAmount(ctx context.Context, orderNo string, amount int64) error {
	err := r.dao.AddRefundedAmount(ctx, orderNo, amount)
	if err != nil {
		return err
	}
	_ = r.cache.DeleteOrder(ctx, orderNo)
	return nil
}

func (r *orderRepository) InsertStatusLog(ctx context.Context, log domain.OrderStatusLog) error {
	db := r.dao.GetDB()
	return r.dao.InsertStatusLog(ctx, db, dao.OrderStatusLogModel{
		OrderId:      log.OrderID,
		FromStatus:   int32(log.FromStatus),
		ToStatus:     int32(log.ToStatus),
		OperatorId:   log.OperatorID,
		OperatorType: log.OperatorType,
		Remark:       log.Remark,
	})
}

func (r *orderRepository) CreateRefund(ctx context.Context, refund domain.RefundOrder) error {
	return r.dao.CreateRefund(ctx, dao.RefundOrderModel{
		TenantId:     refund.TenantID,
		OrderId:      refund.OrderID,
		RefundNo:     refund.RefundNo,
		BuyerId:      refund.BuyerID,
		Type:         refund.Type,
		Status:       int32(refund.Status),
		RefundAmount: refund.RefundAmount,
		Reason:       refund.Reason,
		Items:        refund.Items,
	})
}

func (r *orderRepository) FindRefundByNo(ctx context.Context, refundNo string) (domain.RefundOrder, error) {
	model, err := r.dao.FindRefundByNo(ctx, refundNo)
	if err != nil {
		return domain.RefundOrder{}, err
	}
	return r.toDomainRefund(model), nil
}

func (r *orderRepository) FindLatestActiveRefundByOrderID(ctx context.Context, orderId int64, amount int64) (domain.RefundOrder, error) {
	model, err := r.dao.FindLatestActiveRefundByOrderID(ctx, orderId, amount)
	if err != nil {
		return domain.RefundOrder{}, err
	}
	return r.toDomainRefund(model), nil
}

func (r *orderRepository) UpdateRefundStatus(ctx context.Context, refundNo string, status domain.RefundStatus, updates map[string]any) error {
	return r.dao.UpdateRefundStatus(ctx, refundNo, int32(status), updates)
}

func (r *orderRepository) ListRefunds(ctx context.Context, tenantId, buyerId int64, status int32, page, pageSize int32) ([]domain.RefundOrder, int64, error) {
	offset := int((page - 1) * pageSize)
	models, total, err := r.dao.ListRefunds(ctx, tenantId, buyerId, status, offset, int(pageSize))
	if err != nil {
		return nil, 0, err
	}
	refunds := make([]domain.RefundOrder, 0, len(models))
	for _, m := range models {
		refunds = append(refunds, r.toDomainRefund(m))
	}
	return refunds, total, nil
}

func (r *orderRepository) setCache(ctx context.Context, orderNo string, order domain.Order) {
	orderJSON, _ := json.Marshal(order)
	itemsJSON, _ := json.Marshal(order.Items)
	cd := cache.OrderCacheData{Order: orderJSON, Items: itemsJSON}
	data, _ := json.Marshal(cd)
	_ = r.cache.SetOrder(ctx, orderNo, data)
}

func (r *orderRepository) toOrderModel(o domain.Order) dao.OrderModel {
	return dao.OrderModel{
		TenantId:        o.TenantID,
		OrderNo:         o.OrderNo,
		BuyerId:         o.BuyerID,
		BuyerHash:       o.BuyerHash,
		Status:          int32(o.Status),
		TotalAmount:     o.TotalAmount,
		DiscountAmount:  o.DiscountAmount,
		FreightAmount:   o.FreightAmount,
		PayAmount:       o.PayAmount,
		CouponId:        o.CouponID,
		ReceiverName:    o.ReceiverName,
		ReceiverPhone:   o.ReceiverPhone,
		ReceiverAddress: o.ReceiverAddress,
		Remark:          o.Remark,
	}
}

func (r *orderRepository) toItemModels(items []domain.OrderItem) []dao.OrderItemModel {
	models := make([]dao.OrderItemModel, 0, len(items))
	for _, item := range items {
		models = append(models, dao.OrderItemModel{
			TenantId:    item.TenantID,
			ProductId:   item.ProductID,
			SkuId:       item.SKUID,
			ProductName: item.ProductName,
			SkuSpec:     item.SKUSpec,
			Image:       item.Image,
			Price:       item.Price,
			Quantity:    item.Quantity,
			Subtotal:    item.Subtotal,
		})
	}
	return models
}

func (r *orderRepository) toDomainOrder(m dao.OrderModel) domain.Order {
	return domain.Order{
		ID:              m.ID,
		TenantID:        m.TenantId,
		OrderNo:         m.OrderNo,
		BuyerID:         m.BuyerId,
		BuyerHash:       m.BuyerHash,
		Status:          domain.OrderStatus(m.Status),
		TotalAmount:     m.TotalAmount,
		DiscountAmount:  m.DiscountAmount,
		FreightAmount:   m.FreightAmount,
		PayAmount:       m.PayAmount,
		RefundedAmount:  m.RefundedAmount,
		CouponID:        m.CouponId,
		PaymentNo:       m.PaymentNo,
		ReceiverName:    m.ReceiverName,
		ReceiverPhone:   m.ReceiverPhone,
		ReceiverAddress: m.ReceiverAddress,
		Remark:          m.Remark,
		PaidAt:          m.PaidAt,
		ShippedAt:       m.ShippedAt,
		ReceivedAt:      m.ReceivedAt,
		ClosedAt:        m.ClosedAt,
		Ctime:           time.UnixMilli(m.Ctime),
		Utime:           time.UnixMilli(m.Utime),
	}
}

func (r *orderRepository) toDomainItems(models []dao.OrderItemModel) []domain.OrderItem {
	items := make([]domain.OrderItem, 0, len(models))
	for _, m := range models {
		items = append(items, domain.OrderItem{
			ID:               m.ID,
			OrderID:          m.OrderId,
			TenantID:         m.TenantId,
			ProductID:        m.ProductId,
			SKUID:            m.SkuId,
			ProductName:      m.ProductName,
			SKUSpec:          m.SkuSpec,
			Image:            m.Image,
			Price:            m.Price,
			Quantity:         m.Quantity,
			Subtotal:         m.Subtotal,
			RefundedQuantity: m.RefundedQuantity,
			Ctime:            time.UnixMilli(m.Ctime),
		})
	}
	return items
}

func (r *orderRepository) toDomainRefund(m dao.RefundOrderModel) domain.RefundOrder {
	return domain.RefundOrder{
		ID:           m.ID,
		TenantID:     m.TenantId,
		OrderID:      m.OrderId,
		RefundNo:     m.RefundNo,
		BuyerID:      m.BuyerId,
		Type:         m.Type,
		Status:       domain.RefundStatus(m.Status),
		RefundAmount: m.RefundAmount,
		Reason:       m.Reason,
		RejectReason: m.RejectReason,
		Items:        m.Items,
		Ctime:        time.UnixMilli(m.Ctime),
		Utime:        time.UnixMilli(m.Utime),
	}
}
