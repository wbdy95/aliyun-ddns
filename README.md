# Aliyun DDNS - 阿里云动态DNS客户端

一个用Go语言编写的阿里云DDNS客户端，支持自动更新域名解析记录，兼容老旧Linux系统。

## 功能特性

- ✅ 自动检测公网IP变化
- ✅ 自动更新阿里云DNS解析记录
- ✅ 支持多种IP检测服务，提高可靠性
- ✅ 支持老旧Linux系统（Go 1.13+）
- ✅ 支持多种架构（amd64, 386, arm, arm64）
- ✅ systemd服务集成
- ✅ 详细的日志记录
- ✅ 安全的配置文件权限管理

## 系统要求

- Linux系统
- 网络连接
- 阿里云域名和API密钥

## 快速开始

### 1. 下载和构建

```bash
# 克隆或下载源码
git clone <repository-url>
cd aliyun-ddns

# 构建程序
chmod +x build.sh
./build.sh
```

### 2. 配置

```bash
# 复制配置文件
cp config.json.example config.json

# 编辑配置文件
nano config.json
```

配置文件说明：

```json
{
  "access_key_id": "你的阿里云AccessKey ID",
  "access_key_secret": "你的阿里云AccessKey Secret", 
  "domain_name": "example.com",        // 主域名
  "sub_domain": "home",                // 子域名，最终会解析 home.example.com
  "record_type": "A",                  // 记录类型，通常为A
  "ttl": 600,                         // TTL值，单位秒
  "check_interval": 300               // 检查间隔，单位秒（5分钟）
}
```

### 3. 运行

```bash
# 直接运行
./bin/aliyun-ddns config.json

# 或者安装为系统服务
sudo ./install.sh
```

## 获取阿里云API密钥

1. 登录阿里云控制台
2. 访问 [AccessKey管理页面](https://ram.console.aliyun.com/manage/ak)
3. 创建AccessKey，记录AccessKey ID和AccessKey Secret
4. 确保账户有域名DNS管理权限

## 安装为系统服务

### 自动安装（推荐）

```bash
sudo ./install.sh
```

### 手动安装

1. 复制程序到系统目录：
```bash
sudo mkdir -p /opt/aliyun-ddns
sudo cp bin/aliyun-ddns /opt/aliyun-ddns/
sudo cp config.json /opt/aliyun-ddns/
```

2. 创建专用用户：
```bash
sudo useradd --system --home-dir /opt/aliyun-ddns --shell /usr/sbin/nologin ddns
sudo chown -R ddns:ddns /opt/aliyun-ddns
```

3. 安装systemd服务：
```bash
sudo cp aliyun-ddns.service /etc/systemd/system/
sudo systemctl daemon-reload
```

4. 启动服务：
```bash
sudo systemctl start aliyun-ddns
sudo systemctl enable aliyun-ddns
```

## 服务管理

```bash
# 查看服务状态
sudo systemctl status aliyun-ddns

# 查看实时日志
sudo journalctl -u aliyun-ddns -f

# 重启服务
sudo systemctl restart aliyun-ddns

# 停止服务
sudo systemctl stop aliyun-ddns

# 禁用开机自启
sudo systemctl disable aliyun-ddns
```

## 多架构支持

构建脚本会自动生成多个架构的可执行文件：

- `aliyun-ddns-linux-amd64`: 64位Linux系统
- `aliyun-ddns-linux-386`: 32位Linux系统  
- `aliyun-ddns-linux-arm`: ARM设备（如树莓派等）
- `aliyun-ddns-linux-arm64`: ARM64设备

根据你的系统架构选择对应的可执行文件。

## 老旧系统支持

本程序使用Go 1.13编译，支持以下老旧系统：

- CentOS 6/7
- Ubuntu 14.04+
- Debian 8+
- 其他支持glibc 2.17+的Linux发行版

## 故障排除

### 常见问题

1. **权限错误**
   ```bash
   sudo chown ddns:ddns /opt/aliyun-ddns/config.json
   sudo chmod 600 /opt/aliyun-ddns/config.json
   ```

2. **网络连接问题**
   - 检查防火墙设置
   - 确保可以访问外网
   - 尝试手动访问IP检测服务

3. **API错误**
   - 检查AccessKey是否正确
   - 确保账户有DNS管理权限
   - 检查域名是否已添加到阿里云

4. **DNS记录不存在**
   - 程序会自动尝试创建DNS记录
   - 确保域名已托管到阿里云DNS

### 调试模式

查看详细日志：
```bash
sudo journalctl -u aliyun-ddns -f --no-pager
```

手动运行程序查看输出：
```bash
sudo -u ddns /opt/aliyun-ddns/aliyun-ddns /opt/aliyun-ddns/config.json
```

## 卸载

```bash
# 停止并删除服务
sudo systemctl stop aliyun-ddns
sudo systemctl disable aliyun-ddns
sudo rm /etc/systemd/system/aliyun-ddns.service
sudo systemctl daemon-reload

# 删除程序文件和用户
sudo rm -rf /opt/aliyun-ddns
sudo userdel ddns
```

## 安全说明

- 配置文件包含敏感信息，设置了严格的权限（600）
- 服务以专用用户运行，遵循最小权限原则
- 使用systemd安全特性限制程序权限

## 许可证

MIT License

## 贡献

欢迎提交Issue和Pull Request！