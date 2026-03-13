package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/rermrf/emo/idempotent"
	"github.com/rermrf/emo/logger"
	inventoryv1 "github.com/rermrf/mall/api/proto/gen/inventory/v1"
	paymentv1 "github.com/rermrf/mall/api/proto/gen/payment/v1"
	productv1 "github.com/rermrf/mall/api/proto/gen/product/v1"
	userv1 "github.com/rermrf/mall/api/proto/gen/user/v1"
	"github.com/rermrf/mall/order/domain"
	"github.com/rermrf/mall/order/events"
	"github.com/rermrf/mall/order/repository"
	"github.com/rermrf/mall/pkg/snowflake"
	"github.com/rermrf/mall/pkg/tenantx"
	"gorm.io/gorm"
)

type OrderService interface {
	CreateOrder(ctx context.Context, req CreateOrderReq) (string, int64, error)
	GetOrder(ctx context.Context, orderNo string) (domain.Order, error)
	ListOrders(ctx context.Context, buyerId, tenantId int64, status, page, pageSize int32) ([]domain.Order, int64, error)
	CancelOrder(ctx context.Context, orderNo string, buyerId int64) error
	ConfirmReceive(ctx context.Context, orderNo string, buyerId int64) error
	UpdateOrderStatus(ctx context.Context, orderNo string, status int32, operatorId int64, operatorType int32, remark string) error
	ApplyRefund(ctx context.Context, orderNo string, buyerId int64, refundType int32, amount int64, reason string) (string, error)
	HandleRefund(ctx context.Context, refundNo string, tenantId int64, approved bool, reason string) error
	GetRefundOrder(ctx context.Context, refundNo string) (domain.RefundOrder, error)
	ListRefundOrders(ctx context.Context, tenantId, buyerId int64, status, page, pageSize int32) ([]domain.RefundOrder, int64, error)
	// 内部方法：消费者调用
	HandleOrderPaid(ctx context.Context, evt events.OrderPaidEvent) error
	HandleOrderCloseDelay(ctx context.Context, orderNo string) error
}

type CreateOrderReq struct {
	BuyerID   int64
	TenantID  int64
	Items     []CreateOrderItemReq
	AddressID int64
	CouponID  int64
	Remark    string
	Channel      string // 支付渠道
	IsSeckill    bool   // 是否秒杀订单
	SeckillPrice int64  // 秒杀价格（分）
}

type CreateOrderItemReq struct {
	SKUID    int64
	Quantity int32
}

type orderService struct {
	repo            repository.OrderRepository
	producer        events.Producer
	idempotencySvc  idempotent.IdempotencyService
	node            *snowflake.Node
	productClient   productv1.ProductServiceClient
	inventoryClient inventoryv1.InventoryServiceClient
	paymentClient   paymentv1.PaymentServiceClient
	userClient      userv1.UserServiceClient
	l               logger.Logger
}

func NewOrderService(
	repo repository.OrderRepository,
	producer events.Producer,
	idempotencySvc idempotent.IdempotencyService,
	node *snowflake.Node,
	productClient productv1.ProductServiceClient,
	inventoryClient inventoryv1.InventoryServiceClient,
	paymentClient paymentv1.PaymentServiceClient,
	userClient userv1.UserServiceClient,
	l logger.Logger,
) OrderService {
	return &orderService{
		repo:            repo,
		producer:        producer,
		idempotencySvc:  idempotencySvc,
		node:            node,
		productClient:   productClient,
		inventoryClient: inventoryClient,
		paymentClient:   paymentClient,
		userClient:      userClient,
		l:               l,
	}
}

func (s *orderService) CreateOrder(ctx context.Context, req CreateOrderReq) (string, int64, error) {
	// 1. 布隆过滤器防重
	itemsHash := s.computeItemsHash(req.BuyerID, req.Items)
	bloomKey := fmt.Sprintf("order:create:%d:%s", req.BuyerID, itemsHash)
	exists, err := s.idempotencySvc.Exists(ctx, bloomKey)
	if err != nil {
		return "", 0, fmt.Errorf("防重检查失败: %w", err)
	}
	if exists {
		// 假阳性处理：查 MySQL 确认
		_, dbErr := s.repo.FindByBuyerIdAndHash(ctx, req.BuyerID, itemsHash)
		if dbErr == nil {
			return "", 0, fmt.Errorf("请勿重复提交")
		}
		if dbErr != gorm.ErrRecordNotFound {
			return "", 0, fmt.Errorf("防重查询失败: %w", dbErr)
		}
		// 假阳性，放行
	}

	// 2. 查收货地址
	addrResp, err := s.userClient.ListAddresses(ctx, &userv1.ListAddressesRequest{UserId: req.BuyerID})
	if err != nil {
		return "", 0, fmt.Errorf("查询地址失败: %w", err)
	}
	var addr *userv1.UserAddress
	for _, a := range addrResp.GetAddresses() {
		if a.GetId() == req.AddressID {
			addr = a
			break
		}
	}
	if addr == nil {
		return "", 0, fmt.Errorf("地址不存在")
	}

	// 3. 查商品信息 + 价格校验
	productIds := s.extractProductIds(req.Items)
	prodResp, err := s.productClient.BatchGetProducts(ctx, &productv1.BatchGetProductsRequest{Ids: productIds})
	if err != nil {
		return "", 0, fmt.Errorf("查询商品失败: %w", err)
	}
	// 构建 SKU 索引
	skuMap := s.buildSKUMap(prodResp.GetProducts())
	orderItems, totalAmount, err := s.buildOrderItems(req, skuMap)
	if err != nil {
		return "", 0, err
	}

	// 4. 生成订单号
	orderNo := fmt.Sprintf("%d", s.node.Generate())

	// 5. 预扣库存
	deductItems := make([]*inventoryv1.DeductItem, 0, len(req.Items))
	for _, item := range req.Items {
		deductItems = append(deductItems, &inventoryv1.DeductItem{
			SkuId:    item.SKUID,
			Quantity: item.Quantity,
		})
	}
	deductResp, err := s.inventoryClient.Deduct(ctx, &inventoryv1.DeductRequest{
		OrderId:  s.node.Generate(), // 用 snowflake 生成唯一 deduct order_id
		TenantId: req.TenantID,
		Items:    deductItems,
	})
	if err != nil {
		return "", 0, fmt.Errorf("库存预扣失败: %w", err)
	}
	if !deductResp.GetSuccess() {
		return "", 0, fmt.Errorf("库存不足: %s", deductResp.GetMessage())
	}

	// 6. MySQL 写入订单
	receiverAddr := fmt.Sprintf("%s%s%s%s", addr.GetProvince(), addr.GetCity(), addr.GetDistrict(), addr.GetDetail())
	order := domain.Order{
		TenantID:        req.TenantID,
		OrderNo:         orderNo,
		BuyerID:         req.BuyerID,
		BuyerHash:       itemsHash,
		Status:          domain.OrderStatusPending,
		TotalAmount:     totalAmount,
		PayAmount:       totalAmount, // 暂无优惠
		CouponID:        req.CouponID,
		ReceiverName:    addr.GetName(),
		ReceiverPhone:   addr.GetPhone(),
		ReceiverAddress: receiverAddr,
		Remark:          req.Remark,
		Items:           orderItems,
	}
	order, err = s.repo.CreateOrder(ctx, order)
	if err != nil {
		// 补偿：回滚库存
		if _, rollbackErr := s.inventoryClient.Rollback(ctx, &inventoryv1.RollbackRequest{OrderId: order.ID}); rollbackErr != nil {
			s.l.Error("库存回滚失败", logger.String("orderNo", orderNo), logger.Error(rollbackErr))
		}
		return "", 0, fmt.Errorf("创建订单失败: %w", err)
	}

	// 7. go-delay 超时关单
	if delayErr := s.producer.ProduceCloseDelay(ctx, orderNo); delayErr != nil {
		s.l.Error("发送超时关单延迟消息失败", logger.String("orderNo", orderNo), logger.Error(delayErr))
	}

	// 8. 创建支付单
	channel := req.Channel
	if channel == "" {
		channel = "mock"
	}
	payResp, err := s.paymentClient.CreatePayment(ctx, &paymentv1.CreatePaymentRequest{
		TenantId: req.TenantID,
		OrderId:  order.ID,
		OrderNo:  orderNo,
		Channel:  channel,
		Amount:   order.PayAmount,
	})
	if err != nil {
		// 补偿：回滚库存 + 取消订单
		if _, rollbackErr := s.inventoryClient.Rollback(ctx, &inventoryv1.RollbackRequest{OrderId: order.ID}); rollbackErr != nil {
			s.l.Error("库存回滚失败", logger.String("orderNo", orderNo), logger.Error(rollbackErr))
		}
		if statusErr := s.repo.UpdateStatus(ctx, orderNo, domain.OrderStatusPending, domain.OrderStatusCancelled, map[string]any{
			"closed_at": time.Now().UnixMilli(),
		}); statusErr != nil {
			s.l.Error("取消订单状态更新失败", logger.String("orderNo", orderNo), logger.Error(statusErr))
		}
		return "", 0, fmt.Errorf("创建支付单失败: %w", err)
	}
	if updateErr := s.repo.UpdatePaymentNo(ctx, orderNo, payResp.GetPaymentNo()); updateErr != nil {
		s.l.Error("更新支付单号失败", logger.String("orderNo", orderNo), logger.Error(updateErr))
	}

	return orderNo, order.PayAmount, nil
}

func (s *orderService) GetOrder(ctx context.Context, orderNo string) (domain.Order, error) {
	return s.repo.FindByOrderNo(ctx, orderNo)
}

func (s *orderService) ListOrders(ctx context.Context, buyerId, tenantId int64, status, page, pageSize int32) ([]domain.Order, int64, error) {
	return s.repo.ListOrders(ctx, buyerId, tenantId, status, page, pageSize)
}

func (s *orderService) CancelOrder(ctx context.Context, orderNo string, buyerId int64) error {
	order, err := s.repo.FindByOrderNo(ctx, orderNo)
	if err != nil {
		return err
	}
	if order.BuyerID != buyerId {
		return fmt.Errorf("无权取消此订单")
	}
	if order.Status != domain.OrderStatusPending {
		return fmt.Errorf("当前状态不允许取消")
	}
	err = s.repo.UpdateStatus(ctx, orderNo, domain.OrderStatusPending, domain.OrderStatusCancelled, map[string]any{
		"closed_at": time.Now().UnixMilli(),
	})
	if err != nil {
		return err
	}
	// 回滚库存
	if _, rollbackErr := s.inventoryClient.Rollback(ctx, &inventoryv1.RollbackRequest{OrderId: order.ID}); rollbackErr != nil {
		s.l.Error("库存回滚失败", logger.String("orderNo", orderNo), logger.Error(rollbackErr))
	}
	// 关闭支付单
	if order.PaymentNo != "" {
		if _, closeErr := s.paymentClient.ClosePayment(ctx, &paymentv1.ClosePaymentRequest{PaymentNo: order.PaymentNo}); closeErr != nil {
			s.l.Error("关闭支付单失败", logger.String("orderNo", orderNo), logger.String("paymentNo", order.PaymentNo), logger.Error(closeErr))
		}
	}
	// 写日志
	if logErr := s.repo.InsertStatusLog(ctx, domain.OrderStatusLog{
		OrderID:      order.ID,
		FromStatus:   domain.OrderStatusPending,
		ToStatus:     domain.OrderStatusCancelled,
		OperatorID:   buyerId,
		OperatorType: 1,
		Remark:       "买家取消订单",
	}); logErr != nil {
		s.l.Error("写入状态日志失败", logger.String("orderNo", orderNo), logger.Error(logErr))
	}
	if produceErr := s.producer.ProduceCancelled(ctx, events.OrderCancelledEvent{
		OrderNo: orderNo, TenantID: order.TenantID, Reason: "买家取消",
	}); produceErr != nil {
		s.l.Error("发送取消事件失败", logger.String("orderNo", orderNo), logger.Error(produceErr))
	}
	return nil
}

func (s *orderService) ConfirmReceive(ctx context.Context, orderNo string, buyerId int64) error {
	order, err := s.repo.FindByOrderNo(ctx, orderNo)
	if err != nil {
		return err
	}
	if order.BuyerID != buyerId {
		return fmt.Errorf("无权操作此订单")
	}
	if order.Status != domain.OrderStatusShipped {
		return fmt.Errorf("当前状态不允许确认收货")
	}
	err = s.repo.UpdateStatus(ctx, orderNo, domain.OrderStatusShipped, domain.OrderStatusReceived, map[string]any{
		"received_at": time.Now().UnixMilli(),
	})
	if err != nil {
		return err
	}
	if logErr := s.repo.InsertStatusLog(ctx, domain.OrderStatusLog{
		OrderID: order.ID, FromStatus: domain.OrderStatusShipped, ToStatus: domain.OrderStatusReceived,
		OperatorID: buyerId, OperatorType: 1, Remark: "确认收货",
	}); logErr != nil {
		s.l.Error("写入状态日志失败", logger.String("orderNo", orderNo), logger.Error(logErr))
	}
	return nil
}

func (s *orderService) UpdateOrderStatus(ctx context.Context, orderNo string, status int32, operatorId int64, operatorType int32, remark string) error {
	order, err := s.repo.FindByOrderNo(ctx, orderNo)
	if err != nil {
		return err
	}
	newStatus := domain.OrderStatus(status)
	updates := map[string]any{}
	switch newStatus {
	case domain.OrderStatusShipped:
		updates["shipped_at"] = time.Now().UnixMilli()
	case domain.OrderStatusCompleted:
		// 触发完成事件
		defer func() {
			items := make([]events.CompletedItemInfo, 0, len(order.Items))
			for _, item := range order.Items {
				items = append(items, events.CompletedItemInfo{ProductID: item.ProductID, Quantity: item.Quantity})
			}
			if produceErr := s.producer.ProduceCompleted(ctx, events.OrderCompletedEvent{
				OrderNo: orderNo, TenantID: order.TenantID, Items: items,
			}); produceErr != nil {
				s.l.Error("发送完成事件失败", logger.String("orderNo", orderNo), logger.Error(produceErr))
			}
		}()
	}
	err = s.repo.UpdateStatus(ctx, orderNo, order.Status, newStatus, updates)
	if err != nil {
		return err
	}
	if logErr := s.repo.InsertStatusLog(ctx, domain.OrderStatusLog{
		OrderID: order.ID, FromStatus: order.Status, ToStatus: newStatus,
		OperatorID: operatorId, OperatorType: operatorType, Remark: remark,
	}); logErr != nil {
		s.l.Error("写入状态日志失败", logger.String("orderNo", orderNo), logger.Error(logErr))
	}
	return nil
}

func (s *orderService) ApplyRefund(ctx context.Context, orderNo string, buyerId int64, refundType int32, amount int64, reason string) (string, error) {
	order, err := s.repo.FindByOrderNo(ctx, orderNo)
	if err != nil {
		return "", err
	}
	if order.BuyerID != buyerId {
		return "", fmt.Errorf("无权申请退款")
	}
	if order.Status != domain.OrderStatusPaid && order.Status != domain.OrderStatusShipped && order.Status != domain.OrderStatusReceived {
		return "", fmt.Errorf("当前状态不允许退款")
	}
	if amount > order.PayAmount-order.RefundedAmount {
		return "", fmt.Errorf("退款金额超出可退金额")
	}
	refundNo := fmt.Sprintf("R%d", s.node.Generate())
	refund := domain.RefundOrder{
		TenantID:     order.TenantID,
		OrderID:      order.ID,
		RefundNo:     refundNo,
		BuyerID:      buyerId,
		Type:         refundType,
		Status:       domain.RefundStatusPending,
		RefundAmount: amount,
		Reason:       reason,
	}
	if err := s.repo.CreateRefund(ctx, refund); err != nil {
		return "", err
	}
	return refundNo, nil
}

func (s *orderService) HandleRefund(ctx context.Context, refundNo string, tenantId int64, approved bool, reason string) error {
	refund, err := s.repo.FindRefundByNo(ctx, refundNo)
	if err != nil {
		return err
	}
	if refund.Status != domain.RefundStatusPending {
		return fmt.Errorf("退款单状态不允许处理")
	}
	if !approved {
		return s.repo.UpdateRefundStatus(ctx, refundNo, domain.RefundStatusRejected, map[string]any{
			"reject_reason": reason,
		})
	}
	// 审核通过 → 更新状态为退款中
	err = s.repo.UpdateRefundStatus(ctx, refundNo, domain.RefundStatusRefunding, nil)
	if err != nil {
		return err
	}
	s.l.Info("退款审核通过", logger.String("refundNo", refundNo))
	return nil
}

func (s *orderService) GetRefundOrder(ctx context.Context, refundNo string) (domain.RefundOrder, error) {
	return s.repo.FindRefundByNo(ctx, refundNo)
}

func (s *orderService) ListRefundOrders(ctx context.Context, tenantId, buyerId int64, status, page, pageSize int32) ([]domain.RefundOrder, int64, error) {
	return s.repo.ListRefunds(ctx, tenantId, buyerId, status, page, pageSize)
}

// HandleOrderPaid 消费 order_paid 事件
func (s *orderService) HandleOrderPaid(ctx context.Context, evt events.OrderPaidEvent) error {
	order, err := s.repo.FindByOrderNo(ctx, evt.OrderNo)
	if err != nil {
		return err
	}
	if order.Status != domain.OrderStatusPending {
		return nil // 幂等
	}
	ctx = tenantx.WithTenantID(ctx, order.TenantID)
	err = s.repo.UpdateStatus(ctx, evt.OrderNo, domain.OrderStatusPending, domain.OrderStatusPaid, map[string]any{
		"paid_at":    evt.PaidAt,
		"payment_no": evt.PaymentNo,
	})
	if err != nil {
		return err
	}
	// 确认库存扣减
	_, err = s.inventoryClient.Confirm(ctx, &inventoryv1.ConfirmRequest{OrderId: order.ID})
	if err != nil {
		s.l.Error("确认库存扣减失败", logger.String("orderNo", evt.OrderNo), logger.Error(err))
	}
	if logErr := s.repo.InsertStatusLog(ctx, domain.OrderStatusLog{
		OrderID: order.ID, FromStatus: domain.OrderStatusPending, ToStatus: domain.OrderStatusPaid,
		OperatorType: 4, Remark: "支付成功",
	}); logErr != nil {
		s.l.Error("写入状态日志失败", logger.String("orderNo", evt.OrderNo), logger.Error(logErr))
	}
	return nil
}

// HandleOrderCloseDelay 超时关单
func (s *orderService) HandleOrderCloseDelay(ctx context.Context, orderNo string) error {
	order, err := s.repo.FindByOrderNo(ctx, orderNo)
	if err != nil {
		return err
	}
	if order.Status != domain.OrderStatusPending {
		return nil // 已支付或已取消，跳过
	}
	ctx = tenantx.WithTenantID(ctx, order.TenantID)
	err = s.repo.UpdateStatus(ctx, orderNo, domain.OrderStatusPending, domain.OrderStatusCancelled, map[string]any{
		"closed_at": time.Now().UnixMilli(),
	})
	if err != nil {
		return err
	}
	// 回滚库存
	if _, rollbackErr := s.inventoryClient.Rollback(ctx, &inventoryv1.RollbackRequest{OrderId: order.ID}); rollbackErr != nil {
		s.l.Error("库存回滚失败", logger.String("orderNo", orderNo), logger.Error(rollbackErr))
	}
	// 关闭支付单
	if order.PaymentNo != "" {
		if _, closeErr := s.paymentClient.ClosePayment(ctx, &paymentv1.ClosePaymentRequest{PaymentNo: order.PaymentNo}); closeErr != nil {
			s.l.Error("关闭支付单失败", logger.String("orderNo", orderNo), logger.String("paymentNo", order.PaymentNo), logger.Error(closeErr))
		}
	}
	if logErr := s.repo.InsertStatusLog(ctx, domain.OrderStatusLog{
		OrderID: order.ID, FromStatus: domain.OrderStatusPending, ToStatus: domain.OrderStatusCancelled,
		OperatorType: 4, Remark: "超时未支付，自动关单",
	}); logErr != nil {
		s.l.Error("写入状态日志失败", logger.String("orderNo", orderNo), logger.Error(logErr))
	}
	if produceErr := s.producer.ProduceCancelled(ctx, events.OrderCancelledEvent{
		OrderNo: orderNo, TenantID: order.TenantID, Reason: "超时未支付",
	}); produceErr != nil {
		s.l.Error("发送取消事件失败", logger.String("orderNo", orderNo), logger.Error(produceErr))
	}
	return nil
}

func (s *orderService) computeItemsHash(buyerId int64, items []CreateOrderItemReq) string {
	// 排序保证确定性
	sorted := make([]CreateOrderItemReq, len(items))
	copy(sorted, items)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].SKUID < sorted[j].SKUID })
	data, marshalErr := json.Marshal(sorted)
	if marshalErr != nil {
		s.l.Error("序列化订单项失败", logger.Error(marshalErr))
	}
	h := sha256.Sum256(append([]byte(fmt.Sprintf("%d:", buyerId)), data...))
	return fmt.Sprintf("%x", h[:16])
}

func (s *orderService) extractProductIds(items []CreateOrderItemReq) []int64 {
	// CreateOrderReq 中 item 只有 sku_id
	// BFF 层应该传递 product_id，这里暂时返回空
	// 在 buildOrderItems 中通过 BatchGetProducts 获取
	return nil
}

func (s *orderService) buildSKUMap(products []*productv1.Product) map[int64]*productv1.ProductSKU {
	m := make(map[int64]*productv1.ProductSKU)
	for _, p := range products {
		for _, sku := range p.GetSkus() {
			m[sku.GetId()] = sku
		}
	}
	return m
}

func (s *orderService) buildOrderItems(req CreateOrderReq, skuMap map[int64]*productv1.ProductSKU) ([]domain.OrderItem, int64, error) {
	var totalAmount int64
	items := make([]domain.OrderItem, 0, len(req.Items))
	for _, ri := range req.Items {
		sku, ok := skuMap[ri.SKUID]
		if !ok {
			return nil, 0, fmt.Errorf("SKU %d 不存在", ri.SKUID)
		}
		price := sku.GetPrice()
		// 秒杀订单使用秒杀价
		if req.IsSeckill && req.SeckillPrice > 0 {
			price = req.SeckillPrice
		}
		subtotal := price * int64(ri.Quantity)
		totalAmount += subtotal
		items = append(items, domain.OrderItem{
			TenantID:    req.TenantID,
			ProductID:   sku.GetProductId(),
			SKUID:       ri.SKUID,
			ProductName: sku.GetSpecValues(),
			SKUSpec:     sku.GetSpecValues(),
			Price:       price,
			Quantity:    ri.Quantity,
			Subtotal:    subtotal,
		})
	}
	return items, totalAmount, nil
}
