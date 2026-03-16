package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/rermrf/emo/logger"
	cartv1 "github.com/rermrf/mall/api/proto/gen/cart/v1"
	productv1 "github.com/rermrf/mall/api/proto/gen/product/v1"
	"github.com/rermrf/mall/pkg/ginx"
)

type CartHandler struct {
	cartClient    cartv1.CartServiceClient
	productClient productv1.ProductServiceClient
	l             logger.Logger
}

func NewCartHandler(
	cartClient cartv1.CartServiceClient,
	productClient productv1.ProductServiceClient,
	l logger.Logger,
) *CartHandler {
	return &CartHandler{
		cartClient:    cartClient,
		productClient: productClient,
		l:             l,
	}
}

type AddCartItemReq struct {
	SkuID     int64 `json:"skuId" binding:"required"`
	ProductID int64 `json:"productId" binding:"required"`
	Quantity  int32 `json:"quantity" binding:"required,min=1"`
}

func (h *CartHandler) AddItem(ctx *gin.Context, req AddCartItemReq) (ginx.Result, error) {
	uid, _ := ctx.Get("uid")
	tenantId, _ := ctx.Get("tenant_id")
	_, err := h.cartClient.AddItem(ctx.Request.Context(), &cartv1.AddItemRequest{
		UserId:    uid.(int64),
		SkuId:     req.SkuID,
		ProductId: req.ProductID,
		TenantId:  tenantId.(int64),
		Quantity:  req.Quantity,
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "加入购物车失败")
	}
	return ginx.Result{Code: 0, Msg: "success"}, nil
}

type UpdateCartItemReq struct {
	Quantity       int32 `json:"quantity"`
	Selected       bool  `json:"selected"`
	UpdateSelected bool  `json:"updateSelected"`
}

func (h *CartHandler) UpdateItem(ctx *gin.Context, req UpdateCartItemReq) (ginx.Result, error) {
	uid, _ := ctx.Get("uid")
	skuIdStr := ctx.Param("skuId")
	skuId, err := strconv.ParseInt(skuIdStr, 10, 64)
	if err != nil {
		return ginx.Result{Code: 4, Msg: "无效的 SKU ID"}, nil
	}
	if req.Quantity != 0 && req.Quantity < 1 {
		return ginx.Result{Code: ginx.CodeInvalidQuantity, Msg: "数量必须大于等于1"}, nil
	}
	_, err = h.cartClient.UpdateItem(ctx.Request.Context(), &cartv1.UpdateItemRequest{
		UserId:         uid.(int64),
		SkuId:          skuId,
		Quantity:       req.Quantity,
		Selected:       req.Selected,
		UpdateSelected: req.UpdateSelected,
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "更新购物车失败")
	}
	return ginx.Result{Code: 0, Msg: "success"}, nil
}

func (h *CartHandler) RemoveItem(ctx *gin.Context) {
	uid, _ := ctx.Get("uid")
	skuIdStr := ctx.Param("skuId")
	skuId, err := strconv.ParseInt(skuIdStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusOK, ginx.Result{Code: 4, Msg: "无效的 SKU ID"})
		return
	}
	_, err = h.cartClient.RemoveItem(ctx.Request.Context(), &cartv1.RemoveItemRequest{
		UserId: uid.(int64),
		SkuId:  skuId,
	})
	if err != nil {
		h.l.Error("删除购物车商品失败", logger.Error(err))
		result, _ := ginx.HandleRawError(err)
		ctx.JSON(http.StatusOK, result)
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success"})
}

type CartItemVO struct {
	SkuID        int64  `json:"skuId"`
	ProductID    int64  `json:"productId"`
	Quantity     int32  `json:"quantity"`
	Selected     bool   `json:"selected"`
	ProductName  string `json:"productName"`
	ProductImage string `json:"productImage"`
	SkuSpec      string `json:"skuSpec"`
	Price        int64  `json:"price"`
	Stock        int32  `json:"stock"`
}

func (h *CartHandler) GetCart(ctx *gin.Context) {
	uid, _ := ctx.Get("uid")
	// 1. 获取购物车基础数据
	cartResp, err := h.cartClient.GetCart(ctx.Request.Context(), &cartv1.GetCartRequest{
		UserId: uid.(int64),
	})
	if err != nil {
		h.l.Error("获取购物车失败", logger.Error(err))
		result, _ := ginx.HandleRawError(err)
		ctx.JSON(http.StatusOK, result)
		return
	}
	items := cartResp.GetItems()
	if len(items) == 0 {
		ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: []CartItemVO{}})
		return
	}
	// 2. 收集 productIds，批量查询商品信息
	productIdSet := make(map[int64]struct{})
	for _, item := range items {
		productIdSet[item.GetProductId()] = struct{}{}
	}
	productIds := make([]int64, 0, len(productIdSet))
	for id := range productIdSet {
		productIds = append(productIds, id)
	}
	productResp, err := h.productClient.BatchGetProducts(ctx.Request.Context(), &productv1.BatchGetProductsRequest{
		Ids: productIds,
	})
	if err != nil {
		h.l.Error("批量查询商品信息失败", logger.Error(err))
		// 降级：返回不含商品详情的购物车
		vos := make([]CartItemVO, 0, len(items))
		for _, item := range items {
			vos = append(vos, CartItemVO{
				SkuID:     item.GetSkuId(),
				ProductID: item.GetProductId(),
				Quantity:  item.GetQuantity(),
				Selected:  item.GetSelected(),
			})
		}
		ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: vos})
		return
	}
	// 3. 构建 productId → Product 映射，skuId → SKU 映射
	productMap := make(map[int64]*productv1.Product)
	skuMap := make(map[int64]*productv1.ProductSKU)
	for _, p := range productResp.GetProducts() {
		productMap[p.GetId()] = p
		for _, sku := range p.GetSkus() {
			skuMap[sku.GetId()] = sku
		}
	}
	// 4. 聚合返回
	vos := make([]CartItemVO, 0, len(items))
	for _, item := range items {
		vo := CartItemVO{
			SkuID:     item.GetSkuId(),
			ProductID: item.GetProductId(),
			Quantity:  item.GetQuantity(),
			Selected:  item.GetSelected(),
		}
		if p, ok := productMap[item.GetProductId()]; ok {
			vo.ProductName = p.GetName()
			vo.ProductImage = p.GetMainImage()
		}
		if sku, ok := skuMap[item.GetSkuId()]; ok {
			vo.SkuSpec = sku.GetSpecValues()
			vo.Price = sku.GetPrice()
		}
		vos = append(vos, vo)
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: vos})
}

func (h *CartHandler) ClearCart(ctx *gin.Context) {
	uid, _ := ctx.Get("uid")
	_, err := h.cartClient.ClearCart(ctx.Request.Context(), &cartv1.ClearCartRequest{
		UserId: uid.(int64),
	})
	if err != nil {
		h.l.Error("清空购物车失败", logger.Error(err))
		result, _ := ginx.HandleRawError(err)
		ctx.JSON(http.StatusOK, result)
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success"})
}

type BatchRemoveReq struct {
	SkuIDs []int64 `json:"skuIds" binding:"required,min=1"`
}

func (h *CartHandler) BatchRemove(ctx *gin.Context, req BatchRemoveReq) (ginx.Result, error) {
	uid, _ := ctx.Get("uid")
	_, err := h.cartClient.BatchRemoveItems(ctx.Request.Context(), &cartv1.BatchRemoveItemsRequest{
		UserId: uid.(int64),
		SkuIds: req.SkuIDs,
	})
	if err != nil {
		return ginx.HandleGRPCError(err, "批量删除购物车商品失败")
	}
	return ginx.Result{Code: 0, Msg: "success"}, nil
}
