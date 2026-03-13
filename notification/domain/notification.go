package domain

import "time"

// ==================== 通知模板 ====================

type NotificationTemplate struct {
	ID       int64
	TenantID int64  // 0=平台模板
	Code     string // 模板编码：welcome_sms, order_paid_merchant 等
	Channel  int32  // 1-短信 2-邮件 3-站内信
	Title    string
	Content  string // 模板内容，支持 Go text/template 占位符
	Status   int32  // 1-启用 2-停用
	Ctime    time.Time
	Utime    time.Time
}

// ==================== 通知记录 ====================

type Notification struct {
	ID       int64
	UserID   int64
	TenantID int64
	Channel  int32 // 1-短信 2-邮件 3-站内信
	Title    string
	Content  string
	IsRead   bool
	Status   int32 // 1-待发送 2-已发送 3-发送失败
	Ctime    time.Time
	Utime    time.Time
}
