@echo off
chcp 65001 >nul
title Swagger 文档生成工具

:: 生成 Swagger 文档的批处理脚本
:: 生成的文档将保存到 api/swagger 目录

echo ========================================
echo       Swagger 文档生成脚本
echo ========================================
echo.

:: 获取项目根目录 (脚本在 scripts/swagger/ 目录下，需要向上退两级)
set "SCRIPT_DIR=%~dp0"
cd /d "%SCRIPT_DIR%\..\.."
set "PROJECT_ROOT=%CD%"
set "SWAGGER_DIR=%PROJECT_ROOT%\api\swagger\docs"

echo [1/4] 检查 swag 工具...
where swag >nul 2>&1
if %errorlevel% neq 0 (
    echo swag 工具未安装，正在安装...
    go install github.com/swaggo/swag/cmd/swag@latest
    if %errorlevel% neq 0 (
        echo [错误] swag 工具安装失败
        exit /b 1
    )
    echo [成功] swag 工具安装成功!
) else (
    echo [成功] swag 工具已安装
)

echo.
echo [2/4] 准备输出目录...
if not exist "%SWAGGER_DIR%" (
    mkdir "%SWAGGER_DIR%"
    echo [成功] 创建目录: %SWAGGER_DIR%
) else (
    echo [成功] 输出目录已存在: %SWAGGER_DIR%
)

echo.
echo [3/4] 生成 Swagger 文档...
echo 执行命令: swag init -g cmd/main.go -o api/swagger/docs
swag init -g cmd/main.go -o api/swagger/docs

if %errorlevel% neq 0 (
    echo [错误] Swagger 文档生成失败
    exit /b 1
)

echo.
echo [4/4] 验证生成的文件...

set "ALL_EXIST=true"

if exist "%SWAGGER_DIR%\swagger.yaml" (
    for %%F in ("%SWAGGER_DIR%\swagger.yaml") do echo [成功] swagger.yaml (%%~zF bytes)
) else (
    echo [失败] swagger.yaml (未找到)
    set "ALL_EXIST=false"
)

if exist "%SWAGGER_DIR%\swagger.json" (
    for %%F in ("%SWAGGER_DIR%\swagger.json") do echo [成功] swagger.json (%%~zF bytes)
) else (
    echo [失败] swagger.json (未找到)
    set "ALL_EXIST=false"
)

if exist "%SWAGGER_DIR%\docs.go" (
    for %%F in ("%SWAGGER_DIR%\docs.go") do echo [成功] docs.go (%%~zF bytes)
) else (
    echo [失败] docs.go (未找到)
    set "ALL_EXIST=false"
)

echo.
echo ========================================
if "%ALL_EXIST%"=="true" (
    echo   Swagger 文档生成成功!
    echo   输出目录: %SWAGGER_DIR%
) else (
    echo   Swagger 文档生成不完整
)
echo ========================================

pause
