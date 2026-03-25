package svc

import (
	"context"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/HappyLadySauce/TraveLight/cmd/app/options"
)

// ServiceContext 服务上下文
type ServiceContext struct {
	Config options.Options
	DB     *gorm.DB
	Redis  *redis.Client
}

// NewServiceContext creates a new ServiceContext
// 创建一个新的服务上下文
// 初始化数据库连接和 Redis 连接
func NewServiceContext(c options.Options) (*ServiceContext, error) {
	// 构建 PostgreSQL DSN（使用 keyword/value 格式）
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=%s",
		c.DatabaseOptions.Host,
		c.DatabaseOptions.Username,
		c.DatabaseOptions.Password,
		c.DatabaseOptions.Database,
		c.DatabaseOptions.Port,
		c.DatabaseOptions.TimeZone,
	)

	// 初始化数据库连接
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	// 获取底层 sql.DB 并配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
	// 设置连接池参数
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)

	// 初始化 Redis 连接
	client := redis.NewClient(&redis.Options{
		Addr:     c.RedisOptions.RedisHost,
		Password: c.RedisOptions.RedisPass,
		DB:       c.RedisOptions.RedisDB,
	})
	// 测试 Redis 连接是否成功
	if _, err := client.Ping(context.Background()).Result(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &ServiceContext{
		Config: c,
		DB:     db,
		Redis:  client,
	}, nil
}

// Close closes the ServiceContext and releases resources
// 关闭一个服务上下文并释放资源
func (s *ServiceContext) Close() error {
	var errs []error

	// 关闭数据库连接
	if s.DB != nil {
		if sqlDB, err := s.DB.DB(); err == nil {
			if err := sqlDB.Close(); err != nil {
				errs = append(errs, fmt.Errorf("failed to close database: %w", err))
			}
		}
	}

	// 关闭 Redis 连接
	if s.Redis != nil {
		if err := s.Redis.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close redis: %w", err))
		}
	}

	return errors.Join(errs...)
}
