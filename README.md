# Multi-DDNS - 多提供商动态DNS客户端

一个用Go语言编写的多提供商DDNS客户端，支持阿里云DNS和Cloudflare，自动更新域名解析记录，兼容老旧Linux系统。

## 功能特性

- ✅ 自动检测公网IP变化
- ✅ 支持多个DNS提供商（阿里云、Cloudflare）
- ✅ 一个配置文件可同时配置多个提供商
- ✅ 支持多种IP检测服务，提高可靠性
- ✅ 支持老旧Linux系统（Go 1.13+）
- ✅ 支持多种架构（amd64, 386, arm, arm64）
- ✅ systemd服务集成
- ✅ 详细的日志记录
- ✅ 向后兼容旧版配置格式
- ✅ 安全的配置文件权限管理

## 系统要求

- Linux系统
- 网络连接
- 阿里云域名和API密钥（使用阿里云时）
- Cloudflare账号和API Token（使用Cloudflare时）

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

#### 新版配置格式（推荐）

支持多提供商配置：

```json
{
  "providers": [
    {
      "type": "aliyun",
      "access_key_id": "你的阿里云AccessKey ID",
      "access_key_secret": "你的阿里云AccessKey Secret",
      "domain_name": "example.com",
      "sub_domain": "www",
      "record_type": "A",
      "ttl": 600
    },
    {
      "type": "cloudflare",
      "api_token": "你的Cloudflare API Token",
      "zone_id": "你的Cloudflare Zone ID（可选，程序会自动获取）",
      "domain_name": "example.com",
      "sub_domain": "www",
      "record_type": "A",
      "ttl": 600
    }
  ],
  "check_interval": 300
}
```

#### 旧版配置格式（向后兼容）

```json
{
  "access_key_id": "你的阿里云AccessKey ID",
  "access_key_secret": "你的阿里云AccessKey Secret",
  "domain_name": "example.com",
  "sub_domain": "home",
  "record_type": "A",
  "ttl": 600,
  "check_interval": 300
}
```

### 3. 运行

```bash
# 直接运行
./bin/aliyun-ddns config.json

# 或者安装为系统服务
sudo ./install.sh
```

## 获取API密钥

### 阿里云

1. 登录阿里云控制台
2. 访问 [AccessKey管理页面](https://ram.console.aliyun.com/manage/ak)
3. 创建AccessKey，记录AccessKey ID和AccessKey Secret
4. 确保账户有域名DNS管理权限

### Cloudflare

1. 登录 [Cloudflare Dashboard](https://dash.cloudflare.com/)
2. 进入 "My Profile" → "API Tokens"
3. 创建一个具有以下权限的Token：
   - Zone - DNS - Edit
   - Zone - Zone - Read
4. （可选）获取Zone ID：
   - 进入你的域名设置
   - 在右侧边栏可以找到Zone ID
   - 或者在API URL中查看

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

## 配置说明

### 提供商类型（type）

- `aliyun`: 阿里云DNS
- `cloudflare`: Cloudflare DNS

### 通用参数

- `domain_name`: 主域名
- `sub_domain`: 子域名
- `record_type`: 记录类型（A表示IPv4，AAAA表示IPv6）
- `ttl`: DNS缓存时间（秒）

### 阿里云特有参数

- `access_key_id`: 阿里云AccessKey ID
- `access_key_secret`: 阿里云AccessKey Secret

### Cloudflare特有参数

- `api_token`: Cloudflare API Token
- `zone_id`: Cloudflare Zone ID（可选，程序会自动获取）

### 全局参数

- `check_interval`: IP检查间隔（秒）

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

3. **API错误（阿里云）**
   - 检查AccessKey是否正确
   - 确保账户有DNS管理权限
   - 检查域名是否已添加到阿里云

4. **API错误（Cloudflare）**
   - 检查API Token是否正确
   - 确保Token有DNS编辑权限
   - 检查域名是否已添加到Cloudflare

5. **DNS记录不存在**
   - 程序会自动尝试创建DNS记录
   - 确保域名已托管到对应的DNS提供商

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
