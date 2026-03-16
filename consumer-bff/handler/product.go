package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rermrf/emo/logger"
	productv1 "github.com/rermrf/mall/api/proto/gen/product/v1"
	"github.com/rermrf/mall/pkg/ginx"
)

type ProductHandler struct {
	productClient productv1.ProductServiceClient
	l             logger.Logger
}

func NewProductHandler(productClient productv1.ProductServiceClient, l logger.Logger) *ProductHandler {
	return &ProductHandler{productClient: productClient, l: l}
}

func (h *ProductHandler) GetProduct(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusOK, ginx.Result{Code: 4, Msg: "无效的商品ID"})
		return
	}
	resp, err := h.productClient.GetProduct(ctx.Request.Context(), &productv1.GetProductRequest{
		Id: id,
	})
	if err != nil {
		h.l.Error("查询商品详情失败", logger.Error(err))
		result, _ := ginx.HandleRawError(err)
		ctx.JSON(http.StatusOK, result)
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetProduct()})
}
