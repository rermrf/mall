package ioc

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/rermrf/emo/logger"
	tenantv1 "github.com/rermrf/mall/api/proto/gen/tenant/v1"
	"github.com/rermrf/mall/consumer-bff/handler"
	ijwt "github.com/rermrf/mall/consumer-bff/handler/jwt"
	"github.com/rermrf/mall/consumer-bff/handler/middleware"
	"github.com/rermrf/mall/pkg/ginx"
)

func InitGinServer(
	jwtHandler *ijwt.JWTHandler,
	userHandler *handler.UserHandler,
	tenantHandler *handler.TenantHandler,
	inventoryHandler *handler.InventoryHandler,
	orderHandler *handler.OrderHandler,
	paymentHandler *handler.PaymentHandler,
	cartHandler *handler.CartHandler,
	searchHandler *handler.SearchHandler,
	marketingHandler *handler.MarketingHandler,
	logisticsHandler *handler.LogisticsHandler,
	notificationHandler *handler.NotificationHandler,
	productHandler *handler.ProductHandler,
	tenantClient tenantv1.TenantServiceClient,
	redisClient redis.Cmdable,
	l logger.Logger,
) *gin.Engine {
	engine := gin.Default()
	engine.Use(ginx.DefaultCORS())

	// Health check (before tenant resolve middleware)
	engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	engine.Use(middleware.NewTenantResolve(tenantClient, redisClient).Build())

	pub := engine.Group("/api/v1")
	{
		pub.POST("/signup", ginx.WrapBody[handler.SignupReq](l, userHandler.Signup))
		pub.POST("/login", ginx.WrapBody[ijwt.LoginReq](l, jwtHandler.Login))
		pub.POST("/sms/send", ginx.WrapBody[handler.SendSmsCodeReq](l, userHandler.SendSmsCode))
		pub.POST("/login/phone", ginx.WrapBody[ijwt.LoginByPhoneReq](l, jwtHandler.LoginByPhone))
		pub.POST("/login/oauth", ginx.WrapBody[ijwt.OAuthLoginReq](l, jwtHandler.OAuthLogin))
		pub.GET("/shop", tenantHandler.GetShop)
		// 搜索（公开）
		pub.GET("/search", ginx.WrapQuery[handler.SearchReq](l, searchHandler.Search))
		pub.GET("/search/suggestions", searchHandler.GetSuggestions)
		pub.GET("/search/hot", searchHandler.GetHotWords)
		// 营销（公开）
		pub.GET("/coupons", marketingHandler.ListAvailableCoupons)
		pub.GET("/seckill", marketingHandler.ListSeckillActivities)
		// 商品（公开）
		pub.GET("/products/:id", productHandler.GetProduct)
		pub.GET("/categories", productHandler.ListCategories)
		pub.GET("/products", ginx.WrapQuery[handler.ListProductsReq](l, productHandler.ListProducts))
	}

	auth := engine.Group("/api/v1")
	auth.Use(middleware.NewLoginJWTBuilder(jwtHandler).Build())
	{
		auth.POST("/logout", ginx.WrapBody[ijwt.LogoutReq](l, jwtHandler.Logout))
		auth.POST("/refresh-token", ginx.WrapBody[ijwt.RefreshReq](l, jwtHandler.Refresh))
		auth.GET("/profile", userHandler.GetProfile)
		auth.PUT("/profile", ginx.WrapBody[handler.UpdateProfileReq](l, userHandler.UpdateProfile))
		auth.GET("/addresses", userHandler.ListAddresses)
		auth.POST("/addresses", ginx.WrapBody[handler.CreateAddressReq](l, userHandler.CreateAddress))
		auth.PUT("/addresses/:id", ginx.WrapBody[handler.UpdateAddressReq](l, userHandler.UpdateAddress))
		auth.DELETE("/addresses/:id", userHandler.DeleteAddress)
		// 库存查询
		auth.GET("/inventory/stock/:skuId", inventoryHandler.GetStock)
		auth.POST("/inventory/stock/batch", ginx.WrapBody[handler.BatchGetStockReq](l, inventoryHandler.BatchGetStock))
		// 订单
		auth.POST("/orders", ginx.WrapBody[handler.CreateOrderReq](l, orderHandler.CreateOrder))
		auth.GET("/orders", ginx.WrapQuery[handler.ListOrdersReq](l, orderHandler.ListOrders))
		auth.GET("/orders/:orderNo", orderHandler.GetOrder)
		auth.POST("/orders/:orderNo/cancel", orderHandler.CancelOrder)
		auth.POST("/orders/:orderNo/confirm", orderHandler.ConfirmReceive)
		auth.POST("/orders/:orderNo/refund", ginx.WrapBody[handler.ApplyRefundReq](l, orderHandler.ApplyRefund))
		auth.GET("/refunds", ginx.WrapQuery[handler.ListRefundsReq](l, orderHandler.ListRefundOrders))
		auth.GET("/refunds/:refundNo", orderHandler.GetRefundOrder)
		auth.POST("/refunds/:refundNo/cancel", orderHandler.CancelRefund)
		// 支付
		auth.POST("/payments", ginx.WrapBody[handler.CreatePaymentReq](l, paymentHandler.CreatePayment))
		auth.GET("/payments/:paymentNo", paymentHandler.GetPayment)
		auth.POST("/payments/notify", ginx.WrapBody[handler.HandleNotifyReq](l, paymentHandler.HandleNotify))
		// 购物车
		auth.POST("/cart/items", ginx.WrapBody[handler.AddCartItemReq](l, cartHandler.AddItem))
		auth.PUT("/cart/items/:skuId", ginx.WrapBody[handler.UpdateCartItemReq](l, cartHandler.UpdateItem))
		auth.DELETE("/cart/items/:skuId", cartHandler.RemoveItem)
		auth.GET("/cart", cartHandler.GetCart)
		auth.DELETE("/cart", cartHandler.ClearCart)
		auth.POST("/cart/batch-remove", ginx.WrapBody[handler.BatchRemoveReq](l, cartHandler.BatchRemove))
		// 搜索历史
		auth.GET("/search/history", searchHandler.GetSearchHistory)
		auth.DELETE("/search/history", searchHandler.ClearSearchHistory)
		// 营销（需登录）
		auth.POST("/coupons/:id/receive", marketingHandler.ReceiveCoupon)
		auth.GET("/coupons/mine", marketingHandler.ListMyCoupons)
		auth.POST("/seckill/:itemId", marketingHandler.Seckill)
		// 物流查询
		auth.GET("/orders/:orderNo/logistics", logisticsHandler.GetOrderLogistics)
		// 通知
		auth.GET("/notifications", ginx.WrapQuery[handler.ListNotificationsReq](l, notificationHandler.ListNotifications))
		auth.GET("/notifications/unread-count", notificationHandler.GetUnreadCount)
		auth.PUT("/notifications/:id/read", notificationHandler.MarkRead)
		auth.PUT("/notifications/read-all", notificationHandler.MarkAllRead)
		auth.DELETE("/notifications/:id", notificationHandler.DeleteNotification)
	}

	return engine
}
