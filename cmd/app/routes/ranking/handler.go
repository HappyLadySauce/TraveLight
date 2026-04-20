package ranking

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/HappyLadySauce/TraveLight/cmd/app/routes/auth"
	"github.com/HappyLadySauce/TraveLight/cmd/app/router"
	"github.com/HappyLadySauce/TraveLight/cmd/app/svc"
	v1 "github.com/HappyLadySauce/TraveLight/cmd/app/types/api/v1"
	"github.com/HappyLadySauce/TraveLight/cmd/app/types/common"
	"github.com/HappyLadySauce/TraveLight/pkg/craw"
)

const hotRankingCacheKey = "ranking:attractions:hot"

type Handler struct {
	svcCtx *svc.ServiceContext
}

func RegisterRoutes(svcCtx *svc.ServiceContext) {
	h := &Handler{svcCtx: svcCtx}
	protected := router.V1().Group("", auth.AuthMiddleware(svcCtx))
	protected.POST("/attractions/:id/view", h.incrView)

	public := router.V1().Group("")
	public.GET("/rankings/attractions/hot", h.listHotAttractions)
}

// incrView godoc
//
//	@Summary		景点浏览计数
//	@Description	对指定景点执行一次浏览计数加一
//	@Tags			ranking
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			id	path		int	true	"景点ID"
//	@Success		200	{object}	common.BaseResponse
//	@Failure		400	{object}	common.BaseResponse
//	@Failure		401	{object}	common.BaseResponse
//	@Failure		404	{object}	common.BaseResponse
//	@Failure		500	{object}	common.BaseResponse
//	@Router			/api/v1/attractions/{id}/view [post]
func (h *Handler) incrView(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		common.Fail(c, http.StatusBadRequest, "invalid attraction id")
		return
	}
	result := h.svcCtx.DB.Exec(
		"UPDATE attractions SET view_count = view_count + 1, updated_at = NOW() WHERE id = ?",
		id,
	)
	if result.Error != nil {
		common.Fail(c, http.StatusInternalServerError, "update view count failed")
		return
	}
	if result.RowsAffected == 0 {
		common.Fail(c, http.StatusNotFound, "attraction not found")
		return
	}
	_ = h.svcCtx.Redis.Del(context.Background(), hotRankingCacheKey).Err()
	common.Success(c, gin.H{"updated": true})
}

// listHotAttractions godoc
//
//	@Summary		热门景点排行
//	@Description	按景点点击量从高到低返回排行榜
//	@Tags			ranking
//	@Accept			json
//	@Produce		json
//	@Param			limit	query		int	false	"返回数量上限(1-100)"	default(20)
//	@Success		200		{object}	common.BaseResponse
//	@Failure		400		{object}	common.BaseResponse
//	@Failure		500		{object}	common.BaseResponse
//	@Router			/api/v1/rankings/attractions/hot [get]
func (h *Handler) listHotAttractions(c *gin.Context) {
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if err != nil || limit <= 0 || limit > 100 {
		common.Fail(c, http.StatusBadRequest, "invalid limit")
		return
	}

	ctx := context.Background()
	if cacheData, cacheErr := h.svcCtx.Redis.Get(ctx, hotRankingCacheKey).Result(); cacheErr == nil {
		var items []v1.RankingAttractionItem
		if err := json.Unmarshal([]byte(cacheData), &items); err == nil {
			common.Success(c, v1.RankingResponse{Items: trimRanking(items, limit)})
			return
		}
	}

	rows := make([]craw.Attraction, 0, limit)
	if err := h.svcCtx.DB.Order("view_count DESC").Limit(limit).Find(&rows).Error; err != nil {
		common.Fail(c, http.StatusInternalServerError, "query ranking failed")
		return
	}
	items := make([]v1.RankingAttractionItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, v1.RankingAttractionItem{
			ID:          row.ID,
			Name:        row.Name,
			ImageURL:    row.ImageURL,
			Description: row.Description,
			ViewCount:   row.ViewCount,
		})
	}
	if bytes, err := json.Marshal(items); err == nil {
		_ = h.svcCtx.Redis.Set(ctx, hotRankingCacheKey, string(bytes), 60*time.Second).Err()
	}
	common.Success(c, v1.RankingResponse{Items: items})
}

func trimRanking(items []v1.RankingAttractionItem, limit int) []v1.RankingAttractionItem {
	if len(items) <= limit {
		return items
	}
	return items[:limit]
}
