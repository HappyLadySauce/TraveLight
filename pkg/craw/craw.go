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
	detailURLFmt = "https://chongqing.cncn.com/jingdian/%s"
	maxPages     = 22
)

type Crawler struct {
	db            *gorm.DB
	collector     *colly.Collector
	detailCollector *colly.Collector
	attractions   map[string]*Attraction
	mu            sync.RWMutex
	errors        []error
	errorMu       sync.Mutex
	rateLimiter   *time.Ticker
}

type CrawlerOption func(*Crawler)

func WithRateLimit(interval time.Duration) CrawlerOption {
	return func(c *Crawler) {
		c.rateLimiter = time.NewTicker(interval)
	}
}

func NewCrawler(db *gorm.DB, opts ...CrawlerOption) *Crawler {
	c := &Crawler{
		db:          db,
		attractions: make(map[string]*Attraction),
		rateLimiter: time.NewTicker(500 * time.Millisecond),
	}

	for _, opt := range opts {
		opt(c)
	}

	c.collector = c.createListCollector()
	c.detailCollector = c.createDetailCollector()

	return c
}

func (c *Crawler) createListCollector() *colly.Collector {
	col := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
		colly.AllowURLRevisit(),
	)

	col.SetRequestTimeout(30 * time.Second)

	col.Limit(&colly.LimitRule{
		DomainGlob:  "chongqing.cncn.com",
		Delay:       1 * time.Second,
		RandomDelay: 500 * time.Millisecond,
		Parallelism: 2,
	})

	col.OnRequest(func(r *colly.Request) {
		klog.V(2).InfoS("正在访问列表页面", "url", r.URL.String())
	})

	col.OnError(func(r *colly.Response, err error) {
		klog.ErrorS(err, "列表页面请求失败", "url", r.Request.URL.String(), "statusCode", r.StatusCode)
		c.recordError(fmt.Errorf("列表页面请求失败 [%s]: %w", r.Request.URL.String(), err))
	})

	col.OnHTML("div.plist ul li", func(e *colly.HTMLElement) {
		attraction := c.parseListItem(e)
		if attraction != nil && attraction.Name != "" {
			c.mu.Lock()
			c.attractions[attraction.SourceURL] = attraction
			c.mu.Unlock()

			if attraction.SourceURL != "" {
				detailURL := fmt.Sprintf(detailURLFmt, attraction.SourceURL)
				klog.V(2).InfoS("准备爬取详情页面", "url", detailURL, "name", attraction.Name)
				if err := c.detailCollector.Visit(detailURL); err != nil {
					klog.ErrorS(err, "访问详情页面失败", "url", detailURL)
				}
			}
		}
	})

	return col
}

func (c *Crawler) createDetailCollector() *colly.Collector {
	col := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	)

	col.SetRequestTimeout(30 * time.Second)

	col.Limit(&colly.LimitRule{
		DomainGlob:  "chongqing.cncn.com",
		Delay:       800 * time.Millisecond,
		RandomDelay: 400 * time.Millisecond,
		Parallelism: 2,
	})

	col.OnRequest(func(r *colly.Request) {
		klog.V(2).InfoS("正在访问详情页面", "url", r.URL.String())
	})

	col.OnError(func(r *colly.Response, err error) {
		klog.ErrorS(err, "详情页面请求失败", "url", r.Request.URL.String(), "statusCode", r.StatusCode)
		c.recordError(fmt.Errorf("详情页面请求失败 [%s]: %w", r.Request.URL.String(), err))
	})

	col.OnHTML("div.main", func(e *colly.HTMLElement) {
		c.parseDetailPage(e)
	})

	return col
}

func (c *Crawler) parseListItem(e *colly.HTMLElement) *Attraction {
	attraction := &Attraction{}

	attraction.Name = strings.TrimSpace(e.ChildText("div.info h3 a"))

	if attraction.Name == "" {
		attraction.Name = strings.TrimSpace(e.ChildText("h3 a"))
	}

	attraction.ImageURL = e.ChildAttr("div.img a img", "src")
	if attraction.ImageURL == "" {
		attraction.ImageURL = e.ChildAttr("a img", "src")
	}

	if attraction.ImageURL != "" && !strings.HasPrefix(attraction.ImageURL, "http") {
		attraction.ImageURL = "https:" + attraction.ImageURL
	}

	href := e.ChildAttr("div.img a", "href")
	if href == "" {
		href = e.ChildAttr("h3 a", "href")
	}

	re := regexp.MustCompile(`/jingdian/([^/\.]+)`)
	matches := re.FindStringSubmatch(href)
	if len(matches) > 1 {
		attraction.SourceURL = matches[1]
	}

	return attraction
}

func (c *Crawler) parseDetailPage(e *colly.HTMLElement) {
	currentURL := e.Request.URL.String()
	re := regexp.MustCompile(`/jingdian/([^/\.]+)`)
	matches := re.FindStringSubmatch(currentURL)
	if len(matches) <= 1 {
		return
	}
	sourceID := matches[1]

	c.mu.RLock()
	attraction, exists := c.attractions[sourceID]
	c.mu.RUnlock()

	if !exists {
		attraction = &Attraction{
			SourceURL: sourceID,
		}
		c.mu.Lock()
		c.attractions[sourceID] = attraction
		c.mu.Unlock()
	}

	description := c.extractDescription(e)
	attraction.Description = description

	klog.V(1).InfoS("成功解析详情页面", "name", attraction.Name, "descriptionLength", len(description))
}

func (c *Crawler) extractDescription(e *colly.HTMLElement) string {
	var description strings.Builder

	e.ForEach("div.intro", func(_ int, introEl *colly.HTMLElement) {
		text := strings.TrimSpace(introEl.Text)
		if text != "" {
			description.WriteString(text)
			description.WriteString("\n")
		}
	})

	if description.Len() == 0 {
		e.ForEach("div.content", func(_ int, contentEl *colly.HTMLElement) {
			text := strings.TrimSpace(contentEl.Text)
			if text != "" {
				description.WriteString(text)
				description.WriteString("\n")
			}
		})
	}

	if description.Len() == 0 {
		e.ForEach("div.detail", func(_ int, detailEl *colly.HTMLElement) {
			text := strings.TrimSpace(detailEl.Text)
			if text != "" {
				description.WriteString(text)
				description.WriteString("\n")
			}
		})
	}

	if description.Len() == 0 {
		e.ForEach("div.text", func(_ int, textEl *colly.HTMLElement) {
			text := strings.TrimSpace(textEl.Text)
			if text != "" {
				description.WriteString(text)
				description.WriteString("\n")
			}
		})
	}

	if description.Len() == 0 {
		e.ForEach("div.article", func(_ int, articleEl *colly.HTMLElement) {
			text := strings.TrimSpace(articleEl.Text)
			if text != "" {
				description.WriteString(text)
				description.WriteString("\n")
			}
		})
	}

	if description.Len() == 0 {
		e.ForEach("p", func(_ int, pEl *colly.HTMLElement) {
			text := strings.TrimSpace(pEl.Text)
			if text != "" && len(text) > 20 {
				description.WriteString(text)
				description.WriteString("\n")
			}
		})
	}

	return strings.TrimSpace(description.String())
}

func (c *Crawler) recordError(err error) {
	c.errorMu.Lock()
	defer c.errorMu.Unlock()
	c.errors = append(c.errors, err)
}

func (c *Crawler) GetErrors() []error {
	c.errorMu.Lock()
	defer c.errorMu.Unlock()
	return append([]error(nil), c.errors...)
}

func (c *Crawler) Run(ctx context.Context) error {
	klog.InfoS("开始爬取重庆景点数据", "maxPages", maxPages, "baseURL", baseURL)

	if err := AutoMigrate(c.db); err != nil {
		return fmt.Errorf("数据库迁移失败: %w", err)
	}

	for page := 1; page <= maxPages; page++ {
		select {
		case <-ctx.Done():
			klog.InfoS("爬取被取消", "reason", ctx.Err())
			return ctx.Err()
		default:
		}

		var pageURL string
		if page == 1 {
			pageURL = baseURL
		} else {
			pageURL = fmt.Sprintf("%s1-%d-0-0.html", baseURL, page)
		}

		klog.InfoS("正在爬取页面", "page", page, "url", pageURL)

		if err := c.collector.Visit(pageURL); err != nil {
			klog.ErrorS(err, "访问页面失败", "page", page, "url", pageURL)
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

		klog.InfoS("页面爬取完成", "page", page, "attractionsFound", pageCount)

		time.Sleep(1 * time.Second)
	}

	c.detailCollector.Wait()

	klog.InfoS("所有页面爬取完成，开始保存数据到数据库")

	if err := c.saveToDatabase(ctx); err != nil {
		return fmt.Errorf("保存数据到数据库失败: %w", err)
	}

	c.mu.RLock()
	totalAttractions := len(c.attractions)
	c.mu.RUnlock()

	klog.InfoS("爬取完成", "totalAttractions", totalAttractions, "errors", len(c.errors))

	return nil
}

func (c *Crawler) saveToDatabase(ctx context.Context) error {
	c.mu.RLock()
	attractions := make([]*Attraction, 0, len(c.attractions))
	for _, attr := range c.attractions {
		attractions = append(attractions, attr)
	}
	c.mu.RUnlock()

	if len(attractions) == 0 {
		klog.InfoS("没有数据需要保存")
		return nil
	}

	batchSize := 50
	for i := 0; i < len(attractions); i += batchSize {
		end := i + batchSize
		if end > len(attractions) {
			end = len(attractions)
		}

		batch := attractions[i:end]

		for _, attr := range batch {
			var existing Attraction
			result := c.db.WithContext(ctx).Where("source_url = ?", attr.SourceURL).First(&existing)
			if result.Error == nil {
				if err := c.db.WithContext(ctx).Model(&existing).Updates(map[string]interface{}{
					"name":        attr.Name,
					"image_url":   attr.ImageURL,
					"description": attr.Description,
					"page_number": attr.PageNumber,
				}).Error; err != nil {
					klog.ErrorS(err, "更新景点数据失败", "sourceURL", attr.SourceURL)
					c.recordError(fmt.Errorf("更新景点数据失败 [%s]: %w", attr.SourceURL, err))
				}
			} else {
				if err := c.db.WithContext(ctx).Create(attr).Error; err != nil {
					klog.ErrorS(err, "创建景点数据失败", "sourceURL", attr.SourceURL)
					c.recordError(fmt.Errorf("创建景点数据失败 [%s]: %w", attr.SourceURL, err))
				}
			}
		}

		klog.V(1).InfoS("保存批次完成", "batch", i/batchSize+1, "count", len(batch))
	}

	return nil
}

func (c *Crawler) GetAttractions() []*Attraction {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]*Attraction, 0, len(c.attractions))
	for _, attr := range c.attractions {
		result = append(result, attr)
	}
	return result
}

func (c *Crawler) Close() {
	if c.rateLimiter != nil {
		c.rateLimiter.Stop()
	}
}

func RunCrawler(ctx context.Context, db *gorm.DB, rateLimit time.Duration) error {
	crawler := NewCrawler(db, WithRateLimit(rateLimit))
	defer crawler.Close()

	return crawler.Run(ctx)
}
