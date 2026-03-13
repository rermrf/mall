package dao

import (
	"context"
	"encoding/json"
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
