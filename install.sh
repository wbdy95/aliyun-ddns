#!/bin/bash

# Aliyun DDNS 安装脚本

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

echo "开始安装 Aliyun DDNS..."

# 检查可执行文件
if [[ ! -f "bin/aliyun-ddns" ]]; then
    echo "错误: 未找到可执行文件，请先运行 ./build.sh"
    exit 1
fi

# 创建用户
if ! id "$USER_NAME" &>/dev/null; then
    echo "创建用户 $USER_NAME..."
    useradd --system --home-dir $INSTALL_DIR --shell /usr/sbin/nologin $USER_NAME
fi

# 创建安装目录
echo "创建安装目录 $INSTALL_DIR..."
mkdir -p $INSTALL_DIR
chown $USER_NAME:$USER_NAME $INSTALL_DIR

# 复制文件
echo "复制程序文件..."
cp bin/aliyun-ddns $INSTALL_DIR/
chown $USER_NAME:$USER_NAME $INSTALL_DIR/aliyun-ddns
chmod +x $INSTALL_DIR/aliyun-ddns

# 复制配置文件（如果不存在）
if [[ ! -f "$INSTALL_DIR/config.json" ]]; then
    echo "复制配置文件模板..."
    cp config.json.example $INSTALL_DIR/config.json
    chown $USER_NAME:$USER_NAME $INSTALL_DIR/config.json
    chmod 600 $INSTALL_DIR/config.json
    echo "警告: 请编辑 $INSTALL_DIR/config.json 填入你的配置信息"
fi

# 安装systemd服务
echo "安装systemd服务..."
cp aliyun-ddns.service /etc/systemd/system/
systemctl daemon-reload

echo "安装完成！"
echo ""
echo "接下来的步骤："
echo "1. 编辑配置文件: sudo nano $INSTALL_DIR/config.json"
echo "2. 启动服务: sudo systemctl start $SERVICE_NAME"
echo "3. 设置开机自启: sudo systemctl enable $SERVICE_NAME"
echo "4. 查看运行状态: sudo systemctl status $SERVICE_NAME"
echo "5. 查看日志: sudo journalctl -u $SERVICE_NAME -f"
echo ""
echo "卸载命令:"
echo "sudo systemctl stop $SERVICE_NAME"
echo "sudo systemctl disable $SERVICE_NAME"
echo "sudo rm /etc/systemd/system/$SERVICE_NAME.service"
echo "sudo rm -rf $INSTALL_DIR"
echo "sudo userdel $USER_NAME"