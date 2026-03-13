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
