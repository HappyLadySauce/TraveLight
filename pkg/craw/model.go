package craw

import (
	"time"

	"gorm.io/gorm"
)

// Attraction 景点数据模型
// 用于存储从欣欣旅游网爬取的重庆景点信息
type Attraction struct {
	ID          uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string         `gorm:"type:varchar(255);not null;comment:景点名称" json:"name"`
	ImageURL    string         `gorm:"type:varchar(500);comment:景点图片URL" json:"image_url"`
	Description string         `gorm:"type:text;comment:景点详情介绍" json:"description"`
	SourceURL   string         `gorm:"type:varchar(500);comment:数据来源URL" json:"source_url"`
	PageNumber  int            `gorm:"type:integer;comment:所在页码" json:"page_number"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定表名
func (Attraction) TableName() string {
	return "attractions"
}

// AutoMigrate 自动迁移数据库表结构
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&Attraction{})
}
