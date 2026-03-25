package craw

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"k8s.io/klog/v2"

	v1 "github.com/HappyLadySauce/TraveLight/cmd/app/types/api/v1"
	"github.com/HappyLadySauce/TraveLight/cmd/app/types/common"
	"github.com/HappyLadySauce/TraveLight/pkg/craw"
)

func StartCrawl(c *gin.Context, h *CrawlHandler) {
	h.mu.Lock()
	if h.isRunning {
		h.mu.Unlock()
		common.Fail(c, http.StatusConflict, "爬虫正在运行中，请稍后再试")
		return
	}
	h.isRunning = true
	h.mu.Unlock()

	go func() {
		defer func() {
			h.mu.Lock()
			h.isRunning = false
			h.lastRun = time.Now()
			h.mu.Unlock()
		}()

		ctx := context.Background()
		crawler := craw.NewCrawler(h.svcCtx.DB)
		defer crawler.Close()

		klog.InfoS("开始执行爬虫任务")
		if err := crawler.Run(ctx); err != nil {
			klog.ErrorS(err, "爬虫执行失败")
			h.mu.Lock()
			h.lastError = err
			h.mu.Unlock()
			return
		}

		h.mu.Lock()
		h.lastCount = len(crawler.GetAttractions())
		h.lastError = nil
		h.mu.Unlock()

		klog.InfoS("爬虫任务完成", "count", h.lastCount)
	}()

	common.Success(c, v1.StartCrawlResponse{
		Message: "爬虫任务已启动，请通过 /api/v1/crawl/status 查看状态",
	})
}
