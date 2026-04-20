package comment

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/HappyLadySauce/TraveLight/cmd/app/routes/auth"
	"github.com/HappyLadySauce/TraveLight/cmd/app/router"
	"github.com/HappyLadySauce/TraveLight/cmd/app/svc"
	v1 "github.com/HappyLadySauce/TraveLight/cmd/app/types/api/v1"
	"github.com/HappyLadySauce/TraveLight/cmd/app/types/common"
	"github.com/HappyLadySauce/TraveLight/pkg/model"
)

type Handler struct {
	svcCtx *svc.ServiceContext
}

func RegisterRoutes(svcCtx *svc.ServiceContext) {
	h := &Handler{svcCtx: svcCtx}
	protected := router.V1().Group("", auth.AuthMiddleware(svcCtx))
	protected.POST("/contents/:type/:id/comments", h.createComment)
	protected.DELETE("/comments/:commentId", h.deleteComment)

	public := router.V1().Group("")
	public.GET("/contents/:type/:id/comments", h.listComments)
}

// createComment godoc
//
//	@Summary		发表评论
//	@Description	为文章或景点发布评论
//	@Tags			comment
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			type	path		string					true	"内容类型(article|attraction)"
//	@Param			id		path		int						true	"内容ID"
//	@Param			request	body		v1.CreateCommentRequest	true	"评论内容"
//	@Success		200		{object}	common.BaseResponse
//	@Failure		400		{object}	common.BaseResponse
//	@Failure		401		{object}	common.BaseResponse
//	@Failure		404		{object}	common.BaseResponse
//	@Failure		500		{object}	common.BaseResponse
//	@Router			/api/v1/contents/{type}/{id}/comments [post]
func (h *Handler) createComment(c *gin.Context) {
	userID, ok := auth.CurrentUserID(c)
	if !ok {
		common.Fail(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	contentType, contentID, valid := parseContentIdentity(c)
	if !valid {
		return
	}
	if !h.contentExists(contentType, contentID) {
		common.Fail(c, http.StatusNotFound, "content not found")
		return
	}
	var req v1.CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, http.StatusBadRequest, "invalid request payload")
		return
	}
	comment := model.Comment{
		ContentType: contentType,
		ContentID:   contentID,
		UserID:      userID,
		Body:        strings.TrimSpace(req.Body),
		Status:      "active",
	}
	if err := h.svcCtx.DB.Create(&comment).Error; err != nil {
		common.Fail(c, http.StatusInternalServerError, "create comment failed")
		return
	}
	common.Success(c, gin.H{"comment_id": comment.ID})
}

// listComments godoc
//
//	@Summary		查询评论列表
//	@Description	查询文章或景点下的评论列表
//	@Tags			comment
//	@Accept			json
//	@Produce		json
//	@Param			type	path		string	true	"内容类型(article|attraction)"
//	@Param			id		path		int		true	"内容ID"
//	@Success		200		{object}	common.BaseResponse
//	@Failure		400		{object}	common.BaseResponse
//	@Failure		500		{object}	common.BaseResponse
//	@Router			/api/v1/contents/{type}/{id}/comments [get]
func (h *Handler) listComments(c *gin.Context) {
	contentType, contentID, valid := parseContentIdentity(c)
	if !valid {
		return
	}
	var total int64
	query := h.svcCtx.DB.Model(&model.Comment{}).
		Where("content_type = ? AND content_id = ? AND status = ?", contentType, contentID, "active")
	if err := query.Count(&total).Error; err != nil {
		common.Fail(c, http.StatusInternalServerError, "count comments failed")
		return
	}
	comments := make([]model.Comment, 0, 20)
	if err := query.Order("created_at DESC").Limit(200).Find(&comments).Error; err != nil {
		common.Fail(c, http.StatusInternalServerError, "query comments failed")
		return
	}
	items := make([]v1.CommentItem, 0, len(comments))
	for _, item := range comments {
		items = append(items, v1.CommentItem{
			ID:          item.ID,
			ContentType: item.ContentType,
			ContentID:   item.ContentID,
			UserID:      item.UserID,
			Body:        item.Body,
			CreatedAt:   item.CreatedAt.Format(time.RFC3339),
		})
	}
	common.Success(c, v1.ListCommentResponse{
		Items: items,
		Total: total,
	})
}

// deleteComment godoc
//
//	@Summary		删除评论
//	@Description	删除当前登录用户自己的评论
//	@Tags			comment
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			commentId	path		int	true	"评论ID"
//	@Success		200			{object}	common.BaseResponse
//	@Failure		400			{object}	common.BaseResponse
//	@Failure		401			{object}	common.BaseResponse
//	@Failure		403			{object}	common.BaseResponse
//	@Failure		404			{object}	common.BaseResponse
//	@Failure		500			{object}	common.BaseResponse
//	@Router			/api/v1/comments/{commentId} [delete]
func (h *Handler) deleteComment(c *gin.Context) {
	userID, ok := auth.CurrentUserID(c)
	if !ok {
		common.Fail(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	commentID, err := strconv.ParseUint(c.Param("commentId"), 10, 64)
	if err != nil || commentID == 0 {
		common.Fail(c, http.StatusBadRequest, "invalid comment id")
		return
	}
	var comment model.Comment
	if err := h.svcCtx.DB.First(&comment, commentID).Error; err != nil {
		common.Fail(c, http.StatusNotFound, "comment not found")
		return
	}
	if comment.UserID != userID {
		common.Fail(c, http.StatusForbidden, "cannot delete others comment")
		return
	}
	if err := h.svcCtx.DB.Model(&comment).Updates(map[string]any{
		"status": "deleted",
	}).Error; err != nil {
		common.Fail(c, http.StatusInternalServerError, "delete comment failed")
		return
	}
	common.Success(c, gin.H{"deleted": true})
}

func parseContentIdentity(c *gin.Context) (string, uint64, bool) {
	contentType := strings.ToLower(strings.TrimSpace(c.Param("type")))
	if contentType != model.ContentTypeArticle && contentType != model.ContentTypeAttraction {
		common.Fail(c, http.StatusBadRequest, "invalid content type")
		return "", 0, false
	}
	contentID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || contentID == 0 {
		common.Fail(c, http.StatusBadRequest, "invalid content id")
		return "", 0, false
	}
	return contentType, contentID, true
}

func (h *Handler) contentExists(contentType string, contentID uint64) bool {
	var count int64
	switch contentType {
	case model.ContentTypeArticle:
		h.svcCtx.DB.Model(&model.Article{}).Where("id = ?", contentID).Count(&count)
	case model.ContentTypeAttraction:
		h.svcCtx.DB.Table("attractions").Where("id = ?", contentID).Count(&count)
	}
	return count > 0
}
