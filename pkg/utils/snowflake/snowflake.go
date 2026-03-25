package snowflake

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/bwmarrin/snowflake"
)

var (
	once   sync.Once
	node   *snowflake.Node
	nodeID int64 = 1 // 默认节点ID，可以通过配置修改
)

// Init 初始化雪花算法生成器
// nodeID: 节点ID，范围 0-1023，确保每个节点使用不同的ID
func Init(id int64) error {
	if id < 0 || id > 1023 {
		return fmt.Errorf("node ID must be between 0 and 1023, got: %d", id)
	}
	nodeID = id

	var err error
	once.Do(func() {
		node, err = snowflake.NewNode(nodeID)
		if err != nil {
			err = fmt.Errorf("failed to create snowflake node: %w", err)
		}
	})

	return err
}

// GenerateID 生成一个雪花算法ID并返回字符串格式
func GenerateID() (string, error) {
	if node == nil {
		// 如果未初始化，使用默认节点ID初始化
		if err := Init(nodeID); err != nil {
			return "", fmt.Errorf("snowflake not initialized: %w", err)
		}
	}

	id := node.Generate()
	return strconv.FormatInt(id.Int64(), 10), nil
}

// GenerateIDInt64 生成一个雪花算法ID并返回int64格式
func GenerateIDInt64() (int64, error) {
	if node == nil {
		// 如果未初始化，使用默认节点ID初始化
		if err := Init(nodeID); err != nil {
			return 0, fmt.Errorf("snowflake not initialized: %w", err)
		}
	}

	id := node.Generate()
	return id.Int64(), nil
}
