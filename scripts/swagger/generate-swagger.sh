#!/bin/bash

# 生成 Swagger 文档的 Shell 脚本
# 生成的文档将保存到 api/swagger 目录

set -e

# 获取项目根目录 (脚本在 scripts/swagger/ 目录下，需要向上退两级)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
SWAGGER_DIR="$PROJECT_ROOT/api/swagger/docs"

echo "========================================"
echo "      Swagger 文档生成脚本"
echo "========================================"
echo ""

# 检查 swag 工具是否已安装
echo "[1/4] 检查 swag 工具..."
if ! command -v swag &> /dev/null; then
    echo "swag 工具未安装，正在安装..."
    go install github.com/swaggo/swag/cmd/swag@latest
    if [ $? -ne 0 ]; then
        echo "错误: swag 工具安装失败"
        exit 1
    fi
    echo "swag 工具安装成功!"
else
    echo "swag 工具已安装: $(command -v swag)"
fi

# 确保 api/swagger/docs 目录存在
echo ""
echo "[2/4] 准备输出目录..."
if [ ! -d "$SWAGGER_DIR" ]; then
    mkdir -p "$SWAGGER_DIR"
    echo "创建目录: $SWAGGER_DIR"
else
    echo "输出目录已存在: $SWAGGER_DIR"
fi

# 进入项目根目录
cd "$PROJECT_ROOT"

# 生成 Swagger 文档
echo ""
echo "[3/4] 生成 Swagger 文档..."
echo "执行命令: swag init -g cmd/main.go -o api/swagger/docs"

swag init -g cmd/main.go -o api/swagger/docs

if [ $? -ne 0 ]; then
    echo "错误: Swagger 文档生成失败"
    exit 1
fi

echo ""
echo "[4/4] 验证生成的文件..."

# 检查生成的文件
GENERATED_FILES=("swagger.yaml" "swagger.json" "docs.go")

ALL_EXIST=true
for file in "${GENERATED_FILES[@]}"; do
    FILE_PATH="$SWAGGER_DIR/$file"
    if [ -f "$FILE_PATH" ]; then
        FILE_SIZE=$(stat -f%z "$FILE_PATH" 2>/dev/null || stat -c%s "$FILE_PATH" 2>/dev/null)
        echo "  ✓ $file ($FILE_SIZE bytes)"
    else
        echo "  ✗ $file (未找到)"
        ALL_EXIST=false
    fi
done

echo ""
echo "========================================"
if [ "$ALL_EXIST" = true ]; then
    echo "  Swagger 文档生成成功!"
    echo "  输出目录: $SWAGGER_DIR"
else
    echo "  Swagger 文档生成不完整"
fi
echo "========================================"
