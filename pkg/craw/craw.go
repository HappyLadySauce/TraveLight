package craw

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
	"k8s.io/klog/v2"

	"gorm.io/gorm"
)

const (
	baseURL      = "https://chongqing.cncn.com/jingdian/"
	detailURLFmt = "https://chongqing.cncn.com/jingdian/%s/"
	maxPages     = 22
	workerLimit  = 5
	batchSize    = 20
)

// Crawler 异步爬虫结构体
type Crawler struct {
	db              *gorm.DB
	collector       *colly.Collector
	detailCollector *colly.Collector
	attractions     map[string]*Attraction
	mu              sync.RWMutex
	errors          []error
	errorMu         sync.Mutex
	dbChan          chan *Attraction
	wg              sync.WaitGroup
	ctx             context.Context
	cancel          context.CancelFunc
}

// CrawlerOption 爬虫配置选项
type CrawlerOption func(*Crawler)

// NewCrawler 创建新的爬虫实例
func NewCrawler(db *gorm.DB, opts ...CrawlerOption) *Crawler {
	ctx, cancel := context.WithCancel(context.Background())
	c := &Crawler{
		db:          db,
		attractions: make(map[string]*Attraction),
		dbChan:      make(chan *Attraction, 100),
		ctx:         ctx,
		cancel:      cancel,
	}

	for _, opt := range opts {
		opt(c)
	}

	c.collector = c.createListCollector()
	c.detailCollector = c.createDetailCollector()

	return c
}

// createListCollector 创建列表页收集器
func (c *Crawler) createListCollector() *colly.Collector {
	col := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
		colly.AllowURLRevisit(),
	)

	col.SetRequestTimeout(30 * time.Second)

	col.Limit(&colly.LimitRule{
		DomainGlob:  "chongqing.cncn.com",
		Delay:       500 * time.Millisecond,
		RandomDelay: 300 * time.Millisecond,
		Parallelism: workerLimit,
	})

	col.OnRequest(func(r *colly.Request) {
		klog.InfoS("[列表页] 开始请求", "url", r.URL.String())
	})

	col.OnError(func(r *colly.Response, err error) {
		klog.ErrorS(err, "[列表页] 请求失败", "url", r.Request.URL.String(), "statusCode", r.StatusCode)
		c.recordError(fmt.Errorf("列表页面请求失败 [%s]: %w", r.Request.URL.String(), err))
	})

	col.OnHTML("div.city_spots_list ul li", func(e *colly.HTMLElement) {
		attraction := c.parseListItem(e)
		if attraction != nil && attraction.Name != "" {
			c.mu.Lock()
			c.attractions[attraction.SourceURL] = attraction
			c.mu.Unlock()

			klog.InfoS("[列表页] 解析到景点", "name", attraction.Name, "sourceID", attraction.SourceURL)

			if attraction.SourceURL != "" {
				detailURL := fmt.Sprintf(detailURLFmt, attraction.SourceURL)
				c.wg.Add(1)
				go func(url, name string) {
					defer c.wg.Done()
					klog.InfoS("[详情页] 准备爬取", "name", name, "url", url)
					if err := c.detailCollector.Visit(url); err != nil {
						klog.ErrorS(err, "[详情页] 访问失败", "url", url, "name", name)
						c.recordError(fmt.Errorf("访问详情页面失败 [%s]: %w", url, err))
					}
				}(detailURL, attraction.Name)
			}
		}
	})

	return col
}

// createDetailCollector 创建详情页收集器
func (c *Crawler) createDetailCollector() *colly.Collector {
	col := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
		colly.AllowURLRevisit(),
	)

	col.SetRequestTimeout(30 * time.Second)

	col.Limit(&colly.LimitRule{
		DomainGlob:  "chongqing.cncn.com",
		Delay:       300 * time.Millisecond,
		RandomDelay: 200 * time.Millisecond,
		Parallelism: workerLimit,
	})

	col.OnRequest(func(r *colly.Request) {
		klog.V(2).InfoS("[详情页] 开始请求", "url", r.URL.String())
	})

	col.OnError(func(r *colly.Response, err error) {
		klog.ErrorS(err, "[详情页] 请求失败", "url", r.Request.URL.String(), "statusCode", r.StatusCode)
		c.recordError(fmt.Errorf("详情页面请求失败 [%s]: %w", r.Request.URL.String(), err))
	})

	col.OnHTML("body", func(e *colly.HTMLElement) {
		c.parseDetailPage(e)
	})

	return col
}

// parseListItem 解析列表项
func (c *Crawler) parseListItem(e *colly.HTMLElement) *Attraction {
	attraction := &Attraction{}

	attraction.Name = strings.TrimSpace(e.ChildText("div.title b"))
	if attraction.Name == "" {
		attraction.Name = strings.TrimSpace(e.ChildAttr("a.pic img", "alt"))
	}

	attraction.ImageURL = e.ChildAttr("a.pic img", "data-original")
	if attraction.ImageURL == "" {
		attraction.ImageURL = e.ChildAttr("a img", "data-original")
	}
	if attraction.ImageURL != "" && !strings.HasPrefix(attraction.ImageURL, "http") {
		attraction.ImageURL = "https:" + attraction.ImageURL
	}

	href := e.ChildAttr("a.pic", "href")
	if href == "" {
		href = e.ChildAttr("a", "href")
	}

	re := regexp.MustCompile(`/jingdian/([^/\.]+)`)
	matches := re.FindStringSubmatch(href)
	if len(matches) > 1 {
		attraction.SourceURL = matches[1]
	}

	return attraction
}

// parseDetailPage 解析详情页面
func (c *Crawler) parseDetailPage(e *colly.HTMLElement) {
	currentURL := e.Request.URL.String()
	re := regexp.MustCompile(`/jingdian/([^/\.]+)`)
	matches := re.FindStringSubmatch(currentURL)
	if len(matches) <= 1 {
		klog.Warning("[详情页] 无法从URL提取sourceID", "url", currentURL)
		return
	}
	sourceID := matches[1]

	c.mu.Lock()
	attraction, exists := c.attractions[sourceID]
	if !exists {
		attraction = &Attraction{
			SourceURL: sourceID,
		}
		c.attractions[sourceID] = attraction
	}
	c.mu.Unlock()

	description := c.extractDescription(e)
	attraction.Description = description

	klog.InfoS("[详情页] 解析完成",
		"sourceID", sourceID,
		"name", attraction.Name,
		"descriptionLength", len(description),
		"hasDescription", description != "")

	c.dbChan <- attraction
}

// extractDescription 提取景点描述
func (c *Crawler) extractDescription(e *colly.HTMLElement) string {
	var description strings.Builder

	// 1. 尝试提取景点地址、开放时间、门票等结构化信息
	address := strings.TrimSpace(e.ChildText("div.info p:contains('景点地址')"))
	if address == "" {
		address = c.extractInfoByLabel(e, "景点地址")
	}
	if address != "" {
		description.WriteString(address)
		description.WriteString("\n")
	}

	openTime := strings.TrimSpace(e.ChildText("div.info p:contains('开放时间')"))
	if openTime == "" {
		openTime = c.extractInfoByLabel(e, "开放时间")
	}
	if openTime != "" {
		description.WriteString(openTime)
		description.WriteString("\n")
	}

	ticket := strings.TrimSpace(e.ChildText("div.info p:contains('门票信息')"))
	if ticket == "" {
		ticket = c.extractInfoByLabel(e, "门票信息")
	}
	if ticket != "" {
		description.WriteString(ticket)
		description.WriteString("\n")
	}

	// 2. 提取主要介绍内容 - 根据网页结构，尝试多种选择器
	introSelectors := []string{
		"div.intro",
		"div.content div.intro",
		"div.main div.intro",
		"div.spot_intro",
		"div.detail_intro",
		"div.article",
		"div.content",
		"div.detail",
		"div.text",
	}

	for _, selector := range introSelectors {
		e.ForEach(selector, func(_ int, el *colly.HTMLElement) {
			text := strings.TrimSpace(el.Text)
			if text != "" && len(text) > 10 {
				description.WriteString(text)
				description.WriteString("\n")
			}
		})
		if description.Len() > 200 {
			break
		}
	}

	// 3. 如果还是没有内容，尝试提取所有段落
	if description.Len() == 0 {
		e.ForEach("p", func(_ int, pEl *colly.HTMLElement) {
			text := strings.TrimSpace(pEl.Text)
			if text != "" && len(text) > 20 {
				description.WriteString(text)
				description.WriteString("\n")
			}
		})
	}

	// 4. 最后尝试提取整个主要内容区域
	if description.Len() == 0 {
		mainText := strings.TrimSpace(e.ChildText("div.main"))
		if mainText != "" && len(mainText) > 50 {
			description.WriteString(mainText)
		}
	}

	return strings.TrimSpace(description.String())
}

// extractInfoByLabel 通过标签文本提取信息
func (c *Crawler) extractInfoByLabel(e *colly.HTMLElement, label string) string {
	var result string
	e.ForEach("p, div, span", func(_ int, el *colly.HTMLElement) {
		text := el.Text
		if strings.Contains(text, label) {
			result = strings.TrimSpace(text)
		}
	})
	return result
}

// recordError 记录错误
func (c *Crawler) recordError(err error) {
	c.errorMu.Lock()
	defer c.errorMu.Unlock()
	c.errors = append(c.errors, err)
}

// GetErrors 获取所有错误
func (c *Crawler) GetErrors() []error {
	c.errorMu.Lock()
	defer c.errorMu.Unlock()
	return append([]error(nil), c.errors...)
}

// dbWorker 数据库写入工作协程
func (c *Crawler) dbWorker(ctx context.Context) {
	klog.InfoS("[数据库] 写入工作协程启动")
	defer klog.InfoS("[数据库] 写入工作协程结束")

	batch := make([]*Attraction, 0, batchSize)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}
		klog.InfoS("[数据库] 批量写入", "count", len(batch))
		if err := c.batchSaveToDB(ctx, batch); err != nil {
			klog.ErrorS(err, "[数据库] 批量写入失败", "count", len(batch))
			c.recordError(fmt.Errorf("批量写入数据库失败: %w", err))
		}
		batch = batch[:0]
	}

	for {
		select {
		case <-ctx.Done():
			flush()
			return
		case attraction, ok := <-c.dbChan:
			if !ok {
				flush()
				return
			}
			batch = append(batch, attraction)
			if len(batch) >= batchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

// batchSaveToDB 批量保存到数据库
func (c *Crawler) batchSaveToDB(ctx context.Context, attractions []*Attraction) error {
	if len(attractions) == 0 {
		return nil
	}

	for _, attr := range attractions {
		var existing Attraction
		result := c.db.WithContext(ctx).Where("source_url = ?", attr.SourceURL).First(&existing)
		if result.Error == nil {
			if err := c.db.WithContext(ctx).Model(&existing).Updates(map[string]interface{}{
				"name":        attr.Name,
				"image_url":   attr.ImageURL,
				"description": attr.Description,
				"page_number": attr.PageNumber,
			}).Error; err != nil {
				klog.ErrorS(err, "[数据库] 更新景点数据失败", "sourceURL", attr.SourceURL, "name", attr.Name)
				c.recordError(fmt.Errorf("更新景点数据失败 [%s]: %w", attr.SourceURL, err))
			} else {
				klog.V(2).InfoS("[数据库] 更新景点数据成功", "sourceURL", attr.SourceURL, "name", attr.Name)
			}
		} else {
			if err := c.db.WithContext(ctx).Create(attr).Error; err != nil {
				klog.ErrorS(err, "[数据库] 创建景点数据失败", "sourceURL", attr.SourceURL, "name", attr.Name)
				c.recordError(fmt.Errorf("创建景点数据失败 [%s]: %w", attr.SourceURL, err))
			} else {
				klog.V(2).InfoS("[数据库] 创建景点数据成功", "sourceURL", attr.SourceURL, "name", attr.Name)
			}
		}
	}

	return nil
}

// Run 启动爬虫
func (c *Crawler) Run(ctx context.Context) error {
	klog.InfoS("========================================")
	klog.InfoS("开始爬取重庆景点数据",
		"maxPages", maxPages,
		"baseURL", baseURL,
		"workerLimit", workerLimit,
		"batchSize", batchSize,
	)
	klog.InfoS("========================================")

	if err := AutoMigrate(c.db); err != nil {
		return fmt.Errorf("数据库迁移失败: %w", err)
	}

	// 启动数据库写入工作协程
	go c.dbWorker(ctx)

	startTime := time.Now()

	for page := 1; page <= maxPages; page++ {
		select {
		case <-ctx.Done():
			klog.InfoS("爬取被取消", "reason", ctx.Err())
			c.cancel()
			return ctx.Err()
		default:
		}

		var pageURL string
		if page == 1 {
			pageURL = baseURL
		} else {
			pageURL = fmt.Sprintf("%s1-%d-0-0.html", baseURL, page)
		}

		klog.InfoS("[列表页] 开始爬取", "page", page, "url", pageURL)

		if err := c.collector.Visit(pageURL); err != nil {
			klog.ErrorS(err, "[列表页] 访问失败", "page", page, "url", pageURL)
			c.recordError(fmt.Errorf("访问第%d页失败: %w", page, err))
			continue
		}

		c.collector.Wait()

		c.mu.RLock()
		pageCount := 0
		for _, attr := range c.attractions {
			if attr.PageNumber == 0 || attr.PageNumber == page {
				attr.PageNumber = page
				pageCount++
			}
		}
		c.mu.RUnlock()

		klog.InfoS("[列表页] 爬取完成", "page", page, "attractionsFound", pageCount)
	}

	klog.InfoS("等待所有详情页爬取完成...")
	c.wg.Wait()
	c.detailCollector.Wait()

	// 关闭数据库通道，等待写入完成
	close(c.dbChan)
	time.Sleep(2 * time.Second)

	duration := time.Since(startTime)
	c.mu.RLock()
	totalAttractions := len(c.attractions)
	c.mu.RUnlock()

	klog.InfoS("========================================")
	klog.InfoS("爬取完成",
		"totalAttractions", totalAttractions,
		"errors", len(c.errors),
		"duration", duration.String(),
	)
	klog.InfoS("========================================")

	return nil
}

// GetAttractions 获取所有景点
func (c *Crawler) GetAttractions() []*Attraction {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]*Attraction, 0, len(c.attractions))
	for _, attr := range c.attractions {
		result = append(result, attr)
	}
	return result
}

// Close 关闭爬虫
func (c *Crawler) Close() {
	klog.InfoS("关闭爬虫")
	c.cancel()
}

// RunCrawler 运行爬虫的便捷函数
func RunCrawler(ctx context.Context, db *gorm.DB, rateLimit time.Duration) error {
	crawler := NewCrawler(db)
	defer crawler.Close()

	return crawler.Run(ctx)
}
