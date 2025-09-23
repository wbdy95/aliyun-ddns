#!/bin/bash

# Aliyun DDNS 状态检查脚本

SERVICE_NAME="aliyun-ddns"
INSTALL_DIR="/opt/aliyun-ddns"

echo "=== Aliyun DDNS 状态检查 ==="
echo ""

# 检查服务状态
echo "1. 服务状态:"
if systemctl is-active --quiet $SERVICE_NAME; then
    echo "   ✓ 服务正在运行"
    echo "   启动时间: $(systemctl show $SERVICE_NAME --property=ActiveEnterTimestamp --value)"
else
    echo "   ✗ 服务未运行"
fi

if systemctl is-enabled --quiet $SERVICE_NAME; then
    echo "   ✓ 开机自启已启用"
else
    echo "   ✗ 开机自启未启用"
fi

echo ""

# 检查安装状态
echo "2. 安装状态:"
if [[ -d "$INSTALL_DIR" ]]; then
    echo "   ✓ 安装目录存在: $INSTALL_DIR"
    if [[ -f "$INSTALL_DIR/aliyun-ddns" ]]; then
        echo "   ✓ 可执行文件存在"
    else
        echo "   ✗ 可执行文件不存在"
    fi
    if [[ -f "$INSTALL_DIR/config.json" ]]; then
        echo "   ✓ 配置文件存在"
    else
        echo "   ✗ 配置文件不存在"
    fi
else
    echo "   ✗ 安装目录不存在"
fi

echo ""

# 检查最近日志
echo "3. 最近日志 (最后10行):"
if systemctl is-active --quiet $SERVICE_NAME; then
    journalctl -u $SERVICE_NAME --no-pager -n 10
else
    echo "   服务未运行，无法获取日志"
fi

echo ""

# 检查网络连接
echo "4. 网络连接测试:"
echo -n "   测试获取公网IP ... "
if ip=$(timeout 5 curl -s http://ip.3322.net/ 2>/dev/null); then
    echo "成功: $ip"
else
    echo "失败"
fi

echo -n "   测试阿里云API连接 ... "
if timeout 5 curl -s https://alidns.aliyuncs.com >/dev/null 2>&1; then
    echo "成功"
else
    echo "失败"
fi

echo ""
echo "=== 检查完成 ==="

# 如果服务有问题，提供解决建议
if ! systemctl is-active --quiet $SERVICE_NAME; then
    echo ""
    echo "服务未运行，可以尝试:"
    echo "  sudo systemctl start $SERVICE_NAME     # 启动服务"
    echo "  sudo systemctl status $SERVICE_NAME    # 查看详细状态"
    echo "  sudo journalctl -u $SERVICE_NAME -f    # 查看实时日志"
fi