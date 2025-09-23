#!/bin/bash

# Aliyun DDNS 测试脚本

set -e

echo "=== Aliyun DDNS 测试脚本 ==="
echo ""

# 检查构建文件
echo "1. 检查构建文件..."
if [[ ! -f "bin/aliyun-ddns" ]]; then
    echo "错误: 未找到可执行文件，请先运行 ./build.sh"
    exit 1
fi
echo "✓ 可执行文件存在"

# 检查配置文件
echo ""
echo "2. 检查配置文件..."
if [[ ! -f "config.json.example" ]]; then
    echo "错误: 未找到配置文件模板"
    exit 1
fi
echo "✓ 配置文件模板存在"

# 创建测试配置文件
echo ""
echo "3. 创建测试配置文件..."
if [[ ! -f "config.json" ]]; then
    cp config.json.example config.json
    echo "✓ 已创建 config.json，请编辑填入真实配置"
else
    echo "✓ config.json 已存在"
fi

# 测试IP获取功能
echo ""
echo "4. 测试IP获取功能..."
echo "正在获取当前公网IP..."

# 使用curl测试多个IP服务
ip_services=("http://ip.3322.net/" "http://icanhazip.com/" "http://ipinfo.io/ip")
for service in "${ip_services[@]}"; do
    echo -n "  测试 $service ... "
    if timeout 10 curl -s "$service" > /dev/null 2>&1; then
        ip=$(timeout 10 curl -s "$service" | tr -d '\n\r ')
        echo "成功: $ip"
    else
        echo "失败"
    fi
done

# 检查网络连接
echo ""
echo "5. 测试阿里云API连接..."
echo -n "  测试连接 alidns.aliyuncs.com ... "
if timeout 10 curl -s "https://alidns.aliyuncs.com" > /dev/null 2>&1; then
    echo "成功"
else
    echo "失败（可能需要检查网络或防火墙）"
fi

# 验证程序语法
echo ""
echo "6. 验证程序..."
echo -n "  检查程序可执行性 ... "
if ./bin/aliyun-ddns 2>&1 | grep -q "用法"; then
    echo "成功"
else
    echo "失败"
fi

echo ""
echo "=== 测试完成 ==="
echo ""
echo "接下来的步骤："
echo "1. 编辑 config.json 填入真实的阿里云配置"
echo "2. 运行: ./bin/aliyun-ddns config.json"
echo "3. 或安装为系统服务: sudo ./install.sh"