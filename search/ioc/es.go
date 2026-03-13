package ioc

import (
	"fmt"

	"github.com/olivere/elastic/v7"
	"github.com/rermrf/mall/search/repository/dao"
	"github.com/spf13/viper"
)

func InitES() *elastic.Client {
	type Config struct {
		URL string `yaml:"url"`
	}
	var cfg Config
	err := viper.UnmarshalKey("es", &cfg)
	if err != nil {
		panic(fmt.Errorf("读取 ES 配置失败: %w", err))
	}
	client, err := elastic.NewClient(
		elastic.SetURL(cfg.URL),
		elastic.SetSniff(false),
	)
	if err != nil {
		panic(fmt.Errorf("连接 ES 失败: %w", err))
	}
	err = dao.InitIndex(client)
	if err != nil {
		panic(fmt.Errorf("ES 索引初始化失败: %w", err))
	}
	return client
}
