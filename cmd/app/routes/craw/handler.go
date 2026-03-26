package craw

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/HappyLadySauce/TraveLight/cmd/app/router"
	"github.com/HappyLadySauce/TraveLight/cmd/app/svc"
)

type CrawlHandler struct {
	svcCtx    *svc.ServiceContext
	mu        sync.Mutex
	isRunning bool
	lastRun   time.Time
	lastError error
	lastCount int
}

var crawlHandler *CrawlHandler
var crawlHandlerOnce sync.Once

func GetCrawlHandler(svcCtx *svc.ServiceContext) *CrawlHandler {
	crawlHandlerOnce.Do(func() {
		crawlHandler = &CrawlHandler{
			svcCtx: svcCtx,
		}
	})
	return crawlHandler
}

func RegisterRoutes(svcCtx *svc.ServiceContext) {
	h := GetCrawlHandler(svcCtx)

	group := router.V1().Group("/crawl")
	group.POST("/start", h.handleStartCrawl)
	group.GET("/status", h.handleStatusCrawl)
}

// handleStartCrawl godoc
//
//	@Summary		启动爬虫任务
//	@Description	启动景点数据爬虫任务，异步执行
//	@Tags			crawl
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	v1.StartCrawlResponse	"爬虫任务已启动"
//	@Failure		409	{object}	map[string]string		"爬虫正在运行中"
//	@Router			/api/v1/crawl/start [post]
func (h *CrawlHandler) handleStartCrawl(c *gin.Context) {
	StartCrawl(c, h)
}

// handleStatusCrawl godoc
//
//	@Summary		获取爬虫状态
//	@Description	获取当前爬虫任务的运行状态和上次执行结果
//	@Tags			crawl
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	v1.StatusCrawlResponse	"获取状态成功"
//	@Router			/api/v1/crawl/status [get]
func (h *CrawlHandler) handleStatusCrawl(c *gin.Context) {
	StatusCrawl(c, h)
}
