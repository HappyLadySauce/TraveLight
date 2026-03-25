package v1

type StartCrawlRequest struct{}

type StartCrawlResponse struct {
	Message string `json:"message"`
}

type StatusCrawlRequest struct{}

type StatusCrawlResponse struct {
	IsRunning bool   `json:"is_running"`
	LastRun   string `json:"last_run,omitempty"`
	LastError string `json:"last_error,omitempty"`
	LastCount int    `json:"last_count"`
}
