package v1

// RankingAttractionItem represents one attraction in ranking list.
// RankingAttractionItem 表示排行中的单个景点项。
type RankingAttractionItem struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	ImageURL    string `json:"image_url"`
	Description string `json:"description"`
	ViewCount   int64  `json:"view_count"`
}

// RankingResponse represents hot ranking response.
// RankingResponse 表示热门排行响应。
type RankingResponse struct {
	Items []RankingAttractionItem `json:"items"`
}
