# Cross-Service Integration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix all missing cross-service connections: merchant-bff product integration, admin-bff 4 service integrations, 3 unconsumed Kafka topics, and notification/marketing consumer handler fixes.

**Architecture:** BFF handlers follow the established pattern (struct with gRPC client + logger, request structs with form/json tags, `ginx.WrapBody`/`ginx.WrapQuery` wrappers, `fmt.Errorf` for errors). Kafka consumers follow the existing pattern (consumer struct with handler func, `saramax.NewHandler[T]` generic handler, constructors in `ioc/kafka.go`). All DI via Wire.

**Tech Stack:** Go, Gin, gRPC, Kafka (sarama), Wire DI, etcd service discovery

---

## Task 1: merchant-bff — Product Handler (11 endpoints)

**Files:**
- Create: `merchant-bff/handler/product.go`
- Modify: `merchant-bff/ioc/grpc.go` — add `InitProductClient`
- Modify: `merchant-bff/ioc/gin.go` — add product routes + handler param
- Modify: `merchant-bff/wire.go` — add ProductClient + ProductHandler
- Regenerate: `merchant-bff/wire_gen.go`

### Step 1: Create `merchant-bff/handler/product.go`

```go
package handler

import (
	"fmt"
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
	return &ProductHandler{
		productClient: productClient,
		l:             l,
	}
}

// ==================== 商品 ====================

type CreateProductReq struct {
	CategoryId  int64            `json:"category_id" binding:"required"`
	BrandId     int64            `json:"brand_id"`
	Name        string           `json:"name" binding:"required"`
	Subtitle    string           `json:"subtitle"`
	MainImage   string           `json:"main_image"`
	Images      string           `json:"images"`
	Description string           `json:"description"`
	Status      int32            `json:"status"`
	Skus        []ProductSKUReq  `json:"skus"`
	Specs       []ProductSpecReq `json:"specs"`
}

type ProductSKUReq struct {
	SkuCode    string `json:"sku_code"`
	Price      int64  `json:"price"`
	Stock      int32  `json:"stock"`
	Attributes string `json:"attributes"`
}

type ProductSpecReq struct {
	Name   string `json:"name"`
	Values string `json:"values"`
}

func (h *ProductHandler) CreateProduct(ctx *gin.Context, req CreateProductReq) (ginx.Result, error) {
	tenantId, _ := ctx.Get("tenant_id")
	skus := make([]*productv1.ProductSKU, 0, len(req.Skus))
	for _, s := range req.Skus {
		skus = append(skus, &productv1.ProductSKU{
			SkuCode:    s.SkuCode,
			Price:      s.Price,
			Stock:      s.Stock,
			Attributes: s.Attributes,
		})
	}
	specs := make([]*productv1.ProductSpec, 0, len(req.Specs))
	for _, s := range req.Specs {
		specs = append(specs, &productv1.ProductSpec{
			Name:   s.Name,
			Values: s.Values,
		})
	}
	resp, err := h.productClient.CreateProduct(ctx.Request.Context(), &productv1.CreateProductRequest{
		Product: &productv1.Product{
			TenantId:    tenantId.(int64),
			CategoryId:  req.CategoryId,
			BrandId:     req.BrandId,
			Name:        req.Name,
			Subtitle:    req.Subtitle,
			MainImage:   req.MainImage,
			Images:      req.Images,
			Description: req.Description,
			Status:      req.Status,
			Skus:        skus,
			Specs:       specs,
		},
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("创建商品失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{"id": resp.GetId()}}, nil
}

type UpdateProductReq struct {
	CategoryId  int64            `json:"category_id"`
	BrandId     int64            `json:"brand_id"`
	Name        string           `json:"name"`
	Subtitle    string           `json:"subtitle"`
	MainImage   string           `json:"main_image"`
	Images      string           `json:"images"`
	Description string           `json:"description"`
	Skus        []ProductSKUReq  `json:"skus"`
	Specs       []ProductSpecReq `json:"specs"`
}

func (h *ProductHandler) UpdateProduct(ctx *gin.Context, req UpdateProductReq) (ginx.Result, error) {
	tenantId, _ := ctx.Get("tenant_id")
	idStr := ctx.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	skus := make([]*productv1.ProductSKU, 0, len(req.Skus))
	for _, s := range req.Skus {
		skus = append(skus, &productv1.ProductSKU{
			SkuCode:    s.SkuCode,
			Price:      s.Price,
			Stock:      s.Stock,
			Attributes: s.Attributes,
		})
	}
	specs := make([]*productv1.ProductSpec, 0, len(req.Specs))
	for _, s := range req.Specs {
		specs = append(specs, &productv1.ProductSpec{
			Name:   s.Name,
			Values: s.Values,
		})
	}
	_, err := h.productClient.UpdateProduct(ctx.Request.Context(), &productv1.UpdateProductRequest{
		Product: &productv1.Product{
			Id:          id,
			TenantId:    tenantId.(int64),
			CategoryId:  req.CategoryId,
			BrandId:     req.BrandId,
			Name:        req.Name,
			Subtitle:    req.Subtitle,
			MainImage:   req.MainImage,
			Images:      req.Images,
			Description: req.Description,
			Skus:        skus,
			Specs:       specs,
		},
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("更新商品失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success"}, nil
}

func (h *ProductHandler) GetProduct(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	resp, err := h.productClient.GetProduct(ctx.Request.Context(), &productv1.GetProductRequest{Id: id})
	if err != nil {
		h.l.Error("查询商品详情失败", logger.Error(err))
		ctx.JSON(http.StatusOK, ginx.Result{Code: 5, Msg: "系统错误"})
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetProduct()})
}

type ListProductsReq struct {
	CategoryId int64 `form:"category_id"`
	Status     int32 `form:"status"`
	Page       int32 `form:"page" binding:"required,min=1"`
	PageSize   int32 `form:"page_size" binding:"required,min=1,max=100"`
}

func (h *ProductHandler) ListProducts(ctx *gin.Context, req ListProductsReq) (ginx.Result, error) {
	tenantId, _ := ctx.Get("tenant_id")
	resp, err := h.productClient.ListProducts(ctx.Request.Context(), &productv1.ListProductsRequest{
		TenantId:   tenantId.(int64),
		CategoryId: req.CategoryId,
		Status:     req.Status,
		Page:       req.Page,
		PageSize:   req.PageSize,
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("查询商品列表失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{
		"products": resp.GetProducts(),
		"total":    resp.GetTotal(),
	}}, nil
}

type UpdateProductStatusReq struct {
	Status int32 `json:"status" binding:"required"`
}

func (h *ProductHandler) UpdateProductStatus(ctx *gin.Context, req UpdateProductStatusReq) (ginx.Result, error) {
	tenantId, _ := ctx.Get("tenant_id")
	idStr := ctx.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	_, err := h.productClient.UpdateProductStatus(ctx.Request.Context(), &productv1.UpdateProductStatusRequest{
		Id:       id,
		TenantId: tenantId.(int64),
		Status:   req.Status,
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("更新商品状态失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success"}, nil
}

// ==================== 分类 ====================

type CreateCategoryReq struct {
	ParentId int64  `json:"parent_id"`
	Name     string `json:"name" binding:"required"`
	Level    int32  `json:"level"`
	Sort     int32  `json:"sort"`
	Icon     string `json:"icon"`
	Status   int32  `json:"status"`
}

func (h *ProductHandler) CreateCategory(ctx *gin.Context, req CreateCategoryReq) (ginx.Result, error) {
	tenantId, _ := ctx.Get("tenant_id")
	resp, err := h.productClient.CreateCategory(ctx.Request.Context(), &productv1.CreateCategoryRequest{
		Category: &productv1.Category{
			TenantId: tenantId.(int64),
			ParentId: req.ParentId,
			Name:     req.Name,
			Level:    req.Level,
			Sort:     req.Sort,
			Icon:     req.Icon,
			Status:   req.Status,
		},
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("创建分类失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{"id": resp.GetId()}}, nil
}

type UpdateCategoryReq struct {
	ParentId int64  `json:"parent_id"`
	Name     string `json:"name"`
	Level    int32  `json:"level"`
	Sort     int32  `json:"sort"`
	Icon     string `json:"icon"`
	Status   int32  `json:"status"`
}

func (h *ProductHandler) UpdateCategory(ctx *gin.Context, req UpdateCategoryReq) (ginx.Result, error) {
	tenantId, _ := ctx.Get("tenant_id")
	idStr := ctx.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	_, err := h.productClient.UpdateCategory(ctx.Request.Context(), &productv1.UpdateCategoryRequest{
		Category: &productv1.Category{
			Id:       id,
			TenantId: tenantId.(int64),
			ParentId: req.ParentId,
			Name:     req.Name,
			Level:    req.Level,
			Sort:     req.Sort,
			Icon:     req.Icon,
			Status:   req.Status,
		},
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("更新分类失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success"}, nil
}

type ListCategoriesReq struct{}

func (h *ProductHandler) ListCategories(ctx *gin.Context, _ ListCategoriesReq) (ginx.Result, error) {
	tenantId, _ := ctx.Get("tenant_id")
	resp, err := h.productClient.ListCategories(ctx.Request.Context(), &productv1.ListCategoriesRequest{
		TenantId: tenantId.(int64),
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("查询分类列表失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: resp.GetCategories()}, nil
}

// ==================== 品牌 ====================

type CreateBrandReq struct {
	Name   string `json:"name" binding:"required"`
	Logo   string `json:"logo"`
	Status int32  `json:"status"`
}

func (h *ProductHandler) CreateBrand(ctx *gin.Context, req CreateBrandReq) (ginx.Result, error) {
	tenantId, _ := ctx.Get("tenant_id")
	resp, err := h.productClient.CreateBrand(ctx.Request.Context(), &productv1.CreateBrandRequest{
		Brand: &productv1.Brand{
			TenantId: tenantId.(int64),
			Name:     req.Name,
			Logo:     req.Logo,
			Status:   req.Status,
		},
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("创建品牌失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{"id": resp.GetId()}}, nil
}

type UpdateBrandReq struct {
	Name   string `json:"name"`
	Logo   string `json:"logo"`
	Status int32  `json:"status"`
}

func (h *ProductHandler) UpdateBrand(ctx *gin.Context, req UpdateBrandReq) (ginx.Result, error) {
	tenantId, _ := ctx.Get("tenant_id")
	idStr := ctx.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	_, err := h.productClient.UpdateBrand(ctx.Request.Context(), &productv1.UpdateBrandRequest{
		Brand: &productv1.Brand{
			Id:       id,
			TenantId: tenantId.(int64),
			Name:     req.Name,
			Logo:     req.Logo,
			Status:   req.Status,
		},
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("更新品牌失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success"}, nil
}

type ListBrandsReq struct {
	Page     int32 `form:"page" binding:"required,min=1"`
	PageSize int32 `form:"page_size" binding:"required,min=1,max=100"`
}

func (h *ProductHandler) ListBrands(ctx *gin.Context, req ListBrandsReq) (ginx.Result, error) {
	tenantId, _ := ctx.Get("tenant_id")
	resp, err := h.productClient.ListBrands(ctx.Request.Context(), &productv1.ListBrandsRequest{
		TenantId: tenantId.(int64),
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("查询品牌列表失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{
		"brands": resp.GetBrands(),
		"total":  resp.GetTotal(),
	}}, nil
}
```

### Step 2: Add `InitProductClient` to `merchant-bff/ioc/grpc.go`

The merchant-bff already has a `ProductServiceClient` import and `InitProductClient` — **no change needed**. (Verified: `merchant-bff/ioc/grpc.go` does NOT have `InitProductClient`.)

Wait — re-checking: the merchant-bff `ioc/grpc.go` already imports `productv1` but does NOT have `InitProductClient`. However the `wire.go` already has `ioc.InitProductClient` in `thirdPartySet`. This means there's a build error currently, or the function exists. Let me re-verify.

**Actually confirmed**: merchant-bff `ioc/grpc.go` does NOT have `InitProductClient`. But `wire.go` already lists it in `thirdPartySet`, and `wire_gen.go` calls it. This means the function MUST already exist. Checking the wire_gen.go confirms `productServiceClient := ioc.InitProductClient(client)` is called but the client is only used in `handler.NewCartHandler(cartServiceClient, productServiceClient, logger)`. So `InitProductClient` already exists but is only used by CartHandler.

**No change needed for `merchant-bff/ioc/grpc.go`** — `InitProductClient` already exists.

### Step 3: Modify `merchant-bff/ioc/gin.go` — add `productHandler` param + 11 routes

Add `productHandler *handler.ProductHandler` parameter to `InitGinServer` and add product routes.

**Current** `InitGinServer` signature:
```go
func InitGinServer(
	jwtHandler *ijwt.JWTHandler,
	userHandler *handler.UserHandler,
	tenantHandler *handler.TenantHandler,
	inventoryHandler *handler.InventoryHandler,
	orderHandler *handler.OrderHandler,
	paymentHandler *handler.PaymentHandler,
	marketingHandler *handler.MarketingHandler,
	logisticsHandler *handler.LogisticsHandler,
	notificationHandler *handler.NotificationHandler,
	l logger.Logger,
) *gin.Engine {
```

**New** — add `productHandler *handler.ProductHandler` after `notificationHandler`:
```go
func InitGinServer(
	jwtHandler *ijwt.JWTHandler,
	userHandler *handler.UserHandler,
	tenantHandler *handler.TenantHandler,
	inventoryHandler *handler.InventoryHandler,
	orderHandler *handler.OrderHandler,
	paymentHandler *handler.PaymentHandler,
	marketingHandler *handler.MarketingHandler,
	logisticsHandler *handler.LogisticsHandler,
	notificationHandler *handler.NotificationHandler,
	productHandler *handler.ProductHandler,
	l logger.Logger,
) *gin.Engine {
```

**Add routes** after the notification routes block, before the closing `}` of `auth`:
```go
		// 商品管理
		auth.POST("/products", ginx.WrapBody[handler.CreateProductReq](l, productHandler.CreateProduct))
		auth.PUT("/products/:id", ginx.WrapBody[handler.UpdateProductReq](l, productHandler.UpdateProduct))
		auth.GET("/products/:id", productHandler.GetProduct)
		auth.GET("/products", ginx.WrapQuery[handler.ListProductsReq](l, productHandler.ListProducts))
		auth.PUT("/products/:id/status", ginx.WrapBody[handler.UpdateProductStatusReq](l, productHandler.UpdateProductStatus))
		// 分类管理
		auth.POST("/categories", ginx.WrapBody[handler.CreateCategoryReq](l, productHandler.CreateCategory))
		auth.PUT("/categories/:id", ginx.WrapBody[handler.UpdateCategoryReq](l, productHandler.UpdateCategory))
		auth.GET("/categories", ginx.WrapQuery[handler.ListCategoriesReq](l, productHandler.ListCategories))
		// 品牌管理
		auth.POST("/brands", ginx.WrapBody[handler.CreateBrandReq](l, productHandler.CreateBrand))
		auth.PUT("/brands/:id", ginx.WrapBody[handler.UpdateBrandReq](l, productHandler.UpdateBrand))
		auth.GET("/brands", ginx.WrapQuery[handler.ListBrandsReq](l, productHandler.ListBrands))
```

### Step 4: Modify `merchant-bff/wire.go` — add `handler.NewProductHandler`

Add `handler.NewProductHandler` to `handlerSet`:
```go
var handlerSet = wire.NewSet(
	ijwt.NewJWTHandler,
	handler.NewUserHandler,
	handler.NewTenantHandler,
	handler.NewInventoryHandler,
	handler.NewOrderHandler,
	handler.NewPaymentHandler,
	handler.NewMarketingHandler,
	handler.NewLogisticsHandler,
	handler.NewNotificationHandler,
	handler.NewProductHandler,
	ioc.InitGinServer,
)
```

Note: `ioc.InitProductClient` is already in `thirdPartySet`.

### Step 5: Regenerate `merchant-bff/wire_gen.go`

Run: `cd merchant-bff && wire`

### Step 6: Verify build

Run: `go build ./merchant-bff/...`

### Step 7: Commit

```bash
git add merchant-bff/handler/product.go merchant-bff/ioc/gin.go merchant-bff/wire.go merchant-bff/wire_gen.go
git commit -m "feat(merchant-bff): add product management endpoints (11 routes)"
```

---

## Task 2: admin-bff — Product Handler (6 endpoints: categories + brands)

**Files:**
- Create: `admin-bff/handler/product.go`

### Step 1: Create `admin-bff/handler/product.go`

Admin manages platform-level categories and brands (tenant_id=0).

```go
package handler

import (
	"fmt"
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
	return &ProductHandler{
		productClient: productClient,
		l:             l,
	}
}

// ==================== 平台分类管理 ====================

type AdminCreateCategoryReq struct {
	ParentId int64  `json:"parent_id"`
	Name     string `json:"name" binding:"required"`
	Level    int32  `json:"level"`
	Sort     int32  `json:"sort"`
	Icon     string `json:"icon"`
	Status   int32  `json:"status"`
}

func (h *ProductHandler) CreateCategory(ctx *gin.Context, req AdminCreateCategoryReq) (ginx.Result, error) {
	resp, err := h.productClient.CreateCategory(ctx.Request.Context(), &productv1.CreateCategoryRequest{
		Category: &productv1.Category{
			TenantId: 0, // 平台级分类
			ParentId: req.ParentId,
			Name:     req.Name,
			Level:    req.Level,
			Sort:     req.Sort,
			Icon:     req.Icon,
			Status:   req.Status,
		},
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("创建平台分类失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{"id": resp.GetId()}}, nil
}

type AdminUpdateCategoryReq struct {
	ParentId int64  `json:"parent_id"`
	Name     string `json:"name"`
	Level    int32  `json:"level"`
	Sort     int32  `json:"sort"`
	Icon     string `json:"icon"`
	Status   int32  `json:"status"`
}

func (h *ProductHandler) UpdateCategory(ctx *gin.Context, req AdminUpdateCategoryReq) (ginx.Result, error) {
	idStr := ctx.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	_, err := h.productClient.UpdateCategory(ctx.Request.Context(), &productv1.UpdateCategoryRequest{
		Category: &productv1.Category{
			Id:       id,
			TenantId: 0,
			ParentId: req.ParentId,
			Name:     req.Name,
			Level:    req.Level,
			Sort:     req.Sort,
			Icon:     req.Icon,
			Status:   req.Status,
		},
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("更新平台分类失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success"}, nil
}

type AdminListCategoriesReq struct{}

func (h *ProductHandler) ListCategories(ctx *gin.Context, _ AdminListCategoriesReq) (ginx.Result, error) {
	resp, err := h.productClient.ListCategories(ctx.Request.Context(), &productv1.ListCategoriesRequest{
		TenantId: 0,
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("查询平台分类列表失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: resp.GetCategories()}, nil
}

// ==================== 平台品牌管理 ====================

type AdminCreateBrandReq struct {
	Name   string `json:"name" binding:"required"`
	Logo   string `json:"logo"`
	Status int32  `json:"status"`
}

func (h *ProductHandler) CreateBrand(ctx *gin.Context, req AdminCreateBrandReq) (ginx.Result, error) {
	resp, err := h.productClient.CreateBrand(ctx.Request.Context(), &productv1.CreateBrandRequest{
		Brand: &productv1.Brand{
			TenantId: 0, // 平台级品牌
			Name:     req.Name,
			Logo:     req.Logo,
			Status:   req.Status,
		},
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("创建平台品牌失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{"id": resp.GetId()}}, nil
}

type AdminUpdateBrandReq struct {
	Name   string `json:"name"`
	Logo   string `json:"logo"`
	Status int32  `json:"status"`
}

func (h *ProductHandler) UpdateBrand(ctx *gin.Context, req AdminUpdateBrandReq) (ginx.Result, error) {
	idStr := ctx.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	_, err := h.productClient.UpdateBrand(ctx.Request.Context(), &productv1.UpdateBrandRequest{
		Brand: &productv1.Brand{
			Id:       id,
			TenantId: 0,
			Name:     req.Name,
			Logo:     req.Logo,
			Status:   req.Status,
		},
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("更新平台品牌失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success"}, nil
}

type AdminListBrandsReq struct {
	Page     int32 `form:"page" binding:"required,min=1"`
	PageSize int32 `form:"page_size" binding:"required,min=1,max=100"`
}

func (h *ProductHandler) ListBrands(ctx *gin.Context, req AdminListBrandsReq) (ginx.Result, error) {
	resp, err := h.productClient.ListBrands(ctx.Request.Context(), &productv1.ListBrandsRequest{
		TenantId: 0,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("查询平台品牌列表失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{
		"brands": resp.GetBrands(),
		"total":  resp.GetTotal(),
	}}, nil
}
```

### Step 2: Commit

```bash
git add admin-bff/handler/product.go
git commit -m "feat(admin-bff): add platform category and brand management handler"
```

---

## Task 3: admin-bff — Order Handler (2 endpoints)

**Files:**
- Create: `admin-bff/handler/order.go`

### Step 1: Create `admin-bff/handler/order.go`

```go
package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rermrf/emo/logger"
	orderv1 "github.com/rermrf/mall/api/proto/gen/order/v1"
	"github.com/rermrf/mall/pkg/ginx"
)

type OrderHandler struct {
	orderClient orderv1.OrderServiceClient
	l           logger.Logger
}

func NewOrderHandler(orderClient orderv1.OrderServiceClient, l logger.Logger) *OrderHandler {
	return &OrderHandler{
		orderClient: orderClient,
		l:           l,
	}
}

type AdminListOrdersReq struct {
	TenantId int64 `form:"tenant_id"`
	Status   int32 `form:"status"`
	Page     int32 `form:"page" binding:"required,min=1"`
	PageSize int32 `form:"page_size" binding:"required,min=1,max=100"`
}

func (h *OrderHandler) ListOrders(ctx *gin.Context, req AdminListOrdersReq) (ginx.Result, error) {
	resp, err := h.orderClient.ListOrders(ctx.Request.Context(), &orderv1.ListOrdersRequest{
		TenantId: req.TenantId,
		Status:   req.Status,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("查询全平台订单列表失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{
		"orders": resp.GetOrders(),
		"total":  resp.GetTotal(),
	}}, nil
}

func (h *OrderHandler) GetOrder(ctx *gin.Context) {
	orderNo := ctx.Param("orderNo")
	resp, err := h.orderClient.GetOrder(ctx.Request.Context(), &orderv1.GetOrderRequest{
		OrderNo: orderNo,
	})
	if err != nil {
		h.l.Error("查询订单详情失败", logger.Error(err))
		ctx.JSON(http.StatusOK, ginx.Result{Code: 5, Msg: "系统错误"})
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetOrder()})
}
```

### Step 2: Commit

```bash
git add admin-bff/handler/order.go
git commit -m "feat(admin-bff): add order monitoring handler"
```

---

## Task 4: admin-bff — Payment Handler (2 endpoints)

**Files:**
- Create: `admin-bff/handler/payment.go`

### Step 1: Create `admin-bff/handler/payment.go`

```go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rermrf/emo/logger"
	paymentv1 "github.com/rermrf/mall/api/proto/gen/payment/v1"
	"github.com/rermrf/mall/pkg/ginx"
)

type PaymentHandler struct {
	paymentClient paymentv1.PaymentServiceClient
	l             logger.Logger
}

func NewPaymentHandler(paymentClient paymentv1.PaymentServiceClient, l logger.Logger) *PaymentHandler {
	return &PaymentHandler{
		paymentClient: paymentClient,
		l:             l,
	}
}

func (h *PaymentHandler) GetPayment(ctx *gin.Context) {
	paymentNo := ctx.Param("paymentNo")
	resp, err := h.paymentClient.GetPayment(ctx.Request.Context(), &paymentv1.GetPaymentRequest{
		PaymentNo: paymentNo,
	})
	if err != nil {
		h.l.Error("查询支付详情失败", logger.Error(err))
		ctx.JSON(http.StatusOK, ginx.Result{Code: 5, Msg: "系统错误"})
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetPayment()})
}

func (h *PaymentHandler) GetRefund(ctx *gin.Context) {
	refundNo := ctx.Param("refundNo")
	resp, err := h.paymentClient.GetRefund(ctx.Request.Context(), &paymentv1.GetRefundRequest{
		RefundNo: refundNo,
	})
	if err != nil {
		h.l.Error("查询退款详情失败", logger.Error(err))
		ctx.JSON(http.StatusOK, ginx.Result{Code: 5, Msg: "系统错误"})
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetRefund()})
}
```

### Step 2: Commit

```bash
git add admin-bff/handler/payment.go
git commit -m "feat(admin-bff): add payment monitoring handler"
```

---

## Task 5: admin-bff — Notification Handler (5 endpoints)

**Files:**
- Create: `admin-bff/handler/notification.go`

### Step 1: Create `admin-bff/handler/notification.go`

```go
package handler

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/rermrf/emo/logger"
	notificationv1 "github.com/rermrf/mall/api/proto/gen/notification/v1"
	"github.com/rermrf/mall/pkg/ginx"
)

type NotificationHandler struct {
	notificationClient notificationv1.NotificationServiceClient
	l                  logger.Logger
}

func NewNotificationHandler(notificationClient notificationv1.NotificationServiceClient, l logger.Logger) *NotificationHandler {
	return &NotificationHandler{
		notificationClient: notificationClient,
		l:                  l,
	}
}

// ==================== 通知模板管理 ====================

type CreateTemplateReq struct {
	TenantId int64  `json:"tenant_id"`
	Code     string `json:"code" binding:"required"`
	Channel  int32  `json:"channel" binding:"required"`
	Title    string `json:"title" binding:"required"`
	Content  string `json:"content" binding:"required"`
	Status   int32  `json:"status"`
}

func (h *NotificationHandler) CreateTemplate(ctx *gin.Context, req CreateTemplateReq) (ginx.Result, error) {
	resp, err := h.notificationClient.CreateTemplate(ctx.Request.Context(), &notificationv1.CreateTemplateRequest{
		Template: &notificationv1.NotificationTemplate{
			TenantId: req.TenantId,
			Code:     req.Code,
			Channel:  req.Channel,
			Title:    req.Title,
			Content:  req.Content,
			Status:   req.Status,
		},
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("创建通知模板失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{"id": resp.GetId()}}, nil
}

type UpdateTemplateReq struct {
	TenantId int64  `json:"tenant_id"`
	Code     string `json:"code"`
	Channel  int32  `json:"channel"`
	Title    string `json:"title"`
	Content  string `json:"content"`
	Status   int32  `json:"status"`
}

func (h *NotificationHandler) UpdateTemplate(ctx *gin.Context, req UpdateTemplateReq) (ginx.Result, error) {
	idStr := ctx.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	_, err := h.notificationClient.UpdateTemplate(ctx.Request.Context(), &notificationv1.UpdateTemplateRequest{
		Template: &notificationv1.NotificationTemplate{
			Id:       id,
			TenantId: req.TenantId,
			Code:     req.Code,
			Channel:  req.Channel,
			Title:    req.Title,
			Content:  req.Content,
			Status:   req.Status,
		},
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("更新通知模板失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success"}, nil
}

type ListTemplatesReq struct {
	TenantId int64 `form:"tenant_id"`
	Channel  int32 `form:"channel"`
}

func (h *NotificationHandler) ListTemplates(ctx *gin.Context, req ListTemplatesReq) (ginx.Result, error) {
	resp, err := h.notificationClient.ListTemplates(ctx.Request.Context(), &notificationv1.ListTemplatesRequest{
		TenantId: req.TenantId,
		Channel:  req.Channel,
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("查询通知模板列表失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: resp.GetTemplates()}, nil
}

func (h *NotificationHandler) DeleteTemplate(ctx *gin.Context) (ginx.Result, error) {
	idStr := ctx.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	_, err := h.notificationClient.DeleteTemplate(ctx.Request.Context(), &notificationv1.DeleteTemplateRequest{
		Id: id,
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("删除通知模板失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success"}, nil
}

// ==================== 发送通知 ====================

type SendNotificationReq struct {
	UserId       int64             `json:"user_id" binding:"required"`
	TenantId     int64             `json:"tenant_id"`
	TemplateCode string            `json:"template_code" binding:"required"`
	Channel      int32             `json:"channel" binding:"required"`
	Params       map[string]string `json:"params"`
}

func (h *NotificationHandler) SendNotification(ctx *gin.Context, req SendNotificationReq) (ginx.Result, error) {
	resp, err := h.notificationClient.SendNotification(ctx.Request.Context(), &notificationv1.SendNotificationRequest{
		UserId:       req.UserId,
		TenantId:     req.TenantId,
		TemplateCode: req.TemplateCode,
		Channel:      req.Channel,
		Params:       req.Params,
	})
	if err != nil {
		return ginx.Result{}, fmt.Errorf("发送通知失败: %w", err)
	}
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{"id": resp.GetId()}}, nil
}
```

### Step 2: Commit

```bash
git add admin-bff/handler/notification.go
git commit -m "feat(admin-bff): add notification template management and send handler"
```

---

## Task 6: admin-bff — Wire up 4 new services (grpc + gin + wire)

**Files:**
- Modify: `admin-bff/ioc/grpc.go` — add 4 Init*Client functions
- Modify: `admin-bff/ioc/gin.go` — add 4 handler params + 15 routes
- Modify: `admin-bff/wire.go` — add 4 clients + 4 handlers
- Regenerate: `admin-bff/wire_gen.go`

### Step 1: Add 4 clients to `admin-bff/ioc/grpc.go`

Append after `InitTenantClient`:

```go
import (
	// existing imports...
	notificationv1 "github.com/rermrf/mall/api/proto/gen/notification/v1"
	orderv1 "github.com/rermrf/mall/api/proto/gen/order/v1"
	paymentv1 "github.com/rermrf/mall/api/proto/gen/payment/v1"
	productv1 "github.com/rermrf/mall/api/proto/gen/product/v1"
)

func InitProductClient(etcdClient *clientv3.Client) productv1.ProductServiceClient {
	conn := initServiceConn(etcdClient, "product")
	return productv1.NewProductServiceClient(conn)
}

func InitOrderClient(etcdClient *clientv3.Client) orderv1.OrderServiceClient {
	conn := initServiceConn(etcdClient, "order")
	return orderv1.NewOrderServiceClient(conn)
}

func InitPaymentClient(etcdClient *clientv3.Client) paymentv1.PaymentServiceClient {
	conn := initServiceConn(etcdClient, "payment")
	return paymentv1.NewPaymentServiceClient(conn)
}

func InitNotificationClient(etcdClient *clientv3.Client) notificationv1.NotificationServiceClient {
	conn := initServiceConn(etcdClient, "notification")
	return notificationv1.NewNotificationServiceClient(conn)
}
```

### Step 2: Modify `admin-bff/ioc/gin.go`

**New signature:**
```go
func InitGinServer(
	jwtHandler *ijwt.JWTHandler,
	userHandler *handler.UserHandler,
	tenantHandler *handler.TenantHandler,
	productHandler *handler.ProductHandler,
	orderHandler *handler.OrderHandler,
	paymentHandler *handler.PaymentHandler,
	notificationHandler *handler.NotificationHandler,
	l logger.Logger,
) *gin.Engine {
```

**Add 15 routes** inside the `auth` group after the plan management routes:

```go
		// 平台分类管理
		auth.POST("/categories", ginx.WrapBody[handler.AdminCreateCategoryReq](l, productHandler.CreateCategory))
		auth.PUT("/categories/:id", ginx.WrapBody[handler.AdminUpdateCategoryReq](l, productHandler.UpdateCategory))
		auth.GET("/categories", ginx.WrapQuery[handler.AdminListCategoriesReq](l, productHandler.ListCategories))
		// 平台品牌管理
		auth.POST("/brands", ginx.WrapBody[handler.AdminCreateBrandReq](l, productHandler.CreateBrand))
		auth.PUT("/brands/:id", ginx.WrapBody[handler.AdminUpdateBrandReq](l, productHandler.UpdateBrand))
		auth.GET("/brands", ginx.WrapQuery[handler.AdminListBrandsReq](l, productHandler.ListBrands))
		// 订单监管
		auth.GET("/orders", ginx.WrapQuery[handler.AdminListOrdersReq](l, orderHandler.ListOrders))
		auth.GET("/orders/:orderNo", orderHandler.GetOrder)
		// 支付监管
		auth.GET("/payments/:paymentNo", paymentHandler.GetPayment)
		auth.GET("/refunds/:refundNo", paymentHandler.GetRefund)
		// 通知模板管理
		auth.POST("/notification-templates", ginx.WrapBody[handler.CreateTemplateReq](l, notificationHandler.CreateTemplate))
		auth.PUT("/notification-templates/:id", ginx.WrapBody[handler.UpdateTemplateReq](l, notificationHandler.UpdateTemplate))
		auth.GET("/notification-templates", ginx.WrapQuery[handler.ListTemplatesReq](l, notificationHandler.ListTemplates))
		auth.DELETE("/notification-templates/:id", ginx.Wrap(l, notificationHandler.DeleteTemplate))
		// 发送通知
		auth.POST("/notifications/send", ginx.WrapBody[handler.SendNotificationReq](l, notificationHandler.SendNotification))
```

### Step 3: Modify `admin-bff/wire.go`

```go
var thirdPartySet = wire.NewSet(
	ioc.InitEtcdClient,
	ioc.InitLogger,
	ioc.InitUserClient,
	ioc.InitTenantClient,
	ioc.InitProductClient,
	ioc.InitOrderClient,
	ioc.InitPaymentClient,
	ioc.InitNotificationClient,
)

var handlerSet = wire.NewSet(
	ijwt.NewJWTHandler,
	handler.NewUserHandler,
	handler.NewTenantHandler,
	handler.NewProductHandler,
	handler.NewOrderHandler,
	handler.NewPaymentHandler,
	handler.NewNotificationHandler,
	ioc.InitGinServer,
)
```

### Step 4: Regenerate `admin-bff/wire_gen.go`

Run: `cd admin-bff && wire`

### Step 5: Verify build

Run: `go build ./admin-bff/...`

### Step 6: Commit

```bash
git add admin-bff/ioc/grpc.go admin-bff/ioc/gin.go admin-bff/wire.go admin-bff/wire_gen.go
git commit -m "feat(admin-bff): wire up product, order, payment, notification services"
```

**Note on `ginx.Wrap`**: The `DeleteTemplate` handler takes only `*gin.Context` and returns `(ginx.Result, error)`. Check if `ginx.Wrap` exists for this signature (no request body). If not, use a manual wrapper:
```go
auth.DELETE("/notification-templates/:id", func(c *gin.Context) {
    res, err := notificationHandler.DeleteTemplate(c)
    if err != nil {
        l.Error("删除通知模板失败", logger.Error(err))
        c.JSON(http.StatusOK, ginx.Result{Code: 5, Msg: "系统错误"})
        return
    }
    c.JSON(http.StatusOK, res)
})
```

---

## Task 7: notification-svc — Add 2 new Kafka consumers

**Files:**
- Modify: `notification/events/types.go` — add 2 event types + topic constants
- Modify: `notification/events/consumer.go` — add 2 consumer structs
- Modify: `notification/ioc/kafka.go` — add 2 consumer constructors + update `InitConsumers`
- Modify: `notification/wire.go` — add 2 consumers

### Step 1: Add to `notification/events/types.go`

Append 2 topic constants and 2 event structs:

```go
const (
	// ... existing 5 topics ...
	TopicTenantPlanChanged = "tenant_plan_changed"
	TopicOrderCompleted    = "order_completed"
)

type TenantPlanChangedEvent struct {
	TenantId  int64 `json:"tenant_id"`
	OldPlanId int64 `json:"old_plan_id"`
	NewPlanId int64 `json:"new_plan_id"`
}

type OrderCompletedEvent struct {
	OrderNo  string              `json:"order_no"`
	TenantID int64               `json:"tenant_id"`
	Items    []CompletedItemInfo `json:"items"`
}

type CompletedItemInfo struct {
	ProductID int64 `json:"product_id"`
	Quantity  int32 `json:"quantity"`
}
```

### Step 2: Add to `notification/events/consumer.go`

Append 2 consumer structs (follow exact same pattern as existing 5):

```go
// ==================== TenantPlanChangedConsumer ====================

type TenantPlanChangedConsumer struct {
	client  sarama.ConsumerGroup
	l       logger.Logger
	handler func(ctx context.Context, evt TenantPlanChangedEvent) error
}

func NewTenantPlanChangedConsumer(client sarama.ConsumerGroup, l logger.Logger, handler func(ctx context.Context, evt TenantPlanChangedEvent) error) *TenantPlanChangedConsumer {
	return &TenantPlanChangedConsumer{client: client, l: l, handler: handler}
}

func (c *TenantPlanChangedConsumer) Start() error {
	h := saramax.NewHandler[TenantPlanChangedEvent](c.l, c.Consume)
	go func() {
		for {
			if err := c.client.Consume(context.Background(), []string{TopicTenantPlanChanged}, h); err != nil {
				c.l.Error("消费 tenant_plan_changed 事件出错", logger.Error(err))
				time.Sleep(time.Second)
			}
		}
	}()
	return nil
}

func (c *TenantPlanChangedConsumer) Consume(msg *sarama.ConsumerMessage, evt TenantPlanChangedEvent) error {
	return c.handler(context.Background(), evt)
}

// ==================== OrderCompletedConsumer ====================

type OrderCompletedConsumer struct {
	client  sarama.ConsumerGroup
	l       logger.Logger
	handler func(ctx context.Context, evt OrderCompletedEvent) error
}

func NewOrderCompletedConsumer(client sarama.ConsumerGroup, l logger.Logger, handler func(ctx context.Context, evt OrderCompletedEvent) error) *OrderCompletedConsumer {
	return &OrderCompletedConsumer{client: client, l: l, handler: handler}
}

func (c *OrderCompletedConsumer) Start() error {
	h := saramax.NewHandler[OrderCompletedEvent](c.l, c.Consume)
	go func() {
		for {
			if err := c.client.Consume(context.Background(), []string{TopicOrderCompleted}, h); err != nil {
				c.l.Error("消费 order_completed 事件出错", logger.Error(err))
				time.Sleep(time.Second)
			}
		}
	}()
	return nil
}

func (c *OrderCompletedConsumer) Consume(msg *sarama.ConsumerMessage, evt OrderCompletedEvent) error {
	return c.handler(context.Background(), evt)
}
```

### Step 3: Add to `notification/ioc/kafka.go`

Append 2 constructor functions:

```go
func NewTenantPlanChangedConsumer(
	cg sarama.ConsumerGroup,
	l logger.Logger,
	svc service.NotificationService,
) *events.TenantPlanChangedConsumer {
	return events.NewTenantPlanChangedConsumer(cg, l, func(ctx context.Context, evt events.TenantPlanChangedEvent) error {
		l.Info("收到租户套餐变更事件", logger.Int64("tenantId", evt.TenantId))
		params := map[string]string{
			"OldPlanId": strconv.FormatInt(evt.OldPlanId, 10),
			"NewPlanId": strconv.FormatInt(evt.NewPlanId, 10),
		}
		// TODO: 需要从租户服务获取商家管理员ID，暂用 tenantId 作为 userId
		_, _ = svc.SendNotification(ctx, evt.TenantId, evt.TenantId, "tenant_plan_changed_inapp", 3, params)
		_, _ = svc.SendNotification(ctx, evt.TenantId, evt.TenantId, "tenant_plan_changed_email", 2, params)
		return nil
	})
}

func NewOrderCompletedConsumer(
	cg sarama.ConsumerGroup,
	l logger.Logger,
	svc service.NotificationService,
) *events.OrderCompletedConsumer {
	return events.NewOrderCompletedConsumer(cg, l, func(ctx context.Context, evt events.OrderCompletedEvent) error {
		l.Info("收到订单完成事件", logger.String("orderNo", evt.OrderNo))
		params := map[string]string{
			"OrderNo": evt.OrderNo,
		}
		// TODO: 需要从订单服务获取买家 ID，暂用日志记录
		l.Warn("order_completed 事件暂缺买家ID，跳过站内信通知")
		_ = params
		return nil
	})
}
```

**Update `InitConsumers`** to include the 2 new consumers:

```go
func InitConsumers(
	userRegistered *events.UserRegisteredConsumer,
	orderPaid *events.OrderPaidConsumer,
	orderShipped *events.OrderShippedConsumer,
	inventoryAlert *events.InventoryAlertConsumer,
	tenantApproved *events.TenantApprovedConsumer,
	tenantPlanChanged *events.TenantPlanChangedConsumer,
	orderCompleted *events.OrderCompletedConsumer,
) []saramax.Consumer {
	return []saramax.Consumer{userRegistered, orderPaid, orderShipped, inventoryAlert, tenantApproved, tenantPlanChanged, orderCompleted}
}
```

### Step 4: Update `notification/wire.go`

Add the 2 new consumer constructors to `notificationSet`:

```go
var notificationSet = wire.NewSet(
	// ... existing entries ...
	ioc.NewTenantPlanChangedConsumer,
	ioc.NewOrderCompletedConsumer,
	// ... ioc.InitConsumers stays ...
)
```

### Step 5: Regenerate `notification/wire_gen.go`

Run: `cd notification && wire`

### Step 6: Verify build

Run: `go build ./notification/...`

### Step 7: Commit

```bash
git add notification/events/types.go notification/events/consumer.go notification/ioc/kafka.go notification/wire.go notification/wire_gen.go
git commit -m "feat(notification): add tenant_plan_changed and order_completed consumers"
```

---

## Task 8: order-svc — Add seckill_success Kafka consumer

**Files:**
- Modify: `order/events/types.go` — add SeckillSuccessEvent + topic constant
- Modify: `order/events/consumer.go` — add SeckillSuccessConsumer struct
- Modify: `order/ioc/kafka.go` — add NewSeckillSuccessConsumer + update InitConsumers
- Modify: `order/wire.go` — add consumer

### Step 1: Add to `order/events/types.go`

Append:

```go
const (
	// ... existing constants ...
	TopicSeckillSuccess = "seckill_success"
)

type SeckillSuccessEvent struct {
	UserId       int64 `json:"user_id"`
	ItemId       int64 `json:"item_id"`
	SkuId        int64 `json:"sku_id"`
	SeckillPrice int64 `json:"seckill_price"`
	TenantId     int64 `json:"tenant_id"`
}
```

### Step 2: Add to `order/events/consumer.go`

Append:

```go
// SeckillSuccessConsumer 消费 marketing-svc 的秒杀成功事件
type SeckillSuccessConsumer struct {
	client  sarama.ConsumerGroup
	l       logger.Logger
	handler func(ctx context.Context, evt SeckillSuccessEvent) error
}

func NewSeckillSuccessConsumer(
	client sarama.ConsumerGroup,
	l logger.Logger,
	handler func(ctx context.Context, evt SeckillSuccessEvent) error,
) *SeckillSuccessConsumer {
	return &SeckillSuccessConsumer{client: client, l: l, handler: handler}
}

func (c *SeckillSuccessConsumer) Start() error {
	cg := c.client
	h := saramax.NewHandler[SeckillSuccessEvent](c.l, c.Consume)
	go func() {
		for {
			err := cg.Consume(context.Background(), []string{TopicSeckillSuccess}, h)
			if err != nil {
				c.l.Error("消费 seckill_success 事件出错", logger.Error(err))
				time.Sleep(time.Second)
			}
		}
	}()
	return nil
}

func (c *SeckillSuccessConsumer) Consume(msg *sarama.ConsumerMessage, evt SeckillSuccessEvent) error {
	c.l.Info("收到秒杀成功事件",
		logger.Int64("userId", evt.UserId),
		logger.Int64("skuId", evt.SkuId))
	return c.handler(context.Background(), evt)
}
```

### Step 3: Add to `order/ioc/kafka.go`

Append constructor:

```go
func NewSeckillSuccessConsumer(
	cg sarama.ConsumerGroup,
	l logger.Logger,
	svc service.OrderService,
) *events.SeckillSuccessConsumer {
	return events.NewSeckillSuccessConsumer(cg, l, func(ctx context.Context, evt events.SeckillSuccessEvent) error {
		l.Info("收到秒杀成功事件，准备创建秒杀订单",
			logger.Int64("userId", evt.UserId),
			logger.Int64("skuId", evt.SkuId),
			logger.Int64("seckillPrice", evt.SeckillPrice))
		// TODO: 自动创建秒杀订单
		// 需要：1) 从 product-svc 获取商品详情  2) 从 user-svc 获取默认收货地址
		// 暂时仅记录日志
		l.Warn("秒杀订单自动创建尚未实现，需跨服务获取商品详情和收货地址",
			logger.Int64("userId", evt.UserId),
			logger.Int64("skuId", evt.SkuId))
		return nil
	})
}
```

**Update `InitConsumers`:**

```go
func InitConsumers(
	paidConsumer *events.OrderPaidConsumer,
	closeDelayConsumer *events.OrderCloseDelayConsumer,
	seckillConsumer *events.SeckillSuccessConsumer,
) []saramax.Consumer {
	return []saramax.Consumer{paidConsumer, closeDelayConsumer, seckillConsumer}
}
```

### Step 4: Update `order/wire.go`

Add `ioc.NewSeckillSuccessConsumer` to `orderSet`:

```go
var orderSet = wire.NewSet(
	// ... existing entries ...
	ioc.NewSeckillSuccessConsumer,
	// ioc.InitConsumers stays
	// ioc.InitGRPCServer stays
)
```

### Step 5: Regenerate `order/wire_gen.go`

Run: `cd order && wire`

### Step 6: Verify build

Run: `go build ./order/...`

### Step 7: Commit

```bash
git add order/events/types.go order/events/consumer.go order/ioc/kafka.go order/wire.go order/wire_gen.go
git commit -m "feat(order): add seckill_success consumer (TODO: auto-create order)"
```

---

## Task 9: Fix notification + marketing consumer handlers

**Files:**
- Modify: `notification/ioc/kafka.go` — improve order_paid and order_shipped handlers
- Modify: `marketing/ioc/kafka.go` — implement order_cancelled handler with ReleaseCoupon

### Step 1: Fix `notification/ioc/kafka.go` — `NewOrderPaidConsumer`

Replace the handler body in `NewOrderPaidConsumer`:

```go
func NewOrderPaidConsumer(
	cg sarama.ConsumerGroup,
	l logger.Logger,
	svc service.NotificationService,
) *events.OrderPaidConsumer {
	return events.NewOrderPaidConsumer(cg, l, func(ctx context.Context, evt events.OrderPaidEvent) error {
		l.Info("收到订单支付事件", logger.String("orderNo", evt.OrderNo))
		// TODO: 需要从 order-svc 获取商家 tenant_id 和商家管理员 user_id
		// 目前 OrderPaidEvent 只有 OrderNo/PaymentNo/PaidAt，缺少商家信息
		// 后续可通过 gRPC 调用 order-svc.GetOrder 获取 tenant_id，再查商家管理员
		l.Warn("order_paid: 暂缺商家信息，无法发送站内信通知",
			logger.String("orderNo", evt.OrderNo),
			logger.String("paymentNo", evt.PaymentNo))
		return nil
	})
}
```

### Step 2: Fix `notification/ioc/kafka.go` — `NewOrderShippedConsumer`

Replace the handler body in `NewOrderShippedConsumer`:

```go
func NewOrderShippedConsumer(
	cg sarama.ConsumerGroup,
	l logger.Logger,
	svc service.NotificationService,
) *events.OrderShippedConsumer {
	return events.NewOrderShippedConsumer(cg, l, func(ctx context.Context, evt events.OrderShippedEvent) error {
		l.Info("收到订单发货事件",
			logger.Int64("orderId", evt.OrderId),
			logger.String("trackingNo", evt.TrackingNo))
		// TODO: 需要从 order-svc 获取买家 user_id 和手机号
		// OrderShippedEvent 有 TenantId 但无 buyer_id
		// 后续可通过 gRPC 调用 order-svc 获取订单详情中的 buyer_id
		l.Warn("order_shipped: 暂缺买家信息，无法发送 SMS/站内信通知",
			logger.Int64("orderId", evt.OrderId),
			logger.Int64("tenantId", evt.TenantId))
		return nil
	})
}
```

### Step 3: Fix `marketing/ioc/kafka.go` — `NewOrderCancelledConsumer`

Replace the handler body:

```go
func NewOrderCancelledConsumer(
	cg sarama.ConsumerGroup,
	l logger.Logger,
	svc service.MarketingService,
) *events.OrderCancelledConsumer {
	return events.NewOrderCancelledConsumer(cg, l, func(ctx context.Context, evt events.OrderCancelledEvent) error {
		l.Info("收到订单取消事件",
			logger.String("orderNo", evt.OrderNo),
			logger.Int64("tenantId", evt.TenantID))
		// TODO: 需要从 order-svc 获取订单关联的 userCouponId
		// OrderCancelledEvent 只有 OrderNo/TenantID/Reason，缺少优惠券信息
		// 后续可通过 gRPC 调用 order-svc.GetOrder 获取 coupon_id 后调用 svc.ReleaseCoupon
		l.Warn("order_cancelled: 暂缺优惠券信息，无法释放优惠券",
			logger.String("orderNo", evt.OrderNo))
		return nil
	})
}
```

### Step 4: Verify builds

Run:
```bash
go build ./notification/...
go build ./marketing/...
```

### Step 5: Commit

```bash
git add notification/ioc/kafka.go marketing/ioc/kafka.go
git commit -m "fix: improve notification and marketing consumer handlers with clear TODOs"
```

---

## Task 10: Final Verification

### Step 1: Build all affected packages

```bash
go build ./merchant-bff/...
go build ./admin-bff/...
go build ./notification/...
go build ./order/...
go build ./marketing/...
```

### Step 2: Run go vet

```bash
go vet ./merchant-bff/...
go vet ./admin-bff/...
go vet ./notification/...
go vet ./order/...
go vet ./marketing/...
```

All should pass with zero errors.

---

## File Change Summary

| # | File | Operation | Task |
|---|------|-----------|------|
| 1 | `merchant-bff/handler/product.go` | Create | Task 1 |
| 2 | `merchant-bff/ioc/gin.go` | Modify | Task 1 |
| 3 | `merchant-bff/wire.go` | Modify | Task 1 |
| 4 | `merchant-bff/wire_gen.go` | Regenerate | Task 1 |
| 5 | `admin-bff/handler/product.go` | Create | Task 2 |
| 6 | `admin-bff/handler/order.go` | Create | Task 3 |
| 7 | `admin-bff/handler/payment.go` | Create | Task 4 |
| 8 | `admin-bff/handler/notification.go` | Create | Task 5 |
| 9 | `admin-bff/ioc/grpc.go` | Modify | Task 6 |
| 10 | `admin-bff/ioc/gin.go` | Modify | Task 6 |
| 11 | `admin-bff/wire.go` | Modify | Task 6 |
| 12 | `admin-bff/wire_gen.go` | Regenerate | Task 6 |
| 13 | `notification/events/types.go` | Modify | Task 7 |
| 14 | `notification/events/consumer.go` | Modify | Task 7 |
| 15 | `notification/ioc/kafka.go` | Modify | Task 7 + 9 |
| 16 | `notification/wire.go` | Modify | Task 7 |
| 17 | `notification/wire_gen.go` | Regenerate | Task 7 |
| 18 | `order/events/types.go` | Modify | Task 8 |
| 19 | `order/events/consumer.go` | Modify | Task 8 |
| 20 | `order/ioc/kafka.go` | Modify | Task 8 |
| 21 | `order/wire.go` | Modify | Task 8 |
| 22 | `order/wire_gen.go` | Regenerate | Task 8 |
| 23 | `marketing/ioc/kafka.go` | Modify | Task 9 |

Total: 23 files (5 create + 13 modify + 5 regenerate)
