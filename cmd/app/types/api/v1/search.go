package v1

// SearchResult represents one search hit.
// SearchResult 表示单条搜索结果。
type SearchResult struct {
	Type      string  `json:"type"`
	ID        uint64  `json:"id"`
	Title     string  `json:"title"`
	Snippet   string  `json:"snippet"`
	RankScore float64 `json:"rank_score"`
}

// SearchResponse represents paged search response.
// SearchResponse 表示分页搜索响应。
type SearchResponse struct {
	Items    []SearchResult `json:"items"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
}
