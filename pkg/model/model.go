package model

import (
	"time"

	"gorm.io/gorm"
)

const (
	// ContentTypeArticle represents article content.
	// ContentTypeArticle 表示文章类型内容。
	ContentTypeArticle = "article"
	// ContentTypeAttraction represents attraction content.
	// ContentTypeAttraction 表示景点类型内容。
	ContentTypeAttraction = "attraction"
)

// User stores account and profile info.
// User 存储账号与个人资料信息。
type User struct {
	ID           uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	Username     string         `gorm:"type:varchar(64);uniqueIndex;not null" json:"username"`
	PasswordHash string         `gorm:"type:varchar(255);not null" json:"-"`
	Name         string         `gorm:"type:varchar(64);not null" json:"name"`
	Gender       string         `gorm:"type:varchar(16);not null" json:"gender"`
	Phone        string         `gorm:"type:varchar(20);uniqueIndex;not null" json:"phone"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (User) TableName() string {
	return "users"
}

// Article stores UGC article data.
// Article 存储 UGC 文章数据。
type Article struct {
	ID        uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	Title     string         `gorm:"type:varchar(255);not null;index" json:"title"`
	Content   string         `gorm:"type:text;not null" json:"content"`
	AuthorID  uint64         `gorm:"not null;index" json:"author_id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Article) TableName() string {
	return "articles"
}

// Comment stores comment data for article or attraction.
// Comment 存储文章或景点的评论数据。
type Comment struct {
	ID          uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	ContentType string         `gorm:"type:varchar(20);not null;index:idx_content_target,priority:1" json:"content_type"`
	ContentID   uint64         `gorm:"not null;index:idx_content_target,priority:2" json:"content_id"`
	UserID      uint64         `gorm:"not null;index" json:"user_id"`
	Body        string         `gorm:"type:text;not null" json:"body"`
	Status      string         `gorm:"type:varchar(20);not null;default:'active';index" json:"status"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Comment) TableName() string {
	return "comments"
}
