package craw

import (
	"time"

	"github.com/gin-gonic/gin"

	v1 "github.com/HappyLadySauce/TraveLight/cmd/app/types/api/v1"
	"github.com/HappyLadySauce/TraveLight/cmd/app/types/common"
)

func StatusCrawl(c *gin.Context, h *CrawlHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()

	var lastRunStr string
	if !h.lastRun.IsZero() {
		lastRunStr = h.lastRun.Format(time.RFC3339)
	}

	var lastErrorStr string
	if h.lastError != nil {
		lastErrorStr = h.lastError.Error()
	}

	common.Success(c, v1.StatusCrawlResponse{
		IsRunning: h.isRunning,
		LastRun:   lastRunStr,
		LastError: lastErrorStr,
		LastCount: h.lastCount,
	})
}
