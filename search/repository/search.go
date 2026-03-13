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
