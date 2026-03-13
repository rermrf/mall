package ioc

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rermrf/emo/logger"
	"github.com/rermrf/mall/admin-bff/handler"
	ijwt "github.com/rermrf/mall/admin-bff/handler/jwt"
	"github.com/rermrf/mall/admin-bff/handler/middleware"
	"github.com/rermrf/mall/pkg/ginx"
)

func InitGinServer(
	jwtHandler *ijwt.JWTHandler,
	userHandler *handler.UserHandler,
	tenantHandler *handler.TenantHandler,
	productHandler *handler.ProductHandler,
	orderHandler *handler.OrderHandler,
	paymentHandler *handler.PaymentHandler,
	notificationHandler *handler.NotificationHandler,
	inventoryHandler *handler.InventoryHandler,
	marketingHandler *handler.MarketingHandler,
	logisticsHandler *handler.LogisticsHandler,
	l logger.Logger,
) *gin.Engine {
	engine := gin.Default()
	engine.Use(ginx.DefaultCORS())

	// Health check (before auth middleware)
	engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	loginJWT := middleware.NewLoginJWTBuilder(jwtHandler).Build()
	adminOnly := middleware.AdminOnly()

	// 公开路由
	pub := engine.Group("/api/v1")
	{
		pub.POST("/login", ginx.WrapBody[ijwt.LoginReq](l, jwtHandler.Login))
	}

	// 需要认证 + 管理员权限的路由
	auth := engine.Group("/api/v1")
	auth.Use(loginJWT, adminOnly)
	{
		auth.POST("/logout", ginx.WrapBody[ijwt.LogoutReq](l, jwtHandler.Logout))
		auth.POST("/refresh-token", ginx.WrapBody[ijwt.RefreshReq](l, jwtHandler.Refresh))

		// 用户管理
		auth.GET("/users", ginx.WrapQuery[handler.ListUsersReq](l, userHandler.ListUsers))
		auth.POST("/users/:id/status", ginx.WrapBody[handler.UpdateUserStatusReq](l, userHandler.UpdateUserStatus))

		// 角色管理
		auth.GET("/roles", ginx.WrapQuery[handler.ListRolesReq](l, userHandler.ListRoles))
		auth.POST("/roles", ginx.WrapBody[handler.CreateRoleReq](l, userHandler.CreateRole))
		auth.PUT("/roles/:id", ginx.WrapBody[handler.UpdateRoleReq](l, userHandler.UpdateRole))

		// 租户管理
		auth.POST("/tenants", ginx.WrapBody[handler.CreateTenantReq](l, tenantHandler.CreateTenant))
		auth.GET("/tenants", ginx.WrapQuery[handler.ListTenantsReq](l, tenantHandler.ListTenants))
		auth.GET("/tenants/:id", tenantHandler.GetTenant)
		auth.POST("/tenants/:id/approve", ginx.WrapBody[handler.ApproveTenantReq](l, tenantHandler.ApproveTenant))
		auth.POST("/tenants/:id/freeze", ginx.WrapBody[handler.FreezeTenantReq](l, tenantHandler.FreezeTenant))

		// 套餐管理
		auth.GET("/plans", ginx.WrapQuery[handler.ListPlansReq](l, tenantHandler.ListPlans))
		auth.POST("/plans", ginx.WrapBody[handler.CreatePlanReq](l, tenantHandler.CreatePlan))
		auth.PUT("/plans/:id", ginx.WrapBody[handler.UpdatePlanReq](l, tenantHandler.UpdatePlan))

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
		auth.DELETE("/notification-templates/:id", func(c *gin.Context) {
			res, err := notificationHandler.DeleteTemplate(c)
			if err != nil {
				l.Error("删除通知模板失败", logger.Error(err))
				c.JSON(http.StatusOK, ginx.Result{Code: 5, Msg: "系统错误"})
				return
			}
			c.JSON(http.StatusOK, res)
		})
		// 发送通知
		auth.POST("/notifications/send", ginx.WrapBody[handler.SendNotificationReq](l, notificationHandler.SendNotification))

		// 库存监管
		auth.GET("/inventory/:skuId", inventoryHandler.GetStock)
		auth.POST("/inventory/batch", ginx.WrapBody[handler.AdminBatchGetStockReq](l, inventoryHandler.BatchGetStock))
		auth.GET("/inventory/logs", ginx.WrapQuery[handler.AdminListLogsReq](l, inventoryHandler.ListLogs))

		// 营销监管
		auth.GET("/coupons", ginx.WrapQuery[handler.AdminListCouponsReq](l, marketingHandler.ListCoupons))
		auth.GET("/seckill", ginx.WrapQuery[handler.AdminListSeckillReq](l, marketingHandler.ListSeckill))
		auth.GET("/seckill/:id", marketingHandler.GetSeckill)
		auth.GET("/promotions", ginx.WrapQuery[handler.AdminListPromotionsReq](l, marketingHandler.ListPromotions))

		// 物流监管
		auth.GET("/freight-templates", ginx.WrapQuery[handler.AdminListFreightTemplatesReq](l, logisticsHandler.ListFreightTemplates))
		auth.GET("/freight-templates/:id", logisticsHandler.GetFreightTemplate)
		auth.GET("/shipments/:id", logisticsHandler.GetShipment)
		auth.GET("/orders/:orderNo/logistics", logisticsHandler.GetOrderLogistics)
	}

	return engine
}
