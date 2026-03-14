package validatorx

import (
	"regexp"
	"strings"

	"github.com/rermrf/mall/pkg/ginx"
)

// FieldError 单个字段的校验错误
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationError 聚合多字段校验错误
type ValidationError struct {
	Errors []FieldError
}

// New 创建 ValidationError
func New() *ValidationError {
	return &ValidationError{}
}

// Add 添加一条字段错误
func (e *ValidationError) Add(field, message string) {
	e.Errors = append(e.Errors, FieldError{Field: field, Message: message})
}

// HasErrors 是否存在校验错误
func (e *ValidationError) HasErrors() bool {
	return len(e.Errors) > 0
}

// ToResult 转为 ginx.Result（首条错误作为 msg，全部详情放 data）
func (e *ValidationError) ToResult() ginx.Result {
	msg := "参数校验失败"
	if len(e.Errors) > 0 {
		msg = e.Errors[0].Message
	}
	return ginx.Result{
		Code: ginx.CodeValidation,
		Msg:  msg,
		Data: e.Errors,
	}
}

// ==================== 纯校验函数 ====================

var phoneRegexp = regexp.MustCompile(`^1[3-9]\d{9}$`)
var emailRegexp = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// IsValidPhone 校验中国大陆手机号
func IsValidPhone(phone string) bool {
	return phoneRegexp.MatchString(phone)
}

// IsValidEmail 校验邮箱格式
func IsValidEmail(email string) bool {
	return emailRegexp.MatchString(email)
}

// IsPositive 校验正数（> 0）
func IsPositive(val int64) bool {
	return val > 0
}

// IsNonNegative 校验非负数（>= 0）
func IsNonNegative(val int64) bool {
	return val >= 0
}

// IsValidTimeRange 校验时间区间（start < end）
func IsValidTimeRange(start, end int64) bool {
	return start < end
}

// IsNotBlank 校验字符串非空白
func IsNotBlank(s string) bool {
	return strings.TrimSpace(s) != ""
}

// IsLengthBetween 校验字符串长度在 [min, max] 范围内
func IsLengthBetween(s string, min, max int) bool {
	l := len(s)
	return l >= min && l <= max
}

// ==================== 链式便捷方法 ====================

// CheckPhone 校验手机号格式
func (e *ValidationError) CheckPhone(field, phone string) *ValidationError {
	if !IsValidPhone(phone) {
		e.Add(field, field+"格式不正确，请输入11位手机号")
	}
	return e
}

// CheckEmail 校验邮箱格式
func (e *ValidationError) CheckEmail(field, email string) *ValidationError {
	if !IsValidEmail(email) {
		e.Add(field, field+"格式不正确")
	}
	return e
}

// CheckPositive 校验 int64 必须 > 0
func (e *ValidationError) CheckPositive(field string, val int64) *ValidationError {
	if !IsPositive(val) {
		e.Add(field, field+"必须大于0")
	}
	return e
}

// CheckNonNegative 校验 int64 必须 >= 0
func (e *ValidationError) CheckNonNegative(field string, val int64) *ValidationError {
	if !IsNonNegative(val) {
		e.Add(field, field+"不能为负数")
	}
	return e
}

// CheckPositiveInt32 校验 int32 必须 > 0
func (e *ValidationError) CheckPositiveInt32(field string, val int32) *ValidationError {
	if val <= 0 {
		e.Add(field, field+"必须大于0")
	}
	return e
}

// CheckTimeRange 校验时间区间
func (e *ValidationError) CheckTimeRange(startField, endField string, start, end int64) *ValidationError {
	if !IsValidTimeRange(start, end) {
		e.Add(startField, startField+"必须早于"+endField)
	}
	return e
}

// CheckNotBlank 校验字符串非空白
func (e *ValidationError) CheckNotBlank(field, val string) *ValidationError {
	if !IsNotBlank(val) {
		e.Add(field, field+"不能为空")
	}
	return e
}

// CheckLength 校验字符串长度范围
func (e *ValidationError) CheckLength(field, val string, min, max int) *ValidationError {
	if !IsLengthBetween(val, min, max) {
		e.Add(field, field+"长度必须在"+strings.Join([]string{itoa(min), itoa(max)}, "~")+"之间")
	}
	return e
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}
