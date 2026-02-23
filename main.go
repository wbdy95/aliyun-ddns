package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Config 全局配置结构体
type Config struct {
	Providers      []ProviderConfig `json:"providers"`
	CheckInterval  int              `json:"check_interval"`
}

// ProviderConfig DNS提供商配置
type ProviderConfig struct {
	Type            string `json:"type"`             // aliyun 或 cloudflare
	AccessKeyID     string `json:"access_key_id"`    // 阿里云
	AccessKeySecret string `json:"access_key_secret"`// 阿里云
	APIToken        string `json:"api_token"`        // Cloudflare
	ZoneID          string `json:"zone_id"`          // Cloudflare
	DomainName      string `json:"domain_name"`
	SubDomain       string `json:"sub_domain"`
	RecordType      string `json:"record_type"`
	TTL             int    `json:"ttl"`
}

// DNSRecord DNS记录结构体
type DNSRecord struct {
	RecordId string `json:"RecordId"`
	ID       string `json:"id"`
	Value    string `json:"Value"`
	Content  string `json:"content"`
}

// DNSProvider DNS提供商接口
type DNSProvider interface {
	GetName() string
	GetCurrentIP() (string, error)
	GetDNSRecord() (*DNSRecord, error)
	UpdateRecord(recordID, newIP string) error
	AddRecord(ip string) error
}

// ============ 阿里云相关结构体 ============

// AliyunResponse 阿里云API响应结构体
type AliyunResponse struct {
	RequestId    string      `json:"RequestId"`
	DomainRecords struct {
		Record []DNSRecord `json:"Record"`
	} `json:"DomainRecords"`
	RecordId string `json:"RecordId"`
	Code     string `json:"Code"`
	Message  string `json:"Message"`
}

// AliyunProvider 阿里云DNS提供商
type AliyunProvider struct {
	config     ProviderConfig
	httpClient *http.Client
	currentIP  string
	lastIP     string
}

// NewAliyunProvider 创建阿里云提供商实例
func NewAliyunProvider(config ProviderConfig) *AliyunProvider {
	return &AliyunProvider{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetName 获取提供商名称
func (a *AliyunProvider) GetName() string {
	return "阿里云"
}

// GetCurrentIP 获取当前公网IP
func (a *AliyunProvider) GetCurrentIP() (string, error) {
	return getCurrentIP(a.config.RecordType, a.httpClient)
}

// GetDNSRecord 获取DNS记录
func (a *AliyunProvider) GetDNSRecord() (*DNSRecord, error) {
	params := map[string]string{
		"Action":     "DescribeDomainRecords",
		"DomainName": a.config.DomainName,
		"RRKeyWord":  a.config.SubDomain,
		"Type":       a.config.RecordType,
	}

	resp, err := a.makeAPIRequest(params)
	if err != nil {
		return nil, err
	}

	if len(resp.DomainRecords.Record) == 0 {
		return nil, fmt.Errorf("未找到DNS记录: %s.%s", a.config.SubDomain, a.config.DomainName)
	}

	return &resp.DomainRecords.Record[0], nil
}

// UpdateRecord 更新DNS记录
func (a *AliyunProvider) UpdateRecord(recordID, newIP string) error {
	params := map[string]string{
		"Action":   "UpdateDomainRecord",
		"RecordId": recordID,
		"RR":       a.config.SubDomain,
		"Type":     a.config.RecordType,
		"Value":    newIP,
		"TTL":      strconv.Itoa(a.config.TTL),
	}

	_, err := a.makeAPIRequest(params)
	return err
}

// AddRecord 添加DNS记录
func (a *AliyunProvider) AddRecord(ip string) error {
	params := map[string]string{
		"Action":     "AddDomainRecord",
		"DomainName": a.config.DomainName,
		"RR":         a.config.SubDomain,
		"Type":       a.config.RecordType,
		"Value":      ip,
		"TTL":        strconv.Itoa(a.config.TTL),
	}

	_, err := a.makeAPIRequest(params)
	return err
}

// signRequest 生成阿里云API签名
func (a *AliyunProvider) signRequest(params map[string]string) string {
	params["Format"] = "JSON"
	params["Version"] = "2015-01-09"
	params["AccessKeyId"] = a.config.AccessKeyID
	params["SignatureMethod"] = "HMAC-SHA1"
	params["Timestamp"] = time.Now().UTC().Format("2006-01-02T15:04:05Z")
	params["SignatureVersion"] = "1.0"
	params["SignatureNonce"] = strconv.FormatInt(time.Now().UnixNano(), 10)

	var keys []string
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var queryParts []string
	for _, k := range keys {
		queryParts = append(queryParts, url.QueryEscape(k)+"="+url.QueryEscape(params[k]))
	}
	queryString := strings.Join(queryParts, "&")

	stringToSign := "GET&" + url.QueryEscape("/") + "&" + url.QueryEscape(queryString)

	h := hmac.New(sha1.New, []byte(a.config.AccessKeySecret+"&"))
	h.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	return signature
}

// makeAPIRequest 发送API请求
func (a *AliyunProvider) makeAPIRequest(params map[string]string) (*AliyunResponse, error) {
	signature := a.signRequest(params)
	params["Signature"] = signature

	baseURL := "https://alidns.aliyuncs.com/"
	u, _ := url.Parse(baseURL)
	q := u.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	resp, err := a.httpClient.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response AliyunResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	if response.Code != "" && response.Code != "200" {
		return nil, fmt.Errorf("API错误: %s - %s", response.Code, response.Message)
	}

	return &response, nil
}

// ============ Cloudflare相关结构体 ============

// CloudflareDNSRecord Cloudflare DNS记录结构
type CloudflareDNSRecord struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
	ZoneID  string `json:"zone_id"`
}

// CloudflareListResponse Cloudflare列表响应
type CloudflareListResponse struct {
	Success bool `json:"success"`
	Errors  []struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"errors"`
	Result    []CloudflareDNSRecord `json:"result"`
	ResultInfo struct {
		Count   int `json:"count"`
		Total   int `json:"total"`
	} `json:"result_info"`
}

// CloudflareSingleResponse Cloudflare单个记录响应
type CloudflareSingleResponse struct {
	Success bool `json:"success"`
	Errors  []struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"errors"`
	Result CloudflareDNSRecord `json:"result"`
}

// CloudflareZoneResponse Cloudflare Zone响应
type CloudflareZoneResponse struct {
	Success bool `json:"success"`
	Errors  []struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"errors"`
	Result []struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Status   string `json:"status"`
		Paused   bool   `json:"paused"`
		Type     string `json:"type"`
	} `json:"result"`
}

// CloudflareProvider Cloudflare DNS提供商
type CloudflareProvider struct {
	config     ProviderConfig
	httpClient *http.Client
	currentIP  string
	lastIP     string
}

// NewCloudflareProvider 创建Cloudflare提供商实例
func NewCloudflareProvider(config ProviderConfig) *CloudflareProvider {
	return &CloudflareProvider{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetName 获取提供商名称
func (c *CloudflareProvider) GetName() string {
	return "Cloudflare"
}

// GetCurrentIP 获取当前公网IP
func (c *CloudflareProvider) GetCurrentIP() (string, error) {
	return getCurrentIP(c.config.RecordType, c.httpClient)
}

// GetDNSRecord 获取DNS记录
func (c *CloudflareProvider) GetDNSRecord() (*DNSRecord, error) {
	// 如果没有zone_id，先获取
	zoneID := c.config.ZoneID
	if zoneID == "" {
		var err error
		zoneID, err = c.getZoneID()
		if err != nil {
			return nil, err
		}
		c.config.ZoneID = zoneID
	}

	// 构建完整域名
	fqdn := c.config.SubDomain + "." + c.config.DomainName

	// 获取DNS记录
	reqURL := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records?type=%s&name=%s",
		zoneID, c.config.RecordType, fqdn)

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.config.APIToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var cfResp CloudflareListResponse
	err = json.Unmarshal(body, &cfResp)
	if err != nil {
		return nil, err
	}

	if !cfResp.Success {
		if len(cfResp.Errors) > 0 {
			return nil, fmt.Errorf("Cloudflare API错误: %s", cfResp.Errors[0].Message)
		}
		return nil, fmt.Errorf("Cloudflare API错误: 未知错误")
	}

	if len(cfResp.Result) == 0 {
		return nil, fmt.Errorf("未找到DNS记录: %s", fqdn)
	}

	record := cfResp.Result[0]
	return &DNSRecord{
		ID:      record.ID,
		Value:   record.Content,
		Content: record.Content,
	}, nil
}

// UpdateRecord 更新DNS记录
func (c *CloudflareProvider) UpdateRecord(recordID, newIP string) error {
	zoneID := c.config.ZoneID
	if zoneID == "" {
		var err error
		zoneID, err = c.getZoneID()
		if err != nil {
			return err
		}
		c.config.ZoneID = zoneID
	}

	reqURL := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s", zoneID, recordID)

	updateData := map[string]interface{}{
		"type":    c.config.RecordType,
		"name":    c.config.SubDomain + "." + c.config.DomainName,
		"content": newIP,
		"ttl":     c.config.TTL,
	}

	jsonData, err := json.Marshal(updateData)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PATCH", reqURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+c.config.APIToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var cfResp CloudflareSingleResponse
	err = json.Unmarshal(body, &cfResp)
	if err != nil {
		return err
	}

	if !cfResp.Success {
		if len(cfResp.Errors) > 0 {
			return fmt.Errorf("Cloudflare API错误: %s", cfResp.Errors[0].Message)
		}
		return fmt.Errorf("Cloudflare API错误: 未知错误")
	}

	return nil
}

// AddRecord 添加DNS记录
func (c *CloudflareProvider) AddRecord(ip string) error {
	zoneID := c.config.ZoneID
	if zoneID == "" {
		var err error
		zoneID, err = c.getZoneID()
		if err != nil {
			return err
		}
		c.config.ZoneID = zoneID
	}

	reqURL := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records", zoneID)

	createData := map[string]interface{}{
		"type":    c.config.RecordType,
		"name":    c.config.SubDomain + "." + c.config.DomainName,
		"content": ip,
		"ttl":     c.config.TTL,
	}

	jsonData, err := json.Marshal(createData)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", reqURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+c.config.APIToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var cfResp CloudflareSingleResponse
	err = json.Unmarshal(body, &cfResp)
	if err != nil {
		return err
	}

	if !cfResp.Success {
		if len(cfResp.Errors) > 0 {
			return fmt.Errorf("Cloudflare API错误: %s", cfResp.Errors[0].Message)
		}
		return fmt.Errorf("Cloudflare API错误: 未知错误")
	}

	return nil
}

// getZoneID 获取Zone ID
func (c *CloudflareProvider) getZoneID() (string, error) {
	reqURL := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones?name=%s", c.config.DomainName)

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+c.config.APIToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var cfResp CloudflareZoneResponse
	err = json.Unmarshal(body, &cfResp)
	if err != nil {
		return "", err
	}

	if !cfResp.Success {
		if len(cfResp.Errors) > 0 {
			return "", fmt.Errorf("Cloudflare API错误: %s", cfResp.Errors[0].Message)
		}
		return "", fmt.Errorf("Cloudflare API错误: 未知错误")
	}

	if len(cfResp.Result) == 0 {
		return "", fmt.Errorf("未找到Zone: %s", c.config.DomainName)
	}

	return cfResp.Result[0].ID, nil
}

// ============ 通用函数 ============

// getCurrentIP 获取当前公网IP
func getCurrentIP(recordType string, client *http.Client) (string, error) {
	var ipServices []string

	if recordType == "AAAA" {
		ipServices = []string{
			"https://api6.ipify.org",
			"https://v6.ident.me",
			"https://ipv6.icanhazip.com",
		}
	} else {
		ipServices = []string{
			"http://ip.3322.net/",
			"http://members.3322.org/dyndns/getip",
			"http://icanhazip.com/",
			"http://ipinfo.io/ip",
			"http://ip.42.pl/raw",
		}
	}

	for _, service := range ipServices {
		resp, err := client.Get(service)
		if err != nil {
			log.Printf("获取IP失败 (%s): %v", service, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == 200 {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Printf("读取响应失败 (%s): %v", service, err)
				continue
			}

			ip := strings.TrimSpace(string(body))
			if isValidIP(ip, recordType) {
				return ip, nil
			}
		}
	}

	return "", fmt.Errorf("无法获取当前IP地址")
}

// isValidIP IP验证
func isValidIP(ip, recordType string) bool {
	if recordType == "AAAA" {
		return strings.Contains(ip, ":") && !strings.HasPrefix(ip, ".")
	}

	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return false
	}

	for _, part := range parts {
		if len(part) == 0 || len(part) > 3 {
			return false
		}
		num, err := strconv.Atoi(part)
		if err != nil || num < 0 || num > 255 {
			return false
		}
	}
	return true
}

// ============ 配置加载 ============

// loadConfig 加载配置文件（支持新旧格式）
func loadConfig(configPath string) (Config, error) {
	var config Config

	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return config, err
	}

	// 首先尝试解析为新格式（带providers字段）
	err = json.Unmarshal(data, &config)
	if err != nil {
		// 如果失败，尝试解析为旧格式（向后兼容）
		var oldConfig struct {
			AccessKeyID     string `json:"access_key_id"`
			AccessKeySecret string `json:"access_key_secret"`
			DomainName      string `json:"domain_name"`
			SubDomain       string `json:"sub_domain"`
			RecordType      string `json:"record_type"`
			TTL             int    `json:"ttl"`
			CheckInterval   int    `json:"check_interval"`
		}

		if err2 := json.Unmarshal(data, &oldConfig); err2 != nil {
			return config, err
		}

		// 转换为新格式
		config.Providers = []ProviderConfig{
			{
				Type:            "aliyun",
				AccessKeyID:     oldConfig.AccessKeyID,
				AccessKeySecret: oldConfig.AccessKeySecret,
				DomainName:      oldConfig.DomainName,
				SubDomain:       oldConfig.SubDomain,
				RecordType:      oldConfig.RecordType,
				TTL:             oldConfig.TTL,
			},
		}
		config.CheckInterval = oldConfig.CheckInterval
	}

	// 设置默认值
	for i := range config.Providers {
		if config.Providers[i].RecordType == "" {
			config.Providers[i].RecordType = "A"
		}
		if config.Providers[i].TTL == 0 {
			config.Providers[i].TTL = 600
		}
	}

	if config.CheckInterval == 0 {
		config.CheckInterval = 300
	}

	return config, nil
}

// createProviders 根据配置创建提供商实例
func createProviders(config Config) ([]DNSProvider, error) {
	var providers []DNSProvider

	for _, pc := range config.Providers {
		switch pc.Type {
		case "aliyun":
			providers = append(providers, NewAliyunProvider(pc))
		case "cloudflare":
			providers = append(providers, NewCloudflareProvider(pc))
		default:
			return nil, fmt.Errorf("未知的提供商类型: %s", pc.Type)
		}
	}

	return providers, nil
}

// ============ 主程序 ============

// updateProvider 更新单个提供商的DNS记录
func updateProvider(provider DNSProvider) error {
	currentIP, err := provider.GetCurrentIP()
	if err != nil {
		return fmt.Errorf("[%s] 获取当前IP失败: %v", provider.GetName(), err)
	}

	log.Printf("[%s] 当前IP: %s", provider.GetName(), currentIP)

	// 获取现有DNS记录
	record, err := provider.GetDNSRecord()
	if err != nil {
		// 尝试添加新记录
		log.Printf("[%s] 未找到DNS记录，尝试添加新记录...", provider.GetName())
		err = provider.AddRecord(currentIP)
		if err != nil {
			return fmt.Errorf("[%s] 添加DNS记录失败: %v", provider.GetName(), err)
		}
		log.Printf("[%s] 成功添加DNS记录: %s", provider.GetName(), currentIP)
		return nil
	}

	// 检查是否需要更新
	recordIP := record.Value
	if recordIP == "" {
		recordIP = record.Content
	}

	if recordIP == currentIP {
		log.Printf("[%s] DNS记录已是最新: %s", provider.GetName(), currentIP)
		return nil
	}

	// 更新DNS记录
	log.Printf("[%s] 更新DNS记录: %s -> %s", provider.GetName(), recordIP, currentIP)
	err = provider.UpdateRecord(record.ID, currentIP)
	if err != nil {
		return fmt.Errorf("[%s] 更新DNS记录失败: %v", provider.GetName(), err)
	}

	log.Printf("[%s] 成功更新DNS记录: %s", provider.GetName(), currentIP)
	return nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("用法: ./aliyun-ddns <配置文件路径>")
		fmt.Println("示例: ./aliyun-ddns config.json")
		os.Exit(1)
	}

	configPath := os.Args[1]

	// 加载配置
	config, err := loadConfig(configPath)
	if err != nil {
		log.Fatalf("加载配置文件失败: %v", err)
	}

	// 创建提供商实例
	providers, err := createProviders(config)
	if err != nil {
		log.Fatalf("创建提供商失败: %v", err)
	}

	if len(providers) == 0 {
		log.Fatalf("没有配置任何DNS提供商")
	}

	log.Printf("DDNS客户端启动，%d个提供商", len(providers))

	// 主循环
	for {
		// 依次更新每个提供商
		for _, provider := range providers {
			err := updateProvider(provider)
			if err != nil {
				log.Printf("%v", err)
			}
		}

		// 等待下次检查
		time.Sleep(time.Duration(config.CheckInterval) * time.Second)
	}
}
