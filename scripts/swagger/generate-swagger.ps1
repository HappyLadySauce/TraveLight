# 生成 Swagger 文档的 PowerShell 脚本
# 生成的文档将保存到 api/swagger 目录

$ErrorActionPreference = "Stop"

# 获取项目根目录 (脚本在 scripts/swagger/ 目录下，需要向上退两级)
$ProjectRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$SwaggerDir = Join-Path $ProjectRoot "api\swagger\docs"

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "      Swagger 文档生成脚本" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# 检查 swag 工具是否已安装
Write-Host "[1/4] 检查 swag 工具..." -ForegroundColor Yellow
$swagCmd = Get-Command swag -ErrorAction SilentlyContinue

if (-not $swagCmd) {
    Write-Host "swag 工具未安装，正在安装..." -ForegroundColor Yellow
    go install github.com/swaggo/swag/cmd/swag@latest
    if ($LASTEXITCODE -ne 0) {
        Write-Error "swag 工具安装失败"
        exit 1
    }
    Write-Host "swag 工具安装成功!" -ForegroundColor Green
} else {
    Write-Host "swag 工具已安装: $($swagCmd.Source)" -ForegroundColor Green
}

# 确保 api/swagger/docs 目录存在
Write-Host ""
Write-Host "[2/4] 准备输出目录..." -ForegroundColor Yellow
if (-not (Test-Path $SwaggerDir)) {
    New-Item -ItemType Directory -Path $SwaggerDir -Force | Out-Null
    Write-Host "创建目录: $SwaggerDir" -ForegroundColor Green
} else {
    Write-Host "输出目录已存在: $SwaggerDir" -ForegroundColor Green
}

# 进入项目根目录
Set-Location $ProjectRoot

# 生成 Swagger 文档
Write-Host ""
Write-Host "[3/4] 生成 Swagger 文档..." -ForegroundColor Yellow
Write-Host "执行命令: swag init -g cmd/main.go -o api/swagger/docs" -ForegroundColor Gray

swag init -g cmd/main.go -o api/swagger/docs

if ($LASTEXITCODE -ne 0) {
    Write-Error "Swagger 文档生成失败"
    exit 1
}

Write-Host ""
Write-Host "[4/4] 验证生成的文件..." -ForegroundColor Yellow

# 检查生成的文件
$generatedFiles = @(
    "swagger.yaml",
    "swagger.json",
    "docs.go"
)

$allExist = $true
foreach ($file in $generatedFiles) {
    $filePath = Join-Path $SwaggerDir $file
    if (Test-Path $filePath) {
        $fileSize = (Get-Item $filePath).Length
        Write-Host "  ✓ $file ($fileSize bytes)" -ForegroundColor Green
    } else {
        Write-Host "  ✗ $file (未找到)" -ForegroundColor Red
        $allExist = $false
    }
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
if ($allExist) {
    Write-Host "  Swagger 文档生成成功!" -ForegroundColor Green
    Write-Host "  输出目录: $SwaggerDir" -ForegroundColor Green
} else {
    Write-Host "  Swagger 文档生成不完整" -ForegroundColor Yellow
}
Write-Host "========================================" -ForegroundColor Cyan
