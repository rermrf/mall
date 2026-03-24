package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rermrf/mall/payment/domain"
	"github.com/rermrf/mall/payment/service/channel"
)

func TestCreatePayment(t *testing.T) {
	testCases := []struct {
		name       string
		repo       *fakePaymentRepo
		channels   map[string]channel.Channel
		channel    string
		assertions func(t *testing.T, paymentNo, payURL string, err error, repo *fakePaymentRepo, channels map[string]channel.Channel)
	}{
		{
			name: "同渠道重进应复用支付单并重新生成支付链接",
			repo: &fakePaymentRepo{
				findByOrderNoFn: func(ctx context.Context, orderNo string) (domain.PaymentOrder, error) {
					return domain.PaymentOrder{
						ID:        1,
						PaymentNo: "P_EXIST",
						OrderNo:   orderNo,
						Channel:   "wechat",
						Amount:    1999,
						Status:    domain.PaymentStatusPending,
					}, nil
				},
			},
			channels: map[string]channel.Channel{
				"wechat": &fakeChannel{
					payFn: func(ctx context.Context, payment domain.PaymentOrder) (string, string, error) {
						return "", "https://pay/reuse", nil
					},
				},
			},
			channel: "wechat",
			assertions: func(t *testing.T, paymentNo, payURL string, err error, repo *fakePaymentRepo, channels map[string]channel.Channel) {
				t.Helper()
				ch := channels["wechat"].(*fakeChannel)
				assert.NoError(t, err)
				assert.Equal(t, "P_EXIST", paymentNo)
				assert.Equal(t, "https://pay/reuse", payURL)
				assert.Len(t, ch.payCalls, 1)
				assert.Empty(t, repo.createPaymentCalls)
			},
		},
		{
			name: "切换渠道应关闭旧支付单并创建新支付单",
			repo: &fakePaymentRepo{
				findByOrderNoFn: func(ctx context.Context, orderNo string) (domain.PaymentOrder, error) {
					return domain.PaymentOrder{
						ID:        1,
						PaymentNo: "P_OLD",
						OrderNo:   orderNo,
						Channel:   "wechat",
						Amount:    2999,
						Status:    domain.PaymentStatusPending,
					}, nil
				},
				createPaymentFn: func(ctx context.Context, payment domain.PaymentOrder) (domain.PaymentOrder, error) {
					payment.ID = 2
					return payment, nil
				},
			},
			channels: map[string]channel.Channel{
				"alipay": &fakeChannel{
					payFn: func(ctx context.Context, payment domain.PaymentOrder) (string, string, error) {
						return "", "https://pay/new-alipay", nil
					},
				},
				"wechat": &fakeChannel{},
			},
			channel: "alipay",
			assertions: func(t *testing.T, paymentNo, payURL string, err error, repo *fakePaymentRepo, channels map[string]channel.Channel) {
				t.Helper()
				assert.NoError(t, err)
				assert.NotEqual(t, "P_OLD", paymentNo)
				assert.Equal(t, "https://pay/new-alipay", payURL)
				assert.Len(t, repo.createPaymentCalls, 1)
				assert.Len(t, repo.updateStatusCalls, 1)
				if assert.Len(t, repo.updateStatusCalls, 1) {
					assert.Equal(t, "P_OLD", repo.updateStatusCalls[0].paymentNo)
					assert.Equal(t, domain.PaymentStatusClosed, repo.updateStatusCalls[0].newStatus)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &paymentService{
				repo:           tc.repo,
				producer:       &fakeProducer{},
				idempotencySvc: &fakeIdempotencyService{},
				node:           newTestNode(),
				channels:       tc.channels,
				l:              newTestLogger(),
			}

			paymentNo, payURL, err := svc.CreatePayment(context.Background(), 1, 10, "O20260324", tc.channel, 1999)

			tc.assertions(t, paymentNo, payURL, err, tc.repo, tc.channels)
		})
	}
}

func TestRefund(t *testing.T) {
	testCases := []struct {
		name       string
		amount     int64
		repo       *fakePaymentRepo
		channels   map[string]channel.Channel
		assertions func(t *testing.T, refundNo string, err error, repo *fakePaymentRepo, syncer *fakeRefundSyncer)
	}{
		{
			name:   "部分退款成功后支付单应保持已支付",
			amount: 300,
			repo: &fakePaymentRepo{
				findByPaymentNoFn: func(ctx context.Context, paymentNo string) (domain.PaymentOrder, error) {
					return domain.PaymentOrder{
						ID:        1,
						TenantID:  11,
						PaymentNo: paymentNo,
						OrderNo:   "O20260324",
						Channel:   "mock",
						Amount:    1000,
						Status:    domain.PaymentStatusPaid,
					}, nil
				},
			},
			channels: map[string]channel.Channel{
				"mock": &fakeChannel{
					refundFn: func(ctx context.Context, refund domain.RefundRecord) (string, error) {
						return "MOCK_REFUND_1", nil
					},
				},
			},
			assertions: func(t *testing.T, refundNo string, err error, repo *fakePaymentRepo, syncer *fakeRefundSyncer) {
				t.Helper()
				assert.NoError(t, err)
				assert.NotEmpty(t, refundNo)
				assert.Len(t, repo.createRefundCalls, 1)
				assert.Len(t, repo.refundStatusCalls, 1)
				assert.Len(t, repo.updateStatusCalls, 0)
				if assert.Len(t, syncer.syncCalls, 1) {
					assert.Equal(t, "O20260324", syncer.syncCalls[0].payment.OrderNo)
					assert.Equal(t, int64(300), syncer.syncCalls[0].refund.Amount)
				}
			},
		},
		{
			name:   "全额退款成功后支付单应更新为已退款",
			amount: 1000,
			repo: &fakePaymentRepo{
				findByPaymentNoFn: func(ctx context.Context, paymentNo string) (domain.PaymentOrder, error) {
					return domain.PaymentOrder{
						ID:        1,
						TenantID:  11,
						PaymentNo: paymentNo,
						OrderNo:   "O20260324",
						Channel:   "mock",
						Amount:    1000,
						Status:    domain.PaymentStatusPaid,
					}, nil
				},
			},
			channels: map[string]channel.Channel{
				"mock": &fakeChannel{
					refundFn: func(ctx context.Context, refund domain.RefundRecord) (string, error) {
						return "MOCK_REFUND_2", nil
					},
				},
			},
			assertions: func(t *testing.T, refundNo string, err error, repo *fakePaymentRepo, syncer *fakeRefundSyncer) {
				t.Helper()
				assert.NoError(t, err)
				assert.NotEmpty(t, refundNo)
				assert.Len(t, repo.updateStatusCalls, 1)
				assert.Equal(t, domain.PaymentStatusRefunded, repo.updateStatusCalls[0].newStatus)
				assert.Len(t, syncer.syncCalls, 1)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			syncer := &fakeRefundSyncer{}
			svc := &paymentService{
				repo:           tc.repo,
				producer:       &fakeProducer{},
				idempotencySvc: &fakeIdempotencyService{},
				node:           newTestNode(),
				refundSyncer:   syncer,
				channels:       tc.channels,
				l:              newTestLogger(),
			}

			refundNo, err := svc.Refund(context.Background(), "P20260324", tc.amount, "buyer-request")

			tc.assertions(t, refundNo, err, tc.repo, syncer)
		})
	}
}

func TestCloseOrderPayments(t *testing.T) {
	repo := &fakePaymentRepo{
		listOpenPaymentsByOrderNoFn: func(ctx context.Context, orderNo string) ([]domain.PaymentOrder, error) {
			return []domain.PaymentOrder{
				{PaymentNo: "P1", OrderNo: orderNo, Status: domain.PaymentStatusPending},
				{PaymentNo: "P2", OrderNo: orderNo, Status: domain.PaymentStatusPaying},
			}, nil
		},
	}
	svc := &paymentService{
		repo:           repo,
		producer:       &fakeProducer{},
		idempotencySvc: &fakeIdempotencyService{},
		node:           newTestNode(),
		l:              newTestLogger(),
	}

	err := svc.CloseOrderPayments(context.Background(), "O20260324")

	assert.NoError(t, err)
	if assert.Len(t, repo.updateStatusCalls, 2) {
		assert.Equal(t, "P1", repo.updateStatusCalls[0].paymentNo)
		assert.Equal(t, domain.PaymentStatusClosed, repo.updateStatusCalls[0].newStatus)
		assert.Equal(t, "P2", repo.updateStatusCalls[1].paymentNo)
		assert.Equal(t, domain.PaymentStatusClosed, repo.updateStatusCalls[1].newStatus)
	}
}
