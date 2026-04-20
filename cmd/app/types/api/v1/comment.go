package v1

// CreateCommentRequest represents create comment payload.
// CreateCommentRequest 表示发表评论请求体。
type CreateCommentRequest struct {
	Body string `json:"body" binding:"required,min=1,max=1000"`
}

// CommentItem represents one comment item.
// CommentItem 表示单条评论信息。
type CommentItem struct {
	ID          uint64 `json:"id"`
	ContentType string `json:"content_type"`
	ContentID   uint64 `json:"content_id"`
	UserID      uint64 `json:"user_id"`
	Body        string `json:"body"`
	CreatedAt   string `json:"created_at"`
}

// ListCommentResponse represents comment list response.
// ListCommentResponse 表示评论列表响应。
type ListCommentResponse struct {
	Items []CommentItem `json:"items"`
	Total int64         `json:"total"`
}
