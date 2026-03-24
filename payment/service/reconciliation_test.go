package service

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rermrf/mall/payment/domain"
	"github.com/rermrf/mall/payment/service/channel"
)

func TestRunReconciliation_TracksMissingChannelTradeNoAsMismatch(t *testing.T) {
	repo := &fakePaymentRepo{
		listPaymentsByDateAndChannelFn: func(ctx context.Context, ch string, startTime, endTime int64) ([]domain.PaymentOrder, error) {
			return []domain.PaymentOrder{
				{
					ID:             1,
					TenantID:       11,
					PaymentNo:      "P_LOCAL_1",
					OrderNo:        "O20260324",
					Channel:        ch,
					Amount:         1000,
					Status:         domain.PaymentStatusPaid,
					ChannelTradeNo: "",
				},
			}, nil
		},
	}
	reconDAO := &fakeReconciliationDAO{}
	reconChannel := &fakeChannel{
		billFn: func(ctx context.Context, billDate string) ([]channel.BillItem, error) {
			return nil, nil
		},
	}
	svc := &reconciliationService{
		reconDAO: reconDAO,
		repo:     repo,
		channels: map[string]channel.Channel{"mock": reconChannel},
		node:     newTestNode(),
		l:        newTestLogger(),
	}

	batchID, err := svc.RunReconciliation(context.Background(), "mock", "2026-03-23")

	assert.NoError(t, err)
	assert.Equal(t, int64(1), batchID)
	assert.Len(t, reconDAO.detailsCreated, 1)
	if assert.Len(t, reconDAO.detailsCreated, 1) && assert.Len(t, reconDAO.detailsCreated[0], 1) {
		assert.Equal(t, "P_LOCAL_1", reconDAO.detailsCreated[0][0].PaymentNo)
		assert.True(t, strings.Contains(reconDAO.detailsCreated[0][0].Remark, "渠道交易号"))
	}

	if assert.NotEmpty(t, reconDAO.batchUpdates) {
		last := reconDAO.batchUpdates[len(reconDAO.batchUpdates)-1].updates
		assert.Equal(t, int32(1), last["total_local"])
		assert.Equal(t, int32(1), last["total_mismatch"])
	}
}

func TestRunReconciliation_RejectsUnavailableChannelClearly(t *testing.T) {
	reconDAO := &fakeReconciliationDAO{}
	svc := &reconciliationService{
		reconDAO: reconDAO,
		repo:     &fakePaymentRepo{},
		channels: map[string]channel.Channel{"wechat": &fakeChannel{}},
		node:     newTestNode(),
		l:        newTestLogger(),
	}

	batchID, err := svc.RunReconciliation(context.Background(), "wechat", "2026-03-23")

	assert.Error(t, err)
	assert.Equal(t, int64(1), batchID)
	assert.Contains(t, err.Error(), "对账暂未实现")
	if assert.NotEmpty(t, reconDAO.batchUpdates) {
		last := reconDAO.batchUpdates[len(reconDAO.batchUpdates)-1].updates
		assert.EqualValues(t, 3, last["status"])
		assert.Contains(t, last["error_msg"], "对账暂未实现")
	}
}
