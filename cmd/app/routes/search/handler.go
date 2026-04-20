package search

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/HappyLadySauce/TraveLight/cmd/app/router"
	"github.com/HappyLadySauce/TraveLight/cmd/app/svc"
	v1 "github.com/HappyLadySauce/TraveLight/cmd/app/types/api/v1"
	"github.com/HappyLadySauce/TraveLight/cmd/app/types/common"
)

type Handler struct {
	svcCtx *svc.ServiceContext
}

type searchRow struct {
	Type      string  `gorm:"column:type"`
	ID        uint64  `gorm:"column:id"`
	Title     string  `gorm:"column:title"`
	Snippet   string  `gorm:"column:snippet"`
	RankScore float64 `gorm:"column:rank_score"`
}

func RegisterRoutes(svcCtx *svc.ServiceContext) {
	h := &Handler{svcCtx: svcCtx}
	group := router.V1().Group("/search")
	group.GET("", h.search)
}

// search godoc
//
//	@Summary		全文搜索
//	@Description	根据关键词在文章与景点中执行 PostgreSQL 全文搜索
//	@Tags			search
//	@Accept			json
//	@Produce		json
//	@Param			q			query		string	true	"搜索关键词"
//	@Param			type		query		string	false	"搜索类型(all|article|attraction)"	default(all)
//	@Param			page		query		int		false	"页码(>=1)"	default(1)
//	@Param			pageSize	query		int		false	"每页数量(1-50)"	default(20)
//	@Success		200			{object}	common.BaseResponse
//	@Failure		400			{object}	common.BaseResponse
//	@Failure		500			{object}	common.BaseResponse
//	@Router			/api/v1/search [get]
func (h *Handler) search(c *gin.Context) {
	keyword := strings.TrimSpace(c.Query("q"))
	if keyword == "" || len(keyword) > 100 {
		common.Fail(c, http.StatusBadRequest, "invalid query keyword")
		return
	}
	searchType := strings.ToLower(strings.TrimSpace(c.DefaultQuery("type", "all")))
	if searchType != "all" && searchType != "article" && searchType != "attraction" {
		common.Fail(c, http.StatusBadRequest, "invalid search type")
		return
	}
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page <= 0 {
		common.Fail(c, http.StatusBadRequest, "invalid page")
		return
	}
	pageSize, err := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if err != nil || pageSize <= 0 || pageSize > 50 {
		common.Fail(c, http.StatusBadRequest, "invalid pageSize")
		return
	}
	offset := (page - 1) * pageSize

	rows := make([]searchRow, 0, pageSize)
	if searchType == "all" {
		if err := h.svcCtx.DB.Raw(unionSQL(), keyword, keyword, keyword, keyword, pageSize, offset).Scan(&rows).Error; err != nil {
			common.Fail(c, http.StatusInternalServerError, "search failed")
			return
		}
	} else {
		sql, args := singleSQL(searchType, keyword, pageSize, offset)
		if err := h.svcCtx.DB.Raw(sql, args...).Scan(&rows).Error; err != nil {
			common.Fail(c, http.StatusInternalServerError, "search failed")
			return
		}
	}

	items := make([]v1.SearchResult, 0, len(rows))
	for _, row := range rows {
		items = append(items, v1.SearchResult{
			Type:      row.Type,
			ID:        row.ID,
			Title:     row.Title,
			Snippet:   row.Snippet,
			RankScore: row.RankScore,
		})
	}
	common.Success(c, v1.SearchResponse{
		Items:    items,
		Page:     page,
		PageSize: pageSize,
	})
}

func unionSQL() string {
	return `
SELECT * FROM (
	SELECT
		'article' AS type,
		id,
		title,
		left(content, 200) AS snippet,
		ts_rank(search_vector, plainto_tsquery('simple', ?)) AS rank_score
	FROM articles
	WHERE search_vector @@ plainto_tsquery('simple', ?)
	UNION ALL
	SELECT
		'attraction' AS type,
		id,
		name AS title,
		left(description, 200) AS snippet,
		ts_rank(search_vector, plainto_tsquery('simple', ?)) AS rank_score
	FROM attractions
	WHERE search_vector @@ plainto_tsquery('simple', ?)
) t
ORDER BY rank_score DESC
LIMIT ? OFFSET ?;
`
}

func singleSQL(searchType string, keyword string, pageSize int, offset int) (string, []any) {
	switch searchType {
	case "article":
		return `
SELECT
	'article' AS type,
	id,
	title,
	left(content, 200) AS snippet,
	ts_rank(search_vector, plainto_tsquery('simple', ?)) AS rank_score
FROM articles
WHERE search_vector @@ plainto_tsquery('simple', ?)
ORDER BY rank_score DESC
LIMIT ? OFFSET ?;
`, []any{keyword, keyword, pageSize, offset}
	default:
		return `
SELECT
	'attraction' AS type,
	id,
	name AS title,
	left(description, 200) AS snippet,
	ts_rank(search_vector, plainto_tsquery('simple', ?)) AS rank_score
FROM attractions
WHERE search_vector @@ plainto_tsquery('simple', ?)
ORDER BY rank_score DESC
LIMIT ? OFFSET ?;
`, []any{keyword, keyword, pageSize, offset}
	}
}
