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

func (h *CrawlHandler) handleStartCrawl(c *gin.Context) {
	StartCrawl(c, h)
}

func (h *CrawlHandler) handleStatusCrawl(c *gin.Context) {
	StatusCrawl(c, h)
}
