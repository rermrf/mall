package dao

import (
	"context"
	"fmt"

	"github.com/olivere/elastic/v7"
)

const mallProductMapping = `{
	"mappings": {
		"properties": {
			"id":            { "type": "long" },
			"tenant_id":     { "type": "long" },
			"name":          { "type": "text", "analyzer": "ik_max_word", "search_analyzer": "ik_smart" },
			"subtitle":      { "type": "text", "analyzer": "ik_smart" },
			"category_id":   { "type": "long" },
			"category_name": { "type": "keyword" },
			"brand_id":      { "type": "long" },
			"brand_name":    { "type": "keyword" },
			"price":         { "type": "long" },
			"sales":         { "type": "long" },
			"main_image":    { "type": "keyword" },
			"status":        { "type": "integer" },
			"shop_id":       { "type": "long" },
			"shop_name":     { "type": "keyword" }
		}
	}
}`

func InitIndex(client *elastic.Client) error {
	ctx := context.Background()
	exists, err := client.IndexExists("mall_product").Do(ctx)
	if err != nil {
		return fmt.Errorf("检查索引是否存在失败: %w", err)
	}
	if !exists {
		_, err = client.CreateIndex("mall_product").Body(mallProductMapping).Do(ctx)
		if err != nil {
			return fmt.Errorf("创建索引失败: %w", err)
		}
	}
	return nil
}
