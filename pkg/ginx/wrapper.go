package ginx

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rermrf/emo/logger"
)

// Result 统一响应结构
type Result struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data,omitempty"`
}

// WrapBody 包装带有请求体绑定的 handler，统一错误处理和响应格式
func WrapBody[Req any](l logger.Logger, fn func(ctx *gin.Context, req Req) (Result, error)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req Req
		if err := ctx.Bind(&req); err != nil {
			ctx.JSON(http.StatusBadRequest, Result{Code: 4, Msg: "参数错误"})
			return
		}
		res, err := fn(ctx, req)
		if err != nil {
			l.Error("业务处理错误", logger.Error(err))
			ctx.JSON(http.StatusOK, Result{Code: 5, Msg: "系统错误"})
			return
		}
		ctx.JSON(http.StatusOK, res)
	}
}

// WrapQuery 包装无请求体的 handler（如 GET 请求）
func WrapQuery[Req any](l logger.Logger, fn func(ctx *gin.Context, req Req) (Result, error)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req Req
		if err := ctx.ShouldBindQuery(&req); err != nil {
			ctx.JSON(http.StatusBadRequest, Result{Code: 4, Msg: "参数错误"})
			return
		}
		res, err := fn(ctx, req)
		if err != nil {
			l.Error("业务处理错误", logger.Error(err))
			ctx.JSON(http.StatusOK, Result{Code: 5, Msg: "系统错误"})
			return
		}
		ctx.JSON(http.StatusOK, res)
	}
}
