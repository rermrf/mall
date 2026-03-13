package domain

// Permission 权限
type Permission struct {
	ID       int64
	Code     string // 如 product:create, order:view
	Name     string
	Type     PermissionType
	Resource string // 资源标识
	Ctime    int64
	Utime    int64
}

type PermissionType uint8

const (
	PermissionTypeMenu   PermissionType = iota + 1 // 菜单
	PermissionTypeButton                           // 按钮
	PermissionTypeAPI                              // API
)
