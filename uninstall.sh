#!/bin/bash

# Aliyun DDNS 卸载脚本

set -e

# 检查是否以root权限运行
if [[ $EUID -ne 0 ]]; then
   echo "错误: 此脚本需要root权限运行"
   echo "请使用: sudo $0"
   exit 1
fi

INSTALL_DIR="/opt/aliyun-ddns"
SERVICE_NAME="aliyun-ddns"
USER_NAME="ddns"

echo "开始卸载 Aliyun DDNS..."

# 停止服务
if systemctl is-active --quiet $SERVICE_NAME; then
    echo "停止服务..."
    systemctl stop $SERVICE_NAME
fi

# 禁用服务
if systemctl is-enabled --quiet $SERVICE_NAME; then
    echo "禁用服务..."
    systemctl disable $SERVICE_NAME
fi

# 删除服务文件
if [[ -f "/etc/systemd/system/$SERVICE_NAME.service" ]]; then
    echo "删除服务文件..."
    rm /etc/systemd/system/$SERVICE_NAME.service
    systemctl daemon-reload
fi

# 删除安装目录
if [[ -d "$INSTALL_DIR" ]]; then
    echo "删除安装目录..."
    rm -rf $INSTALL_DIR
fi

# 删除用户
if id "$USER_NAME" &>/dev/null; then
    echo "删除用户 $USER_NAME..."
    userdel $USER_NAME
fi

echo "卸载完成！"