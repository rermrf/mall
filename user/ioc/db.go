package ioc

import (
	"fmt"

	"github.com/rermrf/mall/user/repository/dao"
	"github.com/rermrf/mall/pkg/gormx"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func InitDB() *gorm.DB {
	type Config struct {
		DSN string `yaml:"dsn"`
	}
	var cfg Config
	err := viper.UnmarshalKey("db", &cfg)
	if err != nil {
		panic(fmt.Errorf("读取数据库配置失败: %w", err))
	}
	db, err := gorm.Open(mysql.Open(cfg.DSN), &gorm.Config{})
	if err != nil {
		panic(fmt.Errorf("连接数据库失败: %w", err))
	}
	//err = db.Use(gormPrometheus.New(gormPrometheus.Config{
	//	DBName:          "mall_user",
	//	RefreshInterval: 15,
	//	MetricsCollector: []gormPrometheus.MetricsCollector{
	//		&gormPrometheus.MySQL{},
	//	},
	//}))
	//if err != nil {
	//	panic(fmt.Errorf("初始化 GORM Prometheus 插件失败: %w", err))
	//}
	err = dao.InitTables(db)
	if err != nil {
		panic(fmt.Errorf("数据库表初始化失败: %w", err))
	}
	gormx.RegisterTenantPlugin(db)
	return db
}
