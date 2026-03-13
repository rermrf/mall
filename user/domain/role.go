package domain

// Role 角色
type Role struct {
	ID          int64
	TenantID    int64  // 0=平台角色，其他=商家自定义角色
	Name        string
	Code        string
	Description string
	Ctime       int64
	Utime       int64
}

// UserRole 用户-角色关联
type UserRole struct {
	ID       int64
	UserID   int64
	TenantID int64
	RoleID   int64
	Ctime    int64
	Utime    int64
}

// RolePermission 角色-权限关联
type RolePermission struct {
	ID           int64
	RoleID       int64
	PermissionID int64
	Ctime        int64
}
