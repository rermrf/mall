package dao

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type CategoryModel struct {
	ID       int64  `gorm:"primaryKey;autoIncrement"`
	TenantID int64  `gorm:"not null;index:idx_tenant_parent"`
	ParentID int64  `gorm:"not null;index:idx_tenant_parent,priority:2"`
	Name     string `gorm:"type:varchar(100);not null"`
	Level    int32  `gorm:"not null"`
	Sort     int32  `gorm:"default:0"`
	Icon     string `gorm:"type:varchar(500)"`
	Status   uint8  `gorm:"default:1;not null"`
	Ctime    int64  `gorm:"not null"`
	Utime    int64  `gorm:"not null"`
}

func (CategoryModel) TableName() string {
	return "categories"
}

type CategoryDAO interface {
	Insert(ctx context.Context, c CategoryModel) (CategoryModel, error)
	Update(ctx context.Context, c CategoryModel) error
	FindAllByTenant(ctx context.Context, tenantId int64) ([]CategoryModel, error)
	FindById(ctx context.Context, id int64) (CategoryModel, error)
	Delete(ctx context.Context, id, tenantId int64) error
	CountChildren(ctx context.Context, parentId, tenantId int64) (int64, error)
}

type GORMCategoryDAO struct {
	db *gorm.DB
}

func NewCategoryDAO(db *gorm.DB) CategoryDAO {
	return &GORMCategoryDAO{db: db}
}

func (d *GORMCategoryDAO) Insert(ctx context.Context, c CategoryModel) (CategoryModel, error) {
	now := time.Now().UnixMilli()
	c.Ctime = now
	c.Utime = now
	err := d.db.WithContext(ctx).Create(&c).Error
	return c, err
}

func (d *GORMCategoryDAO) Update(ctx context.Context, c CategoryModel) error {
	c.Utime = time.Now().UnixMilli()
	return d.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", c.ID, c.TenantID).Updates(&c).Error
}

func (d *GORMCategoryDAO) FindAllByTenant(ctx context.Context, tenantId int64) ([]CategoryModel, error) {
	var categories []CategoryModel
	err := d.db.WithContext(ctx).
		Where("tenant_id = ?", tenantId).
		Order("level ASC, sort ASC, id ASC").
		Find(&categories).Error
	return categories, err
}

func (d *GORMCategoryDAO) FindById(ctx context.Context, id int64) (CategoryModel, error) {
	var c CategoryModel
	err := d.db.WithContext(ctx).Where("id = ?", id).First(&c).Error
	return c, err
}

func (d *GORMCategoryDAO) Delete(ctx context.Context, id, tenantId int64) error {
	return d.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", id, tenantId).Delete(&CategoryModel{}).Error
}

func (d *GORMCategoryDAO) CountChildren(ctx context.Context, parentId, tenantId int64) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&CategoryModel{}).
		Where("parent_id = ? AND tenant_id = ?", parentId, tenantId).Count(&count).Error
	return count, err
}
