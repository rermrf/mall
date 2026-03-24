package service

import (
	"context"
	"errors"

	"github.com/rermrf/emo/idempotent"
	"github.com/rermrf/emo/logger"
	"github.com/rermrf/mall/payment/domain"
	paymentevents "github.com/rermrf/mall/payment/events"
	"github.com/rermrf/mall/payment/repository/dao"
	"github.com/rermrf/mall/payment/service/channel"
	"github.com/rermrf/mall/pkg/snowflake"
	"gorm.io/gorm"
)

type fakePaymentRepo struct {
	createPaymentFn              func(ctx context.Context, payment domain.PaymentOrder) (domain.PaymentOrder, error)
	findByPaymentNoFn            func(ctx context.Context, paymentNo string) (domain.PaymentOrder, error)
	findByOrderNoFn              func(ctx context.Context, orderNo string) (domain.PaymentOrder, error)
	listOpenPaymentsByOrderNoFn  func(ctx context.Context, orderNo string) ([]domain.PaymentOrder, error)
	updateStatusFn               func(ctx context.Context, paymentNo string, oldStatus, newStatus domain.PaymentStatus, updates map[string]any) error
	listPaymentsFn               func(ctx context.Context, tenantId int64, status, page, pageSize int32) ([]domain.PaymentOrder, int64, error)
	listPaymentsByDateAndChannelFn func(ctx context.Context, ch string, startTime, endTime int64) ([]domain.PaymentOrder, error)
	createRefundFn               func(ctx context.Context, refund domain.RefundRecord) error
	findRefundByNoFn             func(ctx context.Context, refundNo string) (domain.RefundRecord, error)
	updateRefundStatusFn         func(ctx context.Context, refundNo string, oldStatus, newStatus domain.RefundStatus, updates map[string]any) error
	sumRefundedAmountByPaymentNoFn func(ctx context.Context, paymentNo string) (int64, error)

	createPaymentCalls []domain.PaymentOrder
	updateStatusCalls  []paymentStatusUpdate
	createRefundCalls  []domain.RefundRecord
	refundStatusCalls  []refundStatusUpdate
}

type paymentStatusUpdate struct {
	paymentNo string
	oldStatus domain.PaymentStatus
	newStatus domain.PaymentStatus
	updates   map[string]any
}

type refundStatusUpdate struct {
	refundNo  string
	oldStatus domain.RefundStatus
	newStatus domain.RefundStatus
	updates   map[string]any
}

func (f *fakePaymentRepo) CreatePayment(ctx context.Context, payment domain.PaymentOrder) (domain.PaymentOrder, error) {
	f.createPaymentCalls = append(f.createPaymentCalls, payment)
	if f.createPaymentFn != nil {
		return f.createPaymentFn(ctx, payment)
	}
	return payment, nil
}

func (f *fakePaymentRepo) FindByPaymentNo(ctx context.Context, paymentNo string) (domain.PaymentOrder, error) {
	if f.findByPaymentNoFn != nil {
		return f.findByPaymentNoFn(ctx, paymentNo)
	}
	return domain.PaymentOrder{}, errors.New("unexpected FindByPaymentNo call")
}

func (f *fakePaymentRepo) FindByOrderNo(ctx context.Context, orderNo string) (domain.PaymentOrder, error) {
	if f.findByOrderNoFn != nil {
		return f.findByOrderNoFn(ctx, orderNo)
	}
	return domain.PaymentOrder{}, errors.New("unexpected FindByOrderNo call")
}

func (f *fakePaymentRepo) ListOpenPaymentsByOrderNo(ctx context.Context, orderNo string) ([]domain.PaymentOrder, error) {
	if f.listOpenPaymentsByOrderNoFn != nil {
		return f.listOpenPaymentsByOrderNoFn(ctx, orderNo)
	}
	if f.findByOrderNoFn != nil {
		payment, err := f.findByOrderNoFn(ctx, orderNo)
		if err != nil {
			return nil, err
		}
		return []domain.PaymentOrder{payment}, nil
	}
	return nil, nil
}

func (f *fakePaymentRepo) UpdateStatus(ctx context.Context, paymentNo string, oldStatus, newStatus domain.PaymentStatus, updates map[string]any) error {
	f.updateStatusCalls = append(f.updateStatusCalls, paymentStatusUpdate{
		paymentNo: paymentNo,
		oldStatus: oldStatus,
		newStatus: newStatus,
		updates:   updates,
	})
	if f.updateStatusFn != nil {
		return f.updateStatusFn(ctx, paymentNo, oldStatus, newStatus, updates)
	}
	return nil
}

func (f *fakePaymentRepo) ListPayments(ctx context.Context, tenantId int64, status, page, pageSize int32) ([]domain.PaymentOrder, int64, error) {
	if f.listPaymentsFn != nil {
		return f.listPaymentsFn(ctx, tenantId, status, page, pageSize)
	}
	return nil, 0, nil
}

func (f *fakePaymentRepo) ListPaymentsByDateAndChannel(ctx context.Context, ch string, startTime, endTime int64) ([]domain.PaymentOrder, error) {
	if f.listPaymentsByDateAndChannelFn != nil {
		return f.listPaymentsByDateAndChannelFn(ctx, ch, startTime, endTime)
	}
	return nil, nil
}

func (f *fakePaymentRepo) CreateRefund(ctx context.Context, refund domain.RefundRecord) error {
	f.createRefundCalls = append(f.createRefundCalls, refund)
	if f.createRefundFn != nil {
		return f.createRefundFn(ctx, refund)
	}
	return nil
}

func (f *fakePaymentRepo) FindRefundByNo(ctx context.Context, refundNo string) (domain.RefundRecord, error) {
	if f.findRefundByNoFn != nil {
		return f.findRefundByNoFn(ctx, refundNo)
	}
	return domain.RefundRecord{}, errors.New("unexpected FindRefundByNo call")
}

func (f *fakePaymentRepo) UpdateRefundStatus(ctx context.Context, refundNo string, oldStatus, newStatus domain.RefundStatus, updates map[string]any) error {
	f.refundStatusCalls = append(f.refundStatusCalls, refundStatusUpdate{
		refundNo:  refundNo,
		oldStatus: oldStatus,
		newStatus: newStatus,
		updates:   updates,
	})
	if f.updateRefundStatusFn != nil {
		return f.updateRefundStatusFn(ctx, refundNo, oldStatus, newStatus, updates)
	}
	return nil
}

func (f *fakePaymentRepo) SumRefundedAmountByPaymentNo(ctx context.Context, paymentNo string) (int64, error) {
	if f.sumRefundedAmountByPaymentNoFn != nil {
		return f.sumRefundedAmountByPaymentNoFn(ctx, paymentNo)
	}
	var total int64
	for _, call := range f.createRefundCalls {
		if call.PaymentNo == paymentNo {
			total += call.Amount
		}
	}
	return total, nil
}

type fakeProducer struct {
	orderPaidEvents []paymentevents.OrderPaidEvent
}

func (f *fakeProducer) ProduceOrderPaid(ctx context.Context, evt paymentevents.OrderPaidEvent) error {
	f.orderPaidEvents = append(f.orderPaidEvents, evt)
	return nil
}

type fakeIdempotencyService struct {
	exists bool
	err    error
}

var _ idempotent.IdempotencyService = (*fakeIdempotencyService)(nil)

func (f *fakeIdempotencyService) Exists(ctx context.Context, key string) (bool, error) {
	return f.exists, f.err
}

func (f *fakeIdempotencyService) MExists(ctx context.Context, keys ...string) ([]bool, error) {
	out := make([]bool, 0, len(keys))
	for range keys {
		out = append(out, f.exists)
	}
	return out, f.err
}

type fakeChannel struct {
	payFn          func(ctx context.Context, payment domain.PaymentOrder) (string, string, error)
	queryPaymentFn func(ctx context.Context, paymentNo string) (int32, string, error)
	refundFn       func(ctx context.Context, refund domain.RefundRecord) (string, error)
	queryRefundFn  func(ctx context.Context, refundNo string) (int32, string, error)
	verifyNotifyFn func(ctx context.Context, data map[string]string) (string, string, error)
	billFn         func(ctx context.Context, billDate string) ([]channel.BillItem, error)

	payCalls    []domain.PaymentOrder
	refundCalls []domain.RefundRecord
}

func (f *fakeChannel) Pay(ctx context.Context, payment domain.PaymentOrder) (string, string, error) {
	f.payCalls = append(f.payCalls, payment)
	if f.payFn != nil {
		return f.payFn(ctx, payment)
	}
	return "", "", nil
}

func (f *fakeChannel) QueryPayment(ctx context.Context, paymentNo string) (int32, string, error) {
	if f.queryPaymentFn != nil {
		return f.queryPaymentFn(ctx, paymentNo)
	}
	return 0, "", nil
}

func (f *fakeChannel) Refund(ctx context.Context, refund domain.RefundRecord) (string, error) {
	f.refundCalls = append(f.refundCalls, refund)
	if f.refundFn != nil {
		return f.refundFn(ctx, refund)
	}
	return "", nil
}

func (f *fakeChannel) QueryRefund(ctx context.Context, refundNo string) (int32, string, error) {
	if f.queryRefundFn != nil {
		return f.queryRefundFn(ctx, refundNo)
	}
	return 0, "", nil
}

func (f *fakeChannel) VerifyNotify(ctx context.Context, data map[string]string) (string, string, error) {
	if f.verifyNotifyFn != nil {
		return f.verifyNotifyFn(ctx, data)
	}
	return "", "", nil
}

func (f *fakeChannel) DownloadBill(ctx context.Context, billDate string) ([]channel.BillItem, error) {
	if f.billFn != nil {
		return f.billFn(ctx, billDate)
	}
	return nil, nil
}

type fakeRefundSyncer struct {
	syncCalls []refundSyncCall
	err       error
}

type refundSyncCall struct {
	payment domain.PaymentOrder
	refund  domain.RefundRecord
}

func (f *fakeRefundSyncer) SyncRefund(ctx context.Context, payment domain.PaymentOrder, refund domain.RefundRecord) error {
	f.syncCalls = append(f.syncCalls, refundSyncCall{
		payment: payment,
		refund:  refund,
	})
	return f.err
}

type fakeReconciliationDAO struct {
	createBatchFn   func(ctx context.Context, batch dao.ReconciliationBatchModel) (dao.ReconciliationBatchModel, error)
	updateBatchFn   func(ctx context.Context, id int64, updates map[string]any) error
	createDetailsFn func(ctx context.Context, details []dao.ReconciliationDetailModel) error

	batchesCreated []dao.ReconciliationBatchModel
	batchUpdates   []batchUpdate
	detailsCreated [][]dao.ReconciliationDetailModel
}

type batchUpdate struct {
	id      int64
	updates map[string]any
}

func (f *fakeReconciliationDAO) CreateBatch(ctx context.Context, batch dao.ReconciliationBatchModel) (dao.ReconciliationBatchModel, error) {
	f.batchesCreated = append(f.batchesCreated, batch)
	if f.createBatchFn != nil {
		return f.createBatchFn(ctx, batch)
	}
	batch.ID = int64(len(f.batchesCreated))
	return batch, nil
}

func (f *fakeReconciliationDAO) UpdateBatch(ctx context.Context, id int64, updates map[string]any) error {
	f.batchUpdates = append(f.batchUpdates, batchUpdate{id: id, updates: updates})
	if f.updateBatchFn != nil {
		return f.updateBatchFn(ctx, id, updates)
	}
	return nil
}

func (f *fakeReconciliationDAO) ListBatches(ctx context.Context, offset, limit int) ([]dao.ReconciliationBatchModel, int64, error) {
	return nil, 0, nil
}

func (f *fakeReconciliationDAO) GetBatch(ctx context.Context, id int64) (dao.ReconciliationBatchModel, error) {
	return dao.ReconciliationBatchModel{}, nil
}

func (f *fakeReconciliationDAO) CreateDetails(ctx context.Context, details []dao.ReconciliationDetailModel) error {
	cloned := append([]dao.ReconciliationDetailModel(nil), details...)
	f.detailsCreated = append(f.detailsCreated, cloned)
	if f.createDetailsFn != nil {
		return f.createDetailsFn(ctx, details)
	}
	return nil
}

func (f *fakeReconciliationDAO) ListDetails(ctx context.Context, batchId int64, offset, limit int) ([]dao.ReconciliationDetailModel, int64, error) {
	return nil, 0, nil
}

func (f *fakeReconciliationDAO) FindBatchByChannelAndDate(ctx context.Context, channel, billDate string) (dao.ReconciliationBatchModel, error) {
	return dao.ReconciliationBatchModel{}, gorm.ErrRecordNotFound
}

func newTestNode() *snowflake.Node {
	node, err := snowflake.NewNode(1)
	if err != nil {
		panic(err)
	}
	return node
}

func newTestLogger() logger.Logger {
	return logger.NewNopLogger()
}

// nonReconcilerChannel implements Channel but NOT Reconciler,
// used to test the "channel does not support reconciliation" code path.
type nonReconcilerChannel struct{}

func (c *nonReconcilerChannel) Pay(ctx context.Context, payment domain.PaymentOrder) (string, string, error) {
	return "", "", nil
}

func (c *nonReconcilerChannel) QueryPayment(ctx context.Context, paymentNo string) (int32, string, error) {
	return 0, "", nil
}

func (c *nonReconcilerChannel) Refund(ctx context.Context, refund domain.RefundRecord) (string, error) {
	return "", nil
}

func (c *nonReconcilerChannel) QueryRefund(ctx context.Context, refundNo string) (int32, string, error) {
	return 0, "", nil
}

func (c *nonReconcilerChannel) VerifyNotify(ctx context.Context, data map[string]string) (string, string, error) {
	return "", "", nil
}
