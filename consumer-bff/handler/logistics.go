package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rermrf/emo/logger"
	logisticsv1 "github.com/rermrf/mall/api/proto/gen/logistics/v1"
	orderv1 "github.com/rermrf/mall/api/proto/gen/order/v1"
	"github.com/rermrf/mall/pkg/ginx"
)

type LogisticsHandler struct {
	logisticsClient logisticsv1.LogisticsServiceClient
	orderClient     orderv1.OrderServiceClient
	l               logger.Logger
}

func NewLogisticsHandler(
	logisticsClient logisticsv1.LogisticsServiceClient,
	orderClient orderv1.OrderServiceClient,
	l logger.Logger,
) *LogisticsHandler {
	return &LogisticsHandler{
		logisticsClient: logisticsClient,
		orderClient:     orderClient,
		l:               l,
	}
}

// GetOrderLogistics 查询订单物流（需登录）
func (h *LogisticsHandler) GetOrderLogistics(ctx *gin.Context) {
	orderNo := ctx.Param("orderNo")

	// 通过 order-svc 获取 order_id
	orderResp, err := h.orderClient.GetOrder(ctx.Request.Context(), &orderv1.GetOrderRequest{OrderNo: orderNo})
	if err != nil {
		h.l.Error("查询订单失败", logger.Error(err))
		result, _ := ginx.HandleRawError(err, ginx.OrderErrMappings...)
		ctx.JSON(http.StatusOK, result)
		return
	}

	resp, err := h.logisticsClient.GetShipmentByOrder(ctx.Request.Context(), &logisticsv1.GetShipmentByOrderRequest{
		OrderId: orderResp.GetOrder().GetId(),
	})
	if err != nil {
		h.l.Error("查询物流信息失败", logger.Error(err))
		result, _ := ginx.HandleRawError(err)
		ctx.JSON(http.StatusOK, result)
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetShipment()})
}
