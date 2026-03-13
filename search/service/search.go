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
