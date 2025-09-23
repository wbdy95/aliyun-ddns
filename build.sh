#!/bin/bash

# Aliyun DDNS 构建脚本
# 支持多架构和老旧机器（使用静态链接，避免GLIBC依赖）

set -euo pipefail

echo "开始构建 Aliyun DDNS..."

# 检查Go环境
if ! command -v go &> /dev/null; then
    echo "错误: 未找到Go环境，请先安装Go"
    exit 1
fi

# 创建输出目录
mkdir -p bin

# 静态构建配置（禁用CGO，使用纯Go解析器）
export CGO_ENABLED=0
BUILDTAGS="-tags netgo,osusergo"

# 构建参数
APP_NAME="aliyun-ddns"
VERSION=$(date +"%Y%m%d%H%M%S")
LDFLAGS="-s -w -X main.Version=${VERSION}"

echo "构建版本: ${VERSION}"

echo "构建当前平台版本(静态)..."
go build -trimpath ${BUILDTAGS} -ldflags "${LDFLAGS}" -o bin/${APP_NAME} .

# 构建常见Linux架构版本（兼容老旧机器）
echo "构建 Linux amd64 版本(静态)..."
GOOS=linux GOARCH=amd64 go build -trimpath ${BUILDTAGS} -ldflags "${LDFLAGS}" -o bin/${APP_NAME}-linux-amd64 .

echo "构建 Linux 386 版本（32位，静态）..."
GOOS=linux GOARCH=386 go build -trimpath ${BUILDTAGS} -ldflags "${LDFLAGS}" -o bin/${APP_NAME}-linux-386 .

echo "构建 Linux arm 版本（ARMv6+，静态）..."
GOOS=linux GOARCH=arm GOARM=6 go build -trimpath ${BUILDTAGS} -ldflags "${LDFLAGS}" -o bin/${APP_NAME}-linux-arm .

echo "构建 Linux arm64 版本（静态）..."
GOOS=linux GOARCH=arm64 go build -trimpath ${BUILDTAGS} -ldflags "${LDFLAGS}" -o bin/${APP_NAME}-linux-arm64 .

# 设置执行权限
chmod +x bin/*

echo "构建完成！生成的文件："
ls -la bin/

echo ""
echo "使用说明："
echo "1. 复制 config.json.example 为 config.json 并填入你的配置"
echo "2. 运行: ./bin/${APP_NAME} config.json"
echo "3. 或安装为系统服务: sudo ./install.sh"