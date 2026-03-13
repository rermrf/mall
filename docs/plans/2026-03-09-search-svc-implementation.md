# Search Service + Consumer BFF Search 接口实现计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 实现 search-svc 搜索微服务（7 个 gRPC RPC + 2 个 Kafka Consumer）+ consumer-bff 搜索 HTTP 接口（5 个端点）。基于 Elasticsearch 实现全文搜索。

**Architecture:** DDD 分层（domain → dao(ES) → cache(Redis) → repository → service → grpc → events → ioc → wire）。ES 做搜索存储，Redis 做热搜词和搜索历史，Kafka Consumer 消费 product-svc 事件同步商品数据到 ES。consumer-bff 提供搜索/建议/热搜（公开）+ 搜索历史（需登录）。

**Tech Stack:** Go, gRPC, Elasticsearch (olivere/elastic/v7), Redis, Kafka (Sarama), Wire DI, etcd, Gin (BFF)

---

## Task 1: Domain + ES DAO + Init（search-svc 数据层）

**Files:**
- Create: `search/domain/search.go`
- Create: `search/repository/dao/search.go`
- Create: `search/repository/dao/init.go`

**Step 1: 创建 domain 模型**

Create `search/domain/search.go`:

```go
package domain

type ProductDocument struct {
	ID           int64
	TenantID     int64
	Name         string
	Subtitle     string
	CategoryID   int64
	CategoryName string
	BrandID      int64
	BrandName    string
	Price        int64 // 分
	Sales        int64
	MainImage    string
	Status       int32
	ShopID       int64
	ShopName     string
}

type HotWord struct {
	Word  string
	Count int64
}

type SearchHistory struct {
	Keyword string
	Ctime   int64
}
```

**Step 2: 创建 ES DAO**

Create `search/repository/dao/search.go`:

```go
package dao

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/olivere/elastic/v7"
)

type ProductDoc struct {
	ID           int64  `json:"id"`
	TenantID     int64  `json:"tenant_id"`
	Name         string `json:"name"`
	Subtitle     string `json:"subtitle"`
	CategoryID   int64  `json:"category_id"`
	CategoryName string `json:"category_name"`
	BrandID      int64  `json:"brand_id"`
	BrandName    string `json:"brand_name"`
	Price        int64  `json:"price"`
	Sales        int64  `json:"sales"`
	MainImage    string `json:"main_image"`
	Status       int32  `json:"status"`
	ShopID       int64  `json:"shop_id"`
	ShopName     string `json:"shop_name"`
}

type SearchDAO interface {
	IndexProduct(ctx context.Context, doc ProductDoc) error
	DeleteProduct(ctx context.Context, id int64) error
	SearchProducts(ctx context.Context, query elastic.Query, sortFields []elastic.Sorter, from, size int) ([]ProductDoc, int64, error)
	Suggest(ctx context.Context, prefix string, limit int) ([]string, error)
}

type ElasticSearchDAO struct {
	client    *elastic.Client
	indexName string
}

func NewSearchDAO(client *elastic.Client) SearchDAO {
	return &ElasticSearchDAO{
		client:    client,
		indexName: "mall_product",
	}
}

func (d *ElasticSearchDAO) IndexProduct(ctx context.Context, doc ProductDoc) error {
	_, err := d.client.Index().
		Index(d.indexName).
		Id(strconv.FormatInt(doc.ID, 10)).
		BodyJson(doc).
		Do(ctx)
	return err
}

func (d *ElasticSearchDAO) DeleteProduct(ctx context.Context, id int64) error {
	_, err := d.client.Delete().
		Index(d.indexName).
		Id(strconv.FormatInt(id, 10)).
		Do(ctx)
	if elastic.IsNotFound(err) {
		return nil
	}
	return err
}

func (d *ElasticSearchDAO) SearchProducts(ctx context.Context, query elastic.Query, sortFields []elastic.Sorter, from, size int) ([]ProductDoc, int64, error) {
	search := d.client.Search().
		Index(d.indexName).
		Query(query).
		From(from).
		Size(size)
	for _, s := range sortFields {
		search = search.SortBy(s)
	}
	result, err := search.Do(ctx)
	if err != nil {
		return nil, 0, err
	}
	docs := make([]ProductDoc, 0, len(result.Hits.Hits))
	for _, hit := range result.Hits.Hits {
		var doc ProductDoc
		if err := json.Unmarshal(hit.Source, &doc); err != nil {
			continue
		}
		docs = append(docs, doc)
	}
	return docs, result.TotalHits(), nil
}

func (d *ElasticSearchDAO) Suggest(ctx context.Context, prefix string, limit int) ([]string, error) {
	// 使用 match_phrase_prefix 查询 name 字段
	query := elastic.NewMatchPhrasePrefixQuery("name", prefix)
	result, err := d.client.Search().
		Index(d.indexName).
		Query(query).
		Size(limit).
		FetchSourceContext(elastic.NewFetchSourceContext(true).Include("name")).
		Do(ctx)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{})
	suggestions := make([]string, 0, limit)
	for _, hit := range result.Hits.Hits {
		var doc struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(hit.Source, &doc); err != nil {
			continue
		}
		if _, ok := seen[doc.Name]; !ok {
			seen[doc.Name] = struct{}{}
			suggestions = append(suggestions, doc.Name)
		}
	}
	return suggestions, nil
}
```

**Step 3: 创建 ES 索引初始化**

Create `search/repository/dao/init.go`:

```go
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
```

**Step 4: 验证编译**

```bash
go get github.com/olivere/elastic/v7
go build ./search/...
```

---

## Task 2: Cache + Repository（search-svc 缓存与仓储层）

**Files:**
- Create: `search/repository/cache/search.go`
- Create: `search/repository/search.go`

**Step 1: 创建 Redis 缓存（热搜词 + 搜索历史）**

Create `search/repository/cache/search.go`:

```go
package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type SearchCache interface {
	IncrHotWord(ctx context.Context, keyword string) error
	GetHotWords(ctx context.Context, limit int) ([]redis.Z, error)
	AddSearchHistory(ctx context.Context, userId int64, keyword string) error
	GetSearchHistory(ctx context.Context, userId int64, limit int) ([]string, error)
	ClearSearchHistory(ctx context.Context, userId int64) error
}

type RedisSearchCache struct {
	client redis.Cmdable
}

func NewSearchCache(client redis.Cmdable) SearchCache {
	return &RedisSearchCache{client: client}
}

const hotWordKey = "search:hot"

func historyKey(userId int64) string {
	return fmt.Sprintf("search:history:%d", userId)
}

func (c *RedisSearchCache) IncrHotWord(ctx context.Context, keyword string) error {
	return c.client.ZIncrBy(ctx, hotWordKey, 1, keyword).Err()
}

func (c *RedisSearchCache) GetHotWords(ctx context.Context, limit int) ([]redis.Z, error) {
	return c.client.ZRevRangeWithScores(ctx, hotWordKey, 0, int64(limit-1)).Result()
}

func (c *RedisSearchCache) AddSearchHistory(ctx context.Context, userId int64, keyword string) error {
	key := historyKey(userId)
	pipe := c.client.Pipeline()
	pipe.LRem(ctx, key, 0, keyword)
	pipe.LPush(ctx, key, keyword)
	pipe.LTrim(ctx, key, 0, 19)
	pipe.Expire(ctx, key, 30*24*time.Hour)
	_, err := pipe.Exec(ctx)
	return err
}

func (c *RedisSearchCache) GetSearchHistory(ctx context.Context, userId int64, limit int) ([]string, error) {
	return c.client.LRange(ctx, historyKey(userId), 0, int64(limit-1)).Result()
}

func (c *RedisSearchCache) ClearSearchHistory(ctx context.Context, userId int64) error {
	return c.client.Del(ctx, historyKey(userId)).Err()
}
```

**Step 2: 创建 Repository**

Create `search/repository/search.go`:

```go
package repository

import (
	"context"

	"github.com/olivere/elastic/v7"

	"github.com/rermrf/mall/search/domain"
	"github.com/rermrf/mall/search/repository/cache"
	"github.com/rermrf/mall/search/repository/dao"
)

type SearchRepository interface {
	SyncProduct(ctx context.Context, doc domain.ProductDocument) error
	DeleteProduct(ctx context.Context, id int64) error
	SearchProducts(ctx context.Context, query elastic.Query, sortFields []elastic.Sorter, from, size int) ([]domain.ProductDocument, int64, error)
	Suggest(ctx context.Context, prefix string, limit int) ([]string, error)
	IncrHotWord(ctx context.Context, keyword string) error
	GetHotWords(ctx context.Context, limit int) ([]domain.HotWord, error)
	AddSearchHistory(ctx context.Context, userId int64, keyword string) error
	GetSearchHistory(ctx context.Context, userId int64, limit int) ([]domain.SearchHistory, error)
	ClearSearchHistory(ctx context.Context, userId int64) error
}

type searchRepository struct {
	dao   dao.SearchDAO
	cache cache.SearchCache
}

func NewSearchRepository(d dao.SearchDAO, c cache.SearchCache) SearchRepository {
	return &searchRepository{dao: d, cache: c}
}

func (r *searchRepository) SyncProduct(ctx context.Context, doc domain.ProductDocument) error {
	return r.dao.IndexProduct(ctx, r.toDAO(doc))
}

func (r *searchRepository) DeleteProduct(ctx context.Context, id int64) error {
	return r.dao.DeleteProduct(ctx, id)
}

func (r *searchRepository) SearchProducts(ctx context.Context, query elastic.Query, sortFields []elastic.Sorter, from, size int) ([]domain.ProductDocument, int64, error) {
	docs, total, err := r.dao.SearchProducts(ctx, query, sortFields, from, size)
	if err != nil {
		return nil, 0, err
	}
	items := make([]domain.ProductDocument, 0, len(docs))
	for _, d := range docs {
		items = append(items, r.toDomain(d))
	}
	return items, total, nil
}

func (r *searchRepository) Suggest(ctx context.Context, prefix string, limit int) ([]string, error) {
	return r.dao.Suggest(ctx, prefix, limit)
}

func (r *searchRepository) IncrHotWord(ctx context.Context, keyword string) error {
	return r.cache.IncrHotWord(ctx, keyword)
}

func (r *searchRepository) GetHotWords(ctx context.Context, limit int) ([]domain.HotWord, error) {
	zs, err := r.cache.GetHotWords(ctx, limit)
	if err != nil {
		return nil, err
	}
	words := make([]domain.HotWord, 0, len(zs))
	for _, z := range zs {
		words = append(words, domain.HotWord{
			Word:  z.Member.(string),
			Count: int64(z.Score),
		})
	}
	return words, nil
}

func (r *searchRepository) AddSearchHistory(ctx context.Context, userId int64, keyword string) error {
	return r.cache.AddSearchHistory(ctx, userId, keyword)
}

func (r *searchRepository) GetSearchHistory(ctx context.Context, userId int64, limit int) ([]domain.SearchHistory, error) {
	keywords, err := r.cache.GetSearchHistory(ctx, userId, limit)
	if err != nil {
		return nil, err
	}
	histories := make([]domain.SearchHistory, 0, len(keywords))
	for _, kw := range keywords {
		histories = append(histories, domain.SearchHistory{Keyword: kw})
	}
	return histories, nil
}

func (r *searchRepository) ClearSearchHistory(ctx context.Context, userId int64) error {
	return r.cache.ClearSearchHistory(ctx, userId)
}

func (r *searchRepository) toDAO(doc domain.ProductDocument) dao.ProductDoc {
	return dao.ProductDoc{
		ID:           doc.ID,
		TenantID:     doc.TenantID,
		Name:         doc.Name,
		Subtitle:     doc.Subtitle,
		CategoryID:   doc.CategoryID,
		CategoryName: doc.CategoryName,
		BrandID:      doc.BrandID,
		BrandName:    doc.BrandName,
		Price:        doc.Price,
		Sales:        doc.Sales,
		MainImage:    doc.MainImage,
		Status:       doc.Status,
		ShopID:       doc.ShopID,
		ShopName:     doc.ShopName,
	}
}

func (r *searchRepository) toDomain(doc dao.ProductDoc) domain.ProductDocument {
	return domain.ProductDocument{
		ID:           doc.ID,
		TenantID:     doc.TenantID,
		Name:         doc.Name,
		Subtitle:     doc.Subtitle,
		CategoryID:   doc.CategoryID,
		CategoryName: doc.CategoryName,
		BrandID:      doc.BrandID,
		BrandName:    doc.BrandName,
		Price:        doc.Price,
		Sales:        doc.Sales,
		MainImage:    doc.MainImage,
		Status:       doc.Status,
		ShopID:       doc.ShopID,
		ShopName:     doc.ShopName,
	}
}
```

**Step 3: 验证编译**

```bash
go build ./search/...
```

---

## Task 3: Service + gRPC Handler（search-svc 业务与接口层）

**Files:**
- Create: `search/service/search.go`
- Create: `search/grpc/search.go`

**Step 1: 创建 Service**

Create `search/service/search.go`:

```go
package service

import (
	"context"

	"github.com/olivere/elastic/v7"
	"github.com/rermrf/emo/logger"
	"github.com/rermrf/mall/search/domain"
	"github.com/rermrf/mall/search/repository"
)

type SearchService interface {
	SearchProducts(ctx context.Context, keyword string, categoryId, brandId, priceMin, priceMax, tenantId int64, sortBy string, page, pageSize int32) ([]domain.ProductDocument, int64, error)
	GetSuggestions(ctx context.Context, prefix string, limit int) ([]string, error)
	GetHotWords(ctx context.Context, limit int) ([]domain.HotWord, error)
	GetSearchHistory(ctx context.Context, userId int64, limit int) ([]domain.SearchHistory, error)
	ClearSearchHistory(ctx context.Context, userId int64) error
	SyncProduct(ctx context.Context, doc domain.ProductDocument) error
	DeleteProduct(ctx context.Context, id int64) error
	RecordSearch(ctx context.Context, userId int64, keyword string)
}

type searchService struct {
	repo repository.SearchRepository
	l    logger.Logger
}

func NewSearchService(repo repository.SearchRepository, l logger.Logger) SearchService {
	return &searchService{repo: repo, l: l}
}

func (s *searchService) SearchProducts(ctx context.Context, keyword string, categoryId, brandId, priceMin, priceMax, tenantId int64, sortBy string, page, pageSize int32) ([]domain.ProductDocument, int64, error) {
	boolQuery := elastic.NewBoolQuery()
	// 仅搜索上架商品
	boolQuery.Filter(elastic.NewTermQuery("status", 2))
	if keyword != "" {
		boolQuery.Must(elastic.NewMultiMatchQuery(keyword, "name", "subtitle").Type("best_fields"))
	}
	if tenantId > 0 {
		boolQuery.Filter(elastic.NewTermQuery("tenant_id", tenantId))
	}
	if categoryId > 0 {
		boolQuery.Filter(elastic.NewTermQuery("category_id", categoryId))
	}
	if brandId > 0 {
		boolQuery.Filter(elastic.NewTermQuery("brand_id", brandId))
	}
	if priceMin > 0 || priceMax > 0 {
		rangeQ := elastic.NewRangeQuery("price")
		if priceMin > 0 {
			rangeQ.Gte(priceMin)
		}
		if priceMax > 0 {
			rangeQ.Lte(priceMax)
		}
		boolQuery.Filter(rangeQ)
	}

	var sortFields []elastic.Sorter
	switch sortBy {
	case "price_asc":
		sortFields = append(sortFields, elastic.NewFieldSort("price").Asc())
	case "price_desc":
		sortFields = append(sortFields, elastic.NewFieldSort("price").Desc())
	case "sales_desc":
		sortFields = append(sortFields, elastic.NewFieldSort("sales").Desc())
	default:
		if keyword != "" {
			sortFields = append(sortFields, elastic.NewScoreSort().Desc())
		} else {
			sortFields = append(sortFields, elastic.NewFieldSort("sales").Desc())
		}
	}

	from := int((page - 1) * pageSize)
	return s.repo.SearchProducts(ctx, boolQuery, sortFields, from, int(pageSize))
}

func (s *searchService) GetSuggestions(ctx context.Context, prefix string, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 10
	}
	return s.repo.Suggest(ctx, prefix, limit)
}

func (s *searchService) GetHotWords(ctx context.Context, limit int) ([]domain.HotWord, error) {
	if limit <= 0 {
		limit = 10
	}
	return s.repo.GetHotWords(ctx, limit)
}

func (s *searchService) GetSearchHistory(ctx context.Context, userId int64, limit int) ([]domain.SearchHistory, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.repo.GetSearchHistory(ctx, userId, limit)
}

func (s *searchService) ClearSearchHistory(ctx context.Context, userId int64) error {
	return s.repo.ClearSearchHistory(ctx, userId)
}

func (s *searchService) SyncProduct(ctx context.Context, doc domain.ProductDocument) error {
	return s.repo.SyncProduct(ctx, doc)
}

func (s *searchService) DeleteProduct(ctx context.Context, id int64) error {
	return s.repo.DeleteProduct(ctx, id)
}

func (s *searchService) RecordSearch(ctx context.Context, userId int64, keyword string) {
	if keyword == "" {
		return
	}
	// 异步记录热搜词
	if err := s.repo.IncrHotWord(ctx, keyword); err != nil {
		s.l.Error("记录热搜词失败", logger.Error(err))
	}
	// 异步记录搜索历史（需要登录）
	if userId > 0 {
		if err := s.repo.AddSearchHistory(ctx, userId, keyword); err != nil {
			s.l.Error("记录搜索历史失败", logger.Error(err))
		}
	}
}
```

**Step 2: 创建 gRPC Handler**

Create `search/grpc/search.go`:

```go
package grpc

import (
	"context"

	"google.golang.org/grpc"

	searchv1 "github.com/rermrf/mall/api/proto/gen/search/v1"
	"github.com/rermrf/mall/search/domain"
	"github.com/rermrf/mall/search/service"
)

type SearchGRPCServer struct {
	searchv1.UnimplementedSearchServiceServer
	svc service.SearchService
}

func NewSearchGRPCServer(svc service.SearchService) *SearchGRPCServer {
	return &SearchGRPCServer{svc: svc}
}

func (s *SearchGRPCServer) Register(server *grpc.Server) {
	searchv1.RegisterSearchServiceServer(server, s)
}

func (s *SearchGRPCServer) SearchProducts(ctx context.Context, req *searchv1.SearchProductsRequest) (*searchv1.SearchProductsResponse, error) {
	products, total, err := s.svc.SearchProducts(ctx,
		req.GetKeyword(),
		req.GetCategoryId(),
		req.GetBrandId(),
		req.GetPriceMin(),
		req.GetPriceMax(),
		req.GetTenantId(),
		req.GetSortBy(),
		req.GetPage(),
		req.GetPageSize(),
	)
	if err != nil {
		return nil, err
	}
	pbProducts := make([]*searchv1.SearchProduct, 0, len(products))
	for _, p := range products {
		pbProducts = append(pbProducts, s.toSearchProductDTO(p))
	}
	return &searchv1.SearchProductsResponse{Products: pbProducts, Total: total}, nil
}

func (s *SearchGRPCServer) GetSuggestions(ctx context.Context, req *searchv1.GetSuggestionsRequest) (*searchv1.GetSuggestionsResponse, error) {
	suggestions, err := s.svc.GetSuggestions(ctx, req.GetPrefix(), int(req.GetLimit()))
	if err != nil {
		return nil, err
	}
	return &searchv1.GetSuggestionsResponse{Suggestions: suggestions}, nil
}

func (s *SearchGRPCServer) GetHotWords(ctx context.Context, req *searchv1.GetHotWordsRequest) (*searchv1.GetHotWordsResponse, error) {
	words, err := s.svc.GetHotWords(ctx, int(req.GetLimit()))
	if err != nil {
		return nil, err
	}
	pbWords := make([]*searchv1.HotWord, 0, len(words))
	for _, w := range words {
		pbWords = append(pbWords, &searchv1.HotWord{Word: w.Word, Count: w.Count})
	}
	return &searchv1.GetHotWordsResponse{Words: pbWords}, nil
}

func (s *SearchGRPCServer) GetSearchHistory(ctx context.Context, req *searchv1.GetSearchHistoryRequest) (*searchv1.GetSearchHistoryResponse, error) {
	histories, err := s.svc.GetSearchHistory(ctx, req.GetUserId(), int(req.GetLimit()))
	if err != nil {
		return nil, err
	}
	pbHistories := make([]*searchv1.SearchHistory, 0, len(histories))
	for _, h := range histories {
		pbHistories = append(pbHistories, &searchv1.SearchHistory{Keyword: h.Keyword, Ctime: h.Ctime})
	}
	return &searchv1.GetSearchHistoryResponse{Histories: pbHistories}, nil
}

func (s *SearchGRPCServer) ClearSearchHistory(ctx context.Context, req *searchv1.ClearSearchHistoryRequest) (*searchv1.ClearSearchHistoryResponse, error) {
	err := s.svc.ClearSearchHistory(ctx, req.GetUserId())
	if err != nil {
		return nil, err
	}
	return &searchv1.ClearSearchHistoryResponse{}, nil
}

func (s *SearchGRPCServer) SyncProduct(ctx context.Context, req *searchv1.SyncProductRequest) (*searchv1.SyncProductResponse, error) {
	p := req.GetProduct()
	err := s.svc.SyncProduct(ctx, domain.ProductDocument{
		ID:           p.GetId(),
		TenantID:     p.GetTenantId(),
		Name:         p.GetName(),
		Subtitle:     p.GetSubtitle(),
		CategoryID:   p.GetCategoryId(),
		CategoryName: p.GetCategoryName(),
		BrandID:      p.GetBrandId(),
		BrandName:    p.GetBrandName(),
		Price:        p.GetPrice(),
		Sales:        p.GetSales(),
		MainImage:    p.GetMainImage(),
		Status:       p.GetStatus(),
		ShopID:       p.GetShopId(),
		ShopName:     p.GetShopName(),
	})
	if err != nil {
		return nil, err
	}
	return &searchv1.SyncProductResponse{}, nil
}

func (s *SearchGRPCServer) DeleteProduct(ctx context.Context, req *searchv1.DeleteProductRequest) (*searchv1.DeleteProductResponse, error) {
	err := s.svc.DeleteProduct(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return &searchv1.DeleteProductResponse{}, nil
}

func (s *SearchGRPCServer) toSearchProductDTO(doc domain.ProductDocument) *searchv1.SearchProduct {
	return &searchv1.SearchProduct{
		Id:           doc.ID,
		TenantId:     doc.TenantID,
		Name:         doc.Name,
		Subtitle:     doc.Subtitle,
		CategoryId:   doc.CategoryID,
		CategoryName: doc.CategoryName,
		BrandId:      doc.BrandID,
		BrandName:    doc.BrandName,
		Price:        doc.Price,
		Sales:        doc.Sales,
		MainImage:    doc.MainImage,
		ShopId:       doc.ShopID,
		ShopName:     doc.ShopName,
	}
}
```

**Step 3: 验证编译**

```bash
go build ./search/...
```

---

## Task 4: Events（Kafka Consumer 商品同步）

**Files:**
- Create: `search/events/types.go`
- Create: `search/events/consumer.go`

**Step 1: 创建事件类型**

Create `search/events/types.go`:

```go
package events

const (
	TopicProductUpdated       = "product_updated"
	TopicProductStatusChanged = "product_status_changed"
)

type ProductUpdatedEvent struct {
	ProductId int64 `json:"product_id"`
	TenantId  int64 `json:"tenant_id"`
}

type ProductStatusChangedEvent struct {
	ProductId int64 `json:"product_id"`
	TenantId  int64 `json:"tenant_id"`
	OldStatus int32 `json:"old_status"`
	NewStatus int32 `json:"new_status"`
}
```

**Step 2: 创建 Kafka Consumer**

Create `search/events/consumer.go`:

```go
package events

import (
	"context"
	"time"

	"github.com/IBM/sarama"
	"github.com/rermrf/emo/logger"
	productv1 "github.com/rermrf/mall/api/proto/gen/product/v1"
	"github.com/rermrf/mall/pkg/saramax"
	"github.com/rermrf/mall/search/domain"
	"github.com/rermrf/mall/search/service"
)

// ProductUpdatedConsumer 消费 product_updated 事件
type ProductUpdatedConsumer struct {
	client        sarama.ConsumerGroup
	l             logger.Logger
	productClient productv1.ProductServiceClient
	svc           service.SearchService
}

func NewProductUpdatedConsumer(
	client sarama.ConsumerGroup,
	l logger.Logger,
	productClient productv1.ProductServiceClient,
	svc service.SearchService,
) *ProductUpdatedConsumer {
	return &ProductUpdatedConsumer{
		client:        client,
		l:             l,
		productClient: productClient,
		svc:           svc,
	}
}

func (c *ProductUpdatedConsumer) Start() error {
	h := saramax.NewHandler[ProductUpdatedEvent](c.l, c.Consume)
	go func() {
		for {
			err := c.client.Consume(context.Background(), []string{TopicProductUpdated}, h)
			if err != nil {
				c.l.Error("消费 product_updated 事件出错", logger.Error(err))
				time.Sleep(time.Second)
			}
		}
	}()
	return nil
}

func (c *ProductUpdatedConsumer) Consume(msg *sarama.ConsumerMessage, evt ProductUpdatedEvent) error {
	ctx := context.Background()
	resp, err := c.productClient.GetProduct(ctx, &productv1.GetProductRequest{Id: evt.ProductId})
	if err != nil {
		return err
	}
	p := resp.GetProduct()
	if p.GetStatus() != 2 {
		return c.svc.DeleteProduct(ctx, evt.ProductId)
	}
	return c.svc.SyncProduct(ctx, productToDocument(p))
}

// ProductStatusChangedConsumer 消费 product_status_changed 事件
type ProductStatusChangedConsumer struct {
	client        sarama.ConsumerGroup
	l             logger.Logger
	productClient productv1.ProductServiceClient
	svc           service.SearchService
}

func NewProductStatusChangedConsumer(
	client sarama.ConsumerGroup,
	l logger.Logger,
	productClient productv1.ProductServiceClient,
	svc service.SearchService,
) *ProductStatusChangedConsumer {
	return &ProductStatusChangedConsumer{
		client:        client,
		l:             l,
		productClient: productClient,
		svc:           svc,
	}
}

func (c *ProductStatusChangedConsumer) Start() error {
	h := saramax.NewHandler[ProductStatusChangedEvent](c.l, c.Consume)
	go func() {
		for {
			err := c.client.Consume(context.Background(), []string{TopicProductStatusChanged}, h)
			if err != nil {
				c.l.Error("消费 product_status_changed 事件出错", logger.Error(err))
				time.Sleep(time.Second)
			}
		}
	}()
	return nil
}

func (c *ProductStatusChangedConsumer) Consume(msg *sarama.ConsumerMessage, evt ProductStatusChangedEvent) error {
	ctx := context.Background()
	if evt.NewStatus != 2 {
		return c.svc.DeleteProduct(ctx, evt.ProductId)
	}
	resp, err := c.productClient.GetProduct(ctx, &productv1.GetProductRequest{Id: evt.ProductId})
	if err != nil {
		return err
	}
	return c.svc.SyncProduct(ctx, productToDocument(resp.GetProduct()))
}

func productToDocument(p *productv1.Product) domain.ProductDocument {
	var minPrice int64
	if len(p.GetSkus()) > 0 {
		minPrice = p.GetSkus()[0].GetPrice()
		for _, sku := range p.GetSkus()[1:] {
			if sku.GetPrice() < minPrice {
				minPrice = sku.GetPrice()
			}
		}
	}
	return domain.ProductDocument{
		ID:           p.GetId(),
		TenantID:     p.GetTenantId(),
		Name:         p.GetName(),
		Subtitle:     p.GetSubtitle(),
		CategoryID:   p.GetCategoryId(),
		Price:        minPrice,
		Sales:        p.GetSales(),
		MainImage:    p.GetMainImage(),
		Status:       p.GetStatus(),
	}
}
```

**Step 3: 验证编译**

```bash
go build ./search/...
```

---

## Task 5: IoC + Wire + Config + Main（search-svc 基础设施）

**Files:**
- Create: `search/ioc/es.go`
- Create: `search/ioc/redis.go`
- Create: `search/ioc/logger.go`
- Create: `search/ioc/grpc.go`
- Create: `search/ioc/kafka.go`
- Create: `search/ioc/product_client.go`
- Create: `search/wire.go`
- Create: `search/app.go`
- Create: `search/main.go`
- Create: `search/config/dev.yaml`
- Generate: `search/wire_gen.go`

**Step 1: 创建 IoC — ES**

Create `search/ioc/es.go`:

```go
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
```

**Step 2: 创建 IoC — Redis**

Create `search/ioc/redis.go`:

```go
package ioc

import (
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

func InitRedis() redis.Cmdable {
	type Config struct {
		Addr     string `yaml:"addr"`
		Password string `yaml:"password"`
		DB       int    `yaml:"db"`
	}
	var cfg Config
	err := viper.UnmarshalKey("redis", &cfg)
	if err != nil {
		panic(fmt.Errorf("读取 Redis 配置失败: %w", err))
	}
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	return client
}
```

**Step 3: 创建 IoC — Logger**

Create `search/ioc/logger.go`:

```go
package ioc

import (
	"github.com/rermrf/emo/logger"
	"go.uber.org/zap"
)

func InitLogger() logger.Logger {
	l, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	return logger.NewZapLogger(l)
}
```

**Step 4: 创建 IoC — gRPC + etcd**

Create `search/ioc/grpc.go`:

```go
package ioc

import (
	"fmt"

	"github.com/rermrf/emo/logger"
	sgrpc "github.com/rermrf/mall/search/grpc"
	"github.com/rermrf/mall/pkg/grpcx"
	"github.com/rermrf/mall/pkg/tenantx"
	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
)

func InitEtcdClient() *clientv3.Client {
	var cfg struct {
		Addrs []string `yaml:"addrs"`
	}
	err := viper.UnmarshalKey("etcd", &cfg)
	if err != nil {
		panic(fmt.Errorf("读取 etcd 配置失败: %w", err))
	}
	client, err := clientv3.New(clientv3.Config{Endpoints: cfg.Addrs})
	if err != nil {
		panic(fmt.Errorf("连接 etcd 失败: %w", err))
	}
	return client
}

func InitGRPCServer(searchServer *sgrpc.SearchGRPCServer, l logger.Logger) *grpcx.Server {
	type Config struct {
		Port      int      `yaml:"port"`
		EtcdAddrs []string `yaml:"etcdAddrs"`
	}
	var cfg Config
	err := viper.UnmarshalKey("grpc", &cfg)
	if err != nil {
		panic(fmt.Errorf("读取 gRPC 配置失败: %w", err))
	}
	server := grpc.NewServer(grpc.UnaryInterceptor(tenantx.GRPCUnaryServerInterceptor()))
	searchServer.Register(server)
	return &grpcx.Server{
		Server:    server,
		Port:      cfg.Port,
		EtcdAddrs: cfg.EtcdAddrs,
		Name:      "search",
		L:         l,
	}
}
```

**Step 5: 创建 IoC — Kafka**

Create `search/ioc/kafka.go`:

```go
package ioc

import (
	"fmt"

	"github.com/IBM/sarama"
	"github.com/rermrf/emo/logger"
	productv1 "github.com/rermrf/mall/api/proto/gen/product/v1"
	"github.com/rermrf/mall/pkg/saramax"
	"github.com/rermrf/mall/search/events"
	"github.com/rermrf/mall/search/service"
	"github.com/spf13/viper"
)

func InitKafka() sarama.Client {
	type Config struct {
		Addrs []string `yaml:"addrs"`
	}
	var cfg Config
	err := viper.UnmarshalKey("kafka", &cfg)
	if err != nil {
		panic(fmt.Errorf("读取 Kafka 配置失败: %w", err))
	}
	saramaCfg := sarama.NewConfig()
	saramaCfg.Producer.Return.Successes = true
	client, err := sarama.NewClient(cfg.Addrs, saramaCfg)
	if err != nil {
		panic(fmt.Errorf("连接 Kafka 失败: %w", err))
	}
	return client
}

func InitConsumerGroup(client sarama.Client) sarama.ConsumerGroup {
	cg, err := sarama.NewConsumerGroupFromClient("search-svc", client)
	if err != nil {
		panic(fmt.Errorf("创建 Kafka ConsumerGroup 失败: %w", err))
	}
	return cg
}

func NewProductUpdatedConsumer(
	cg sarama.ConsumerGroup,
	l logger.Logger,
	productClient productv1.ProductServiceClient,
	svc service.SearchService,
) *events.ProductUpdatedConsumer {
	return events.NewProductUpdatedConsumer(cg, l, productClient, svc)
}

func NewProductStatusChangedConsumer(
	cg sarama.ConsumerGroup,
	l logger.Logger,
	productClient productv1.ProductServiceClient,
	svc service.SearchService,
) *events.ProductStatusChangedConsumer {
	return events.NewProductStatusChangedConsumer(cg, l, productClient, svc)
}

func InitConsumers(
	updatedConsumer *events.ProductUpdatedConsumer,
	statusChangedConsumer *events.ProductStatusChangedConsumer,
) []saramax.Consumer {
	return []saramax.Consumer{updatedConsumer, statusChangedConsumer}
}
```

**Step 6: 创建 IoC — Product Client**

Create `search/ioc/product_client.go`:

```go
package ioc

import (
	"fmt"

	productv1 "github.com/rermrf/mall/api/proto/gen/product/v1"
	"github.com/rermrf/mall/pkg/tenantx"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/naming/resolver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func InitProductClient(etcdClient *clientv3.Client) productv1.ProductServiceClient {
	etcdResolver, err := resolver.NewBuilder(etcdClient)
	if err != nil {
		panic(fmt.Errorf("创建 etcd resolver 失败: %w", err))
	}
	conn, err := grpc.NewClient(
		"etcd:///service/product",
		grpc.WithResolvers(etcdResolver),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(tenantx.GRPCUnaryClientInterceptor()),
	)
	if err != nil {
		panic(fmt.Errorf("连接 product gRPC 服务失败: %w", err))
	}
	return productv1.NewProductServiceClient(conn)
}
```

**Step 7: 创建 App**

Create `search/app.go`:

```go
package main

import (
	"github.com/rermrf/mall/pkg/grpcx"
	"github.com/rermrf/mall/pkg/saramax"
)

type App struct {
	Server    *grpcx.Server
	Consumers []saramax.Consumer
}
```

**Step 8: 创建 Wire DI**

Create `search/wire.go`:

```go
//go:build wireinject

package main

import (
	"github.com/google/wire"
	sgrpc "github.com/rermrf/mall/search/grpc"
	"github.com/rermrf/mall/search/ioc"
	"github.com/rermrf/mall/search/repository"
	"github.com/rermrf/mall/search/repository/cache"
	"github.com/rermrf/mall/search/repository/dao"
	"github.com/rermrf/mall/search/service"
)

var thirdPartySet = wire.NewSet(
	ioc.InitES,
	ioc.InitRedis,
	ioc.InitKafka,
	ioc.InitLogger,
	ioc.InitEtcdClient,
	ioc.InitProductClient,
)

var searchSet = wire.NewSet(
	dao.NewSearchDAO,
	cache.NewSearchCache,
	repository.NewSearchRepository,
	service.NewSearchService,
	sgrpc.NewSearchGRPCServer,
	ioc.InitGRPCServer,
	ioc.InitConsumerGroup,
	ioc.NewProductUpdatedConsumer,
	ioc.NewProductStatusChangedConsumer,
	ioc.InitConsumers,
)

func InitApp() *App {
	wire.Build(thirdPartySet, searchSet, wire.Struct(new(App), "*"))
	return new(App)
}
```

**Step 9: 创建 main.go**

Create `search/main.go`:

```go
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func main() {
	initViper()
	app := InitApp()

	for _, c := range app.Consumers {
		if err := c.Start(); err != nil {
			panic(fmt.Errorf("启动消费者失败: %w", err))
		}
	}

	go func() {
		if err := app.Server.Serve(); err != nil {
			fmt.Println("gRPC 服务启动失败:", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	fmt.Println("正在关闭服务...")
	app.Server.Close()
}

func initViper() {
	configFile := pflag.String("config", "config/dev.yaml", "配置文件路径")
	pflag.Parse()
	viper.SetConfigFile(*configFile)
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("读取配置文件失败: %w", err))
	}
}
```

**Step 10: 创建配置文件**

Create `search/config/dev.yaml`:

```yaml
es:
  url: "http://rermrf.icu:9200"

redis:
  addr: "rermrf.icu:6379"
  password: ""
  db: 7

kafka:
  addrs:
    - "rermrf.icu:9094"

grpc:
  port: 8088
  etcdAddrs:
    - "rermrf.icu:2379"

etcd:
  addrs:
    - "rermrf.icu:2379"
```

**Step 11: 生成 Wire 代码并验证**

```bash
cd search && wire && cd ..
go build ./search/...
go vet ./search/...
```

---

## Task 6: Consumer BFF Search 接口（5 个端点）

**Files:**
- Create: `consumer-bff/handler/search.go`
- Modify: `consumer-bff/ioc/grpc.go` — +InitSearchClient
- Modify: `consumer-bff/ioc/gin.go` — +searchHandler 参数 + 5 路由（3 pub + 2 auth）
- Modify: `consumer-bff/wire.go` — +InitSearchClient + NewSearchHandler

**Step 1: 创建 SearchHandler**

Create `consumer-bff/handler/search.go`:

```go
package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/rermrf/emo/logger"
	searchv1 "github.com/rermrf/mall/api/proto/gen/search/v1"
	"github.com/rermrf/mall/pkg/ginx"
)

type SearchHandler struct {
	searchClient searchv1.SearchServiceClient
	l            logger.Logger
}

func NewSearchHandler(searchClient searchv1.SearchServiceClient, l logger.Logger) *SearchHandler {
	return &SearchHandler{
		searchClient: searchClient,
		l:            l,
	}
}

type SearchReq struct {
	Keyword    string `form:"keyword"`
	CategoryID int64  `form:"category_id"`
	BrandID    int64  `form:"brand_id"`
	PriceMin   int64  `form:"price_min"`
	PriceMax   int64  `form:"price_max"`
	SortBy     string `form:"sort_by"`
	Page       int32  `form:"page"`
	PageSize   int32  `form:"page_size"`
}

func (h *SearchHandler) Search(ctx *gin.Context, req SearchReq) (ginx.Result, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	tenantId, _ := ctx.Get("tenant_id")
	resp, err := h.searchClient.SearchProducts(ctx.Request.Context(), &searchv1.SearchProductsRequest{
		Keyword:    req.Keyword,
		CategoryId: req.CategoryID,
		BrandId:    req.BrandID,
		PriceMin:   req.PriceMin,
		PriceMax:   req.PriceMax,
		TenantId:   tenantId.(int64),
		SortBy:     req.SortBy,
		Page:       req.Page,
		PageSize:   req.PageSize,
	})
	if err != nil {
		return ginx.Result{}, err
	}
	// 异步记录搜索（通过 gRPC 调 search-svc 的 RecordSearch 不暴露 RPC，
	// 所以在 BFF 层不记录，由 search-svc 在 SearchProducts 内部处理也可以；
	// 但按设计搜索记录在 service 层处理更简单）
	return ginx.Result{Code: 0, Msg: "success", Data: map[string]any{
		"products": resp.GetProducts(),
		"total":    resp.GetTotal(),
	}}, nil
}

func (h *SearchHandler) GetSuggestions(ctx *gin.Context) {
	prefix := ctx.Query("prefix")
	limitStr := ctx.DefaultQuery("limit", "10")
	limit, _ := strconv.ParseInt(limitStr, 10, 32)
	if limit <= 0 {
		limit = 10
	}
	resp, err := h.searchClient.GetSuggestions(ctx.Request.Context(), &searchv1.GetSuggestionsRequest{
		Prefix: prefix,
		Limit:  int32(limit),
	})
	if err != nil {
		h.l.Error("获取搜索建议失败", logger.Error(err))
		ctx.JSON(http.StatusOK, ginx.Result{Code: 5, Msg: "系统错误"})
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetSuggestions()})
}

func (h *SearchHandler) GetHotWords(ctx *gin.Context) {
	limitStr := ctx.DefaultQuery("limit", "10")
	limit, _ := strconv.ParseInt(limitStr, 10, 32)
	if limit <= 0 {
		limit = 10
	}
	resp, err := h.searchClient.GetHotWords(ctx.Request.Context(), &searchv1.GetHotWordsRequest{
		Limit: int32(limit),
	})
	if err != nil {
		h.l.Error("获取热搜词失败", logger.Error(err))
		ctx.JSON(http.StatusOK, ginx.Result{Code: 5, Msg: "系统错误"})
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetWords()})
}

func (h *SearchHandler) GetSearchHistory(ctx *gin.Context) {
	uid, _ := ctx.Get("user_id")
	limitStr := ctx.DefaultQuery("limit", "20")
	limit, _ := strconv.ParseInt(limitStr, 10, 32)
	if limit <= 0 {
		limit = 20
	}
	resp, err := h.searchClient.GetSearchHistory(ctx.Request.Context(), &searchv1.GetSearchHistoryRequest{
		UserId: uid.(int64),
		Limit:  int32(limit),
	})
	if err != nil {
		h.l.Error("获取搜索历史失败", logger.Error(err))
		ctx.JSON(http.StatusOK, ginx.Result{Code: 5, Msg: "系统错误"})
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetHistories()})
}

func (h *SearchHandler) ClearSearchHistory(ctx *gin.Context) {
	uid, _ := ctx.Get("user_id")
	_, err := h.searchClient.ClearSearchHistory(ctx.Request.Context(), &searchv1.ClearSearchHistoryRequest{
		UserId: uid.(int64),
	})
	if err != nil {
		h.l.Error("清空搜索历史失败", logger.Error(err))
		ctx.JSON(http.StatusOK, ginx.Result{Code: 5, Msg: "系统错误"})
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success"})
}
```

**Step 2: 修改 consumer-bff/ioc/grpc.go — 添加 InitSearchClient**

在 import 块中添加：
```go
searchv1 "github.com/rermrf/mall/api/proto/gen/search/v1"
```

在文件末尾添加：
```go
func InitSearchClient(etcdClient *clientv3.Client) searchv1.SearchServiceClient {
	conn := initServiceConn(etcdClient, "search")
	return searchv1.NewSearchServiceClient(conn)
}
```

**Step 3: 修改 consumer-bff/ioc/gin.go**

`InitGinServer` 函数签名添加 `searchHandler *handler.SearchHandler` 参数。

在 `pub` 路由组中添加搜索公开路由（搜索、建议、热搜不需要登录）：
```go
		// 搜索（公开）
		pub.GET("/search", ginx.WrapQuery[handler.SearchReq](l, searchHandler.Search))
		pub.GET("/search/suggestions", searchHandler.GetSuggestions)
		pub.GET("/search/hot", searchHandler.GetHotWords)
```

在 `auth` 路由组中添加搜索历史路由（需要登录）：
```go
		// 搜索历史
		auth.GET("/search/history", searchHandler.GetSearchHistory)
		auth.DELETE("/search/history", searchHandler.ClearSearchHistory)
```

**Step 4: 修改 consumer-bff/wire.go**

`thirdPartySet` 添加 `ioc.InitSearchClient`。
`handlerSet` 添加 `handler.NewSearchHandler`。

**Step 5: 重新生成 Wire 代码并验证**

```bash
cd consumer-bff && wire && cd ..
go build ./consumer-bff/...
go vet ./consumer-bff/...
```

---

## 文件清单总览

| # | 文件路径 | 操作 | Task |
|---|---------|------|------|
| 1 | `search/domain/search.go` | 新建 | 1 |
| 2 | `search/repository/dao/search.go` | 新建 | 1 |
| 3 | `search/repository/dao/init.go` | 新建 | 1 |
| 4 | `search/repository/cache/search.go` | 新建 | 2 |
| 5 | `search/repository/search.go` | 新建 | 2 |
| 6 | `search/service/search.go` | 新建 | 3 |
| 7 | `search/grpc/search.go` | 新建 | 3 |
| 8 | `search/events/types.go` | 新建 | 4 |
| 9 | `search/events/consumer.go` | 新建 | 4 |
| 10 | `search/ioc/es.go` | 新建 | 5 |
| 11 | `search/ioc/redis.go` | 新建 | 5 |
| 12 | `search/ioc/logger.go` | 新建 | 5 |
| 13 | `search/ioc/grpc.go` | 新建 | 5 |
| 14 | `search/ioc/kafka.go` | 新建 | 5 |
| 15 | `search/ioc/product_client.go` | 新建 | 5 |
| 16 | `search/wire.go` | 新建 | 5 |
| 17 | `search/app.go` | 新建 | 5 |
| 18 | `search/main.go` | 新建 | 5 |
| 19 | `search/config/dev.yaml` | 新建 | 5 |
| 20 | `search/wire_gen.go` | 生成 | 5 |
| 21 | `consumer-bff/handler/search.go` | 新建 | 6 |
| 22 | `consumer-bff/ioc/grpc.go` | 修改 | 6 |
| 23 | `consumer-bff/ioc/gin.go` | 修改 | 6 |
| 24 | `consumer-bff/wire.go` | 修改 | 6 |
| 25 | `consumer-bff/wire_gen.go` | 重新生成 | 6 |

共 25 个文件（19 新建 + 3 修改 + 1 生成 + 2 重新生成）

## 验证

```bash
go build ./search/...
go vet ./search/...
go build ./consumer-bff/...
go vet ./consumer-bff/...
```
