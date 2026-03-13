package handler

import (
	"net/http"
	"strconv"

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

// ==================== 运费模板 ====================

type CreateFreightTemplateReq struct {
	Name          string           `json:"name" binding:"required"`
	ChargeType    int32            `json:"charge_type" binding:"required"`
	FreeThreshold int64            `json:"free_threshold"`
	Rules         []FreightRuleReq `json:"rules" binding:"required,min=1"`
}

type FreightRuleReq struct {
	Regions         string `json:"regions" binding:"required"`
	FirstUnit       int32  `json:"first_unit" binding:"required"`
	FirstPrice      int64  `json:"first_price" binding:"required"`
	AdditionalUnit  int32  `json:"additional_unit" binding:"required"`
	AdditionalPrice int64  `json:"additional_price" binding:"required"`
}

func (h *LogisticsHandler) CreateFreightTemplate(ctx *gin.Context, req CreateFreightTemplateReq) (ginx.Result, error) {
	tenantId, _ := ctx.Get("tenant_id")
	rules := make([]*logisticsv1.FreightRule, 0, len(req.Rules))
	for _, r := range req.Rules {
		rules = append(rules, &logisticsv1.FreightRule{
			Regions: r.Regions, FirstUnit: r.FirstUnit, FirstPrice: r.FirstPrice,
			AdditionalUnit: r.AdditionalUnit, AdditionalPrice: r.AdditionalPrice,
		})
	}
	resp, err := h.logisticsClient.CreateFreightTemplate(ctx.Request.Context(), &logisticsv1.CreateFreightTemplateRequest{
		Template: &logisticsv1.FreightTemplate{
			TenantId: tenantId.(int64), Name: req.Name,
			ChargeType: req.ChargeType, FreeThreshold: req.FreeThreshold,
			Rules: rules,
		},
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "创建运费模板失败")
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{"id": resp.GetId()}}, nil
}

type UpdateFreightTemplateReq struct {
	Name          string           `json:"name"`
	ChargeType    int32            `json:"charge_type"`
	FreeThreshold int64            `json:"free_threshold"`
	Rules         []FreightRuleReq `json:"rules"`
}

func (h *LogisticsHandler) UpdateFreightTemplate(ctx *gin.Context, req UpdateFreightTemplateReq) (ginx.Result, error) {
	tenantId, _ := ctx.Get("tenant_id")
	idStr := ctx.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	rules := make([]*logisticsv1.FreightRule, 0, len(req.Rules))
	for _, r := range req.Rules {
		rules = append(rules, &logisticsv1.FreightRule{
			Regions: r.Regions, FirstUnit: r.FirstUnit, FirstPrice: r.FirstPrice,
			AdditionalUnit: r.AdditionalUnit, AdditionalPrice: r.AdditionalPrice,
		})
	}
	_, err := h.logisticsClient.UpdateFreightTemplate(ctx.Request.Context(), &logisticsv1.UpdateFreightTemplateRequest{
		Template: &logisticsv1.FreightTemplate{
			Id: id, TenantId: tenantId.(int64), Name: req.Name,
			ChargeType: req.ChargeType, FreeThreshold: req.FreeThreshold,
			Rules: rules,
		},
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "更新运费模板失败")
	}
	return ginx.Result{Code: 0, Msg: "success"}, nil
}

func (h *LogisticsHandler) GetFreightTemplate(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	resp, err := h.logisticsClient.GetFreightTemplate(ctx.Request.Context(), &logisticsv1.GetFreightTemplateRequest{Id: id})
	if err != nil {
		h.l.Error("查询运费模板详情失败", logger.Error(err))
		result, _ := ginx.HandleRawError(err)
		ctx.JSON(http.StatusOK, result)
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetTemplate()})
}

func (h *LogisticsHandler) ListFreightTemplates(ctx *gin.Context) {
	tenantId, _ := ctx.Get("tenant_id")
	resp, err := h.logisticsClient.ListFreightTemplates(ctx.Request.Context(), &logisticsv1.ListFreightTemplatesRequest{
		TenantId: tenantId.(int64),
	})
	if err != nil {
		h.l.Error("查询运费模板列表失败", logger.Error(err))
		result, _ := ginx.HandleRawError(err)
		ctx.JSON(http.StatusOK, result)
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetTemplates()})
}

func (h *LogisticsHandler) DeleteFreightTemplate(ctx *gin.Context) {
	tenantId, _ := ctx.Get("tenant_id")
	idStr := ctx.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	_, err := h.logisticsClient.DeleteFreightTemplate(ctx.Request.Context(), &logisticsv1.DeleteFreightTemplateRequest{
		Id: id, TenantId: tenantId.(int64),
	})
	if err != nil {
		h.l.Error("删除运费模板失败", logger.Error(err))
		result, _ := ginx.HandleRawError(err)
		ctx.JSON(http.StatusOK, result)
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success"})
}

// ==================== 发货（聚合端点） ====================

type ShipOrderReq struct {
	CarrierCode string `json:"carrier_code" binding:"required"`
	CarrierName string `json:"carrier_name" binding:"required"`
	TrackingNo  string `json:"tracking_no" binding:"required"`
}

func (h *LogisticsHandler) ShipOrder(ctx *gin.Context, req ShipOrderReq) (ginx.Result, error) {
	tenantId, _ := ctx.Get("tenant_id")
	uid, _ := ctx.Get("uid")
	orderNo := ctx.Param("orderNo")

	// 1. 通过 order-svc 获取订单信息（取 order_id）
	orderResp, err := h.orderClient.GetOrder(ctx.Request.Context(), &orderv1.GetOrderRequest{OrderNo: orderNo})
	if err != nil {
		return ginx.HandleGRPCError(err, "查询订单失败")
	}

	// 2. 创建物流单
	_, err = h.logisticsClient.CreateShipment(ctx.Request.Context(), &logisticsv1.CreateShipmentRequest{
		TenantId:    tenantId.(int64),
		OrderId:     orderResp.GetOrder().GetId(),
		CarrierCode: req.CarrierCode,
		CarrierName: req.CarrierName,
		TrackingNo:  req.TrackingNo,
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "创建物流单失败")
	}

	// 3. 更新订单状态为已发货
	_, err = h.orderClient.UpdateOrderStatus(ctx.Request.Context(), &orderv1.UpdateOrderStatusRequest{
		OrderNo:      orderNo,
		Status:       3, // shipped
		OperatorId:   uid.(int64),
		OperatorType: 2, // 商家
		Remark:       "商家发货",
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "更新订单状态失败")
	}

	return ginx.Result{Code: 0, Msg: "success"}, nil
}

// ==================== 查物流 ====================

func (h *LogisticsHandler) GetOrderLogistics(ctx *gin.Context) {
	orderNo := ctx.Param("orderNo")

	// 通过 order-svc 获取 order_id
	orderResp, err := h.orderClient.GetOrder(ctx.Request.Context(), &orderv1.GetOrderRequest{OrderNo: orderNo})
	if err != nil {
		h.l.Error("查询订单失败", logger.Error(err))
		result, _ := ginx.HandleRawError(err)
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
