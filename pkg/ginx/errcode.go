package ginx

import (
	"fmt"
	"strings"
)

// ==================== 错误码常量 ====================
// 格式: HTTP状态码前缀 + 3位序号
// 0 = 成功, 4 = 参数错误, 5 = 系统错误 (wrapper 兜底)

const (
	CodeSuccess = 0
	CodeBadReq  = 4
	CodeSystem  = 5

	// 401 - 认证错误
	CodeUnauthorized      = 401001
	CodeInvalidCredentials = 401002
	CodeSmsCodeFailed     = 401003

	// 403 - 权限错误
	CodeForbidden    = 403001
	CodeAdminOnly    = 403002
	CodeMerchantOnly = 403003

	// 404 - 资源不存在
	CodeUserNotFound     = 404001
	CodeOrderNotFound    = 404002
	CodeProductNotFound  = 404003
	CodeAddressNotFound  = 404004
	CodeCouponNotFound   = 404005
	CodeTemplateNotFound = 404006
	CodeShipmentNotFound = 404007
	CodeTenantNotFound   = 404008

	// 409 - 业务冲突
	CodeDuplicateUser      = 409001
	CodeInsufficientStock  = 409002
	CodeOrderStatusDenied  = 409003
	CodeCouponUnavailable  = 409004
	CodeSeckillUnavailable = 409005
	CodeQuotaExceeded      = 409006
	CodeDuplicateSubmit    = 409007
	CodeRefundDenied       = 409008

	// 422 - 业务验证失败
	CodeUserFrozen       = 422001
	CodeRefundExceedAmt  = 422002
	CodeCategoryHasChild = 422003
	CodeBrandHasProduct  = 422004

	// 422 - 通用参数校验
	CodeValidation       = 422010 // 通用校验失败
	CodeInvalidPhone     = 422011
	CodeInvalidEmail     = 422012
	CodeInvalidPrice     = 422013
	CodeInvalidTimeRange = 422014
	CodeInvalidQuantity  = 422015
)

// ErrMapping 定义 gRPC 错误消息到前端错误码的映射
type ErrMapping struct {
	Contains string // gRPC error message 包含此字符串时匹配
	Code     int    // 返回给前端的错误码
	Msg      string // 返回给前端的消息，为空时使用 Contains 的值
}

// HandleGRPCError 将 gRPC 错误转换为前端响应
// 匹配到已知业务错误 → 返回 (Result{Code, Msg}, nil)
// 未匹配 → 返回 (Result{}, error) 让 wrapper 兜底为 "系统错误"
func HandleGRPCError(err error, context string, mappings ...ErrMapping) (Result, error) {
	msg := err.Error()
	for _, m := range mappings {
		if strings.Contains(msg, m.Contains) {
			display := m.Msg
			if display == "" {
				display = m.Contains
			}
			return Result{Code: m.Code, Msg: display}, nil
		}
	}
	return Result{}, fmt.Errorf("%s: %w", context, err)
}

// HandleRawError 用于 raw ctx.JSON handler 的错误处理
// 返回 (Result, bool): bool=true 表示是业务错误，已填充 Result
// bool=false 表示是系统错误，调用方应返回通用错误
func HandleRawError(err error, mappings ...ErrMapping) (Result, bool) {
	msg := err.Error()
	for _, m := range mappings {
		if strings.Contains(msg, m.Contains) {
			display := m.Msg
			if display == "" {
				display = m.Contains
			}
			return Result{Code: m.Code, Msg: display}, true
		}
	}
	return Result{Code: CodeSystem, Msg: "系统错误"}, false
}

// ==================== 常用映射预设 ====================

// UserErrMappings 用户相关的常用错误映射
var UserErrMappings = []ErrMapping{
	{Contains: "用户已存在", Code: CodeDuplicateUser},
	{Contains: "用户不存在", Code: CodeUserNotFound},
	{Contains: "用户名或密码错误", Code: CodeInvalidCredentials},
	{Contains: "验证码错误或已过期", Code: CodeSmsCodeFailed},
	{Contains: "用户已被冻结", Code: CodeUserFrozen},
}

// OrderErrMappings 订单相关的常用错误映射
var OrderErrMappings = []ErrMapping{
	{Contains: "库存不足", Code: CodeInsufficientStock},
	{Contains: "地址不存在", Code: CodeAddressNotFound},
	{Contains: "请勿重复提交", Code: CodeDuplicateSubmit},
	{Contains: "当前状态不允许取消", Code: CodeOrderStatusDenied, Msg: "当前订单状态不允许取消"},
	{Contains: "当前状态不允许确认收货", Code: CodeOrderStatusDenied, Msg: "当前订单状态不允许确认收货"},
	{Contains: "当前状态不允许退款", Code: CodeOrderStatusDenied, Msg: "当前订单状态不允许退款"},
	{Contains: "无权取消此订单", Code: CodeForbidden, Msg: "无权操作此订单"},
	{Contains: "无权操作此订单", Code: CodeForbidden},
	{Contains: "无权申请退款", Code: CodeForbidden, Msg: "无权操作此订单"},
	{Contains: "退款金额超出可退金额", Code: CodeRefundExceedAmt},
	{Contains: "退款单状态不允许处理", Code: CodeOrderStatusDenied, Msg: "退款单状态不允许此操作"},
}

// MarketingErrMappings 营销相关的常用错误映射
var MarketingErrMappings = []ErrMapping{
	{Contains: "优惠券不存在", Code: CodeCouponNotFound},
	{Contains: "优惠券已过期", Code: CodeCouponUnavailable, Msg: "优惠券已过期"},
	{Contains: "领取次数已达上限", Code: CodeCouponUnavailable, Msg: "领取次数已达上限"},
	{Contains: "秒杀活动未开始或已结束", Code: CodeSeckillUnavailable},
	{Contains: "秒杀库存不足", Code: CodeSeckillUnavailable, Msg: "已抢光"},
	{Contains: "已参与过该秒杀", Code: CodeSeckillUnavailable, Msg: "已参与过该秒杀"},
}

// ProductErrMappings 商品相关的常用错误映射
var ProductErrMappings = []ErrMapping{
	{Contains: "商品不存在", Code: CodeProductNotFound},
	{Contains: "分类不存在", Code: CodeProductNotFound, Msg: "分类不存在"},
	{Contains: "品牌不存在", Code: CodeProductNotFound, Msg: "品牌不存在"},
	{Contains: "商品配额已超限", Code: CodeQuotaExceeded},
	{Contains: "该分类下有子分类", Code: CodeCategoryHasChild},
	{Contains: "该分类下有商品", Code: CodeCategoryHasChild, Msg: "该分类下有商品，不能删除"},
	{Contains: "该品牌下有商品", Code: CodeBrandHasProduct},
	{Contains: "分类层级不能超过3级", Code: CodeBadReq, Msg: "分类层级不能超过3级"},
}

// TenantErrMappings 租户相关的常用错误映射
var TenantErrMappings = []ErrMapping{
	{Contains: "租户不存在", Code: CodeTenantNotFound},
	{Contains: "配额不足", Code: CodeQuotaExceeded, Msg: "配额不足"},
}
