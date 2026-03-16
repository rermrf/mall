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
	CategoryID int64  `form:"categoryId"`
	BrandID    int64  `form:"brandId"`
	PriceMin   int64  `form:"priceMin"`
	PriceMax   int64  `form:"priceMax"`
	SortBy     string `form:"sortBy"`
	Page       int32  `form:"page"`
	PageSize   int32  `form:"pageSize"`
}

func (h *SearchHandler) Search(ctx *gin.Context, req SearchReq) (ginx.Result, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.PriceMin > 0 && req.PriceMax > 0 && req.PriceMin > req.PriceMax {
		return ginx.Result{Code: ginx.CodeInvalidPrice, Msg: "最低价不能大于最高价"}, nil
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
		return ginx.HandleGRPCError(err, "搜索商品失败")
	}
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
		result, _ := ginx.HandleRawError(err)
		ctx.JSON(http.StatusOK, result)
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
		result, _ := ginx.HandleRawError(err)
		ctx.JSON(http.StatusOK, result)
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetWords()})
}

func (h *SearchHandler) GetSearchHistory(ctx *gin.Context) {
	uid, _ := ctx.Get("uid")
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
		result, _ := ginx.HandleRawError(err)
		ctx.JSON(http.StatusOK, result)
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success", Data: resp.GetHistories()})
}

func (h *SearchHandler) ClearSearchHistory(ctx *gin.Context) {
	uid, _ := ctx.Get("uid")
	_, err := h.searchClient.ClearSearchHistory(ctx.Request.Context(), &searchv1.ClearSearchHistoryRequest{
		UserId: uid.(int64),
	})
	if err != nil {
		h.l.Error("清空搜索历史失败", logger.Error(err))
		result, _ := ginx.HandleRawError(err)
		ctx.JSON(http.StatusOK, result)
		return
	}
	ctx.JSON(http.StatusOK, ginx.Result{Code: 0, Msg: "success"})
}
