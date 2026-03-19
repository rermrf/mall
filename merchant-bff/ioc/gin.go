package ioc

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rermrf/emo/logger"
	"github.com/rermrf/mall/merchant-bff/handler"
	ijwt "github.com/rermrf/mall/merchant-bff/handler/jwt"
	"github.com/rermrf/mall/merchant-bff/handler/middleware"
	"github.com/rermrf/mall/pkg/ginx"
)

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
	accountHandler *handler.AccountHandler,
	l logger.Logger,
) *gin.Engine {
	engine := gin.Default()
	engine.Use(ginx.DefaultCORS())

	// Health check (before auth middleware)
	engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	loginJWT := middleware.NewLoginJWTBuilder(jwtHandler).Build()
	tenantExtract := middleware.TenantExtract()

	// 公开路由
	pub := engine.Group("/api/v1")
	{
		pub.POST("/login", ginx.WrapBody[ijwt.LoginReq](l, jwtHandler.Login))
		pub.POST("/refresh-token", ginx.WrapBody[ijwt.RefreshReq](l, jwtHandler.Refresh))
	}

	// 需要认证 + 商家身份的路由
	auth := engine.Group("/api/v1")
	auth.Use(loginJWT, tenantExtract)
	{
		auth.POST("/logout", ginx.WrapBody[ijwt.LogoutReq](l, jwtHandler.Logout))

		// 个人信息
		auth.GET("/profile", userHandler.GetProfile)
		auth.PUT("/profile", ginx.WrapBody[handler.UpdateProfileReq](l, userHandler.UpdateProfile))

		// 员工管理
		auth.GET("/staff", ginx.WrapQuery[handler.ListStaffReq](l, userHandler.ListStaff))
		auth.POST("/staff/:id/role", ginx.WrapBody[handler.AssignRoleReq](l, userHandler.AssignRole))

		// 角色管理
		auth.GET("/roles", ginx.WrapQuery[handler.ListRolesReq](l, userHandler.ListRoles))
		auth.POST("/roles", ginx.WrapBody[handler.CreateRoleReq](l, userHandler.CreateRole))
		auth.PUT("/roles/:id", ginx.WrapBody[handler.UpdateRoleReq](l, userHandler.UpdateRole))

		// 店铺管理
		auth.GET("/shop", tenantHandler.GetShop)
		auth.PUT("/shop", ginx.WrapBody[handler.UpdateShopReq](l, tenantHandler.UpdateShop))

		// 配额查询
		auth.GET("/quotas/:type", tenantHandler.CheckQuota)

		// 库存管理
		auth.POST("/inventory/stock", ginx.WrapBody[handler.SetStockReq](l, inventoryHandler.SetStock))
		auth.GET("/inventory/stock/:skuId", inventoryHandler.GetStock)
		auth.POST("/inventory/stock/batch", ginx.WrapBody[handler.BatchGetStockReq](l, inventoryHandler.BatchGetStock))
		auth.GET("/inventory/logs", ginx.WrapQuery[handler.ListLogsReq](l, inventoryHandler.ListLogs))

		// 订单管理
		auth.GET("/orders", ginx.WrapQuery[handler.ListOrdersReq](l, orderHandler.ListOrders))
		auth.GET("/orders/:orderNo", orderHandler.GetOrder)
		auth.POST("/orders/:orderNo/ship", ginx.WrapBody[handler.ShipOrderReq](l, logisticsHandler.ShipOrder))
		auth.POST("/orders/:orderNo/refund/handle", ginx.WrapBody[handler.HandleRefundReq](l, orderHandler.HandleRefund))
		auth.GET("/refunds", ginx.WrapQuery[handler.ListRefundsReq](l, orderHandler.ListRefundOrders))

		// 支付管理
		auth.GET("/payments", ginx.WrapQuery[handler.ListPaymentsReq](l, paymentHandler.ListPayments))
		auth.GET("/payments/:paymentNo", paymentHandler.GetPayment)
		auth.POST("/payments/:paymentNo/refund", ginx.WrapBody[handler.RefundReq](l, paymentHandler.Refund))
		auth.GET("/refunds/:refundNo/payment", paymentHandler.GetRefund)

		// 营销管理 - 优惠券
		auth.POST("/coupons", ginx.WrapBody[handler.CreateCouponReq](l, marketingHandler.CreateCoupon))
		auth.PUT("/coupons/:id", ginx.WrapBody[handler.UpdateCouponReq](l, marketingHandler.UpdateCoupon))
		auth.GET("/coupons/:id", marketingHandler.GetCoupon)
		auth.GET("/coupons", ginx.WrapQuery[handler.ListCouponsReq](l, marketingHandler.ListCoupons))
		// 营销管理 - 秒杀
		auth.POST("/seckill", ginx.WrapBody[handler.CreateSeckillReq](l, marketingHandler.CreateSeckill))
		auth.PUT("/seckill/:id", ginx.WrapBody[handler.UpdateSeckillReq](l, marketingHandler.UpdateSeckill))
		auth.GET("/seckill", ginx.WrapQuery[handler.ListSeckillReq](l, marketingHandler.ListSeckill))
		auth.GET("/seckill/:id", marketingHandler.GetSeckill)
		// 营销管理 - 满减规则
		auth.POST("/promotions", ginx.WrapBody[handler.CreatePromotionReq](l, marketingHandler.CreatePromotion))
		auth.PUT("/promotions/:id", ginx.WrapBody[handler.UpdatePromotionReq](l, marketingHandler.UpdatePromotion))
		auth.GET("/promotions", ginx.WrapQuery[handler.ListPromotionsReq](l, marketingHandler.ListPromotions))

		// 物流查询
		auth.GET("/orders/:orderNo/logistics", logisticsHandler.GetOrderLogistics)
		// 运费模板管理
		auth.POST("/freight-templates", ginx.WrapBody[handler.CreateFreightTemplateReq](l, logisticsHandler.CreateFreightTemplate))
		auth.PUT("/freight-templates/:id", ginx.WrapBody[handler.UpdateFreightTemplateReq](l, logisticsHandler.UpdateFreightTemplate))
		auth.GET("/freight-templates/:id", logisticsHandler.GetFreightTemplate)
		auth.GET("/freight-templates", logisticsHandler.ListFreightTemplates)
		auth.DELETE("/freight-templates/:id", logisticsHandler.DeleteFreightTemplate)

		// 通知
		auth.GET("/notifications", ginx.WrapQuery[handler.ListNotificationsReq](l, notificationHandler.ListNotifications))
		auth.GET("/notifications/unread-count", notificationHandler.GetUnreadCount)
		auth.PUT("/notifications/:id/read", notificationHandler.MarkRead)
		auth.PUT("/notifications/read-all", notificationHandler.MarkAllRead)

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

		// 账户管理
		auth.GET("/account", accountHandler.GetAccount)
		auth.PUT("/account/bank-info", ginx.WrapBody[handler.UpdateBankInfoReq](l, accountHandler.UpdateBankInfo))
		auth.GET("/account/summary", accountHandler.GetAccountSummary)
		auth.GET("/settlements", ginx.WrapQuery[handler.ListSettlementsReq](l, accountHandler.ListSettlements))
		auth.POST("/withdrawals", ginx.WrapBody[handler.RequestWithdrawalReq](l, accountHandler.RequestWithdrawal))
		auth.GET("/withdrawals", ginx.WrapQuery[handler.ListWithdrawalsReq](l, accountHandler.ListWithdrawals))
		auth.GET("/transactions", ginx.WrapQuery[handler.ListTransactionsReq](l, accountHandler.ListTransactions))
	}

	return engine
}
