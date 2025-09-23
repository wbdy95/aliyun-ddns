package main

import (
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

// Config 配置结构体
type Config struct {
	AccessKeyID     string `json:"access_key_id"`
	AccessKeySecret string `json:"access_key_secret"`
	DomainName      string `json:"domain_name"`
	SubDomain       string `json:"sub_domain"`
	RecordType      string `json:"record_type"`
	TTL             int    `json:"ttl"`
	CheckInterval   int    `json:"check_interval"`
}

// DNSRecord DNS记录结构体
type DNSRecord struct {
	RecordId string `json:"RecordId"`
	Value    string `json:"Value"`
}

// Response 阿里云API响应结构体
type Response struct {
	RequestId    string      `json:"RequestId"`
	DomainRecords struct {
		Record []DNSRecord `json:"Record"`
	} `json:"DomainRecords"`
	RecordId string `json:"RecordId"`
	Code     string `json:"Code"`
	Message  string `json:"Message"`
}

// DDNS 动态DNS客户端
type DDNS struct {
	config      Config
	currentIP   string
	lastIP      string
	recordID    string
	httpClient  *http.Client
}

// NewDDNS 创建新的DDNS实例
func NewDDNS(configPath string) (*DDNS, error) {
	config, err := loadConfig(configPath)
	if err != nil {
		return nil, err
	}

	return &DDNS{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// loadConfig 加载配置文件
func loadConfig(configPath string) (Config, error) {
	var config Config
	
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return config, err
	}

	err = json.Unmarshal(data, &config)
	if err != nil {
		return config, err
	}

	// 设置默认值
	if config.RecordType == "" {
		config.RecordType = "A"
	}
	if config.TTL == 0 {
		config.TTL = 600
	}
	if config.CheckInterval == 0 {
		config.CheckInterval = 300 // 5分钟
	}

	return config, nil
}

// getCurrentIP 获取当前公网IP
func (d *DDNS) getCurrentIP() (string, error) {
	// 使用多个IP检测服务，增加可靠性
	ipServices := []string{
		"http://ip.3322.net/",
		"http://members.3322.org/dyndns/getip",
		"http://icanhazip.com/",
		"http://ipinfo.io/ip",
		"http://ip.42.pl/raw",
	}

	for _, service := range ipServices {
		resp, err := d.httpClient.Get(service)
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
			if isValidIP(ip) {
				return ip, nil
			}
		}
	}

	return "", fmt.Errorf("无法获取当前IP地址")
}

// isValidIP 简单的IP验证
func isValidIP(ip string) bool {
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

// signRequest 生成阿里云API签名
func (d *DDNS) signRequest(params map[string]string) string {
	// 添加公共参数
	params["Format"] = "JSON"
	params["Version"] = "2015-01-09"
	params["AccessKeyId"] = d.config.AccessKeyID
	params["SignatureMethod"] = "HMAC-SHA1"
	params["Timestamp"] = time.Now().UTC().Format("2006-01-02T15:04:05Z")
	params["SignatureVersion"] = "1.0"
	params["SignatureNonce"] = strconv.FormatInt(time.Now().UnixNano(), 10)

	// 对参数进行排序
	var keys []string
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 构建查询字符串
	var queryParts []string
	for _, k := range keys {
		queryParts = append(queryParts, url.QueryEscape(k)+"="+url.QueryEscape(params[k]))
	}
	queryString := strings.Join(queryParts, "&")

	// 构建待签名字符串
	stringToSign := "GET&" + url.QueryEscape("/") + "&" + url.QueryEscape(queryString)

	// 计算签名
	h := hmac.New(sha1.New, []byte(d.config.AccessKeySecret+"&"))
	h.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	return signature
}

// makeAPIRequest 发送API请求
func (d *DDNS) makeAPIRequest(params map[string]string) (*Response, error) {
	signature := d.signRequest(params)
	params["Signature"] = signature

	// 构建URL
	baseURL := "https://alidns.aliyuncs.com/"
	u, _ := url.Parse(baseURL)
	q := u.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	// 发送请求
	resp, err := d.httpClient.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response Response
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	if response.Code != "" && response.Code != "200" {
		return nil, fmt.Errorf("API错误: %s - %s", response.Code, response.Message)
	}

	return &response, nil
}

// getDNSRecord 获取DNS记录
func (d *DDNS) getDNSRecord() (*DNSRecord, error) {
	params := map[string]string{
		"Action":     "DescribeDomainRecords",
		"DomainName": d.config.DomainName,
		"RRKeyWord":  d.config.SubDomain,
		"Type":       d.config.RecordType,
	}

	resp, err := d.makeAPIRequest(params)
	if err != nil {
		return nil, err
	}

	if len(resp.DomainRecords.Record) == 0 {
		return nil, fmt.Errorf("未找到DNS记录: %s.%s", d.config.SubDomain, d.config.DomainName)
	}

	return &resp.DomainRecords.Record[0], nil
}

// updateDNSRecord 更新DNS记录
func (d *DDNS) updateDNSRecord(recordID, newIP string) error {
	params := map[string]string{
		"Action":   "UpdateDomainRecord",
		"RecordId": recordID,
		"RR":       d.config.SubDomain,
		"Type":     d.config.RecordType,
		"Value":    newIP,
		"TTL":      strconv.Itoa(d.config.TTL),
	}

	_, err := d.makeAPIRequest(params)
	return err
}

// addDNSRecord 添加DNS记录
func (d *DDNS) addDNSRecord(ip string) error {
	params := map[string]string{
		"Action":     "AddDomainRecord",
		"DomainName": d.config.DomainName,
		"RR":         d.config.SubDomain,
		"Type":       d.config.RecordType,
		"Value":      ip,
		"TTL":        strconv.Itoa(d.config.TTL),
	}

	_, err := d.makeAPIRequest(params)
	return err
}

// Run 运行DDNS客户端
func (d *DDNS) Run() error {
	log.Printf("DDNS客户端启动，域名: %s.%s", d.config.SubDomain, d.config.DomainName)
	
	for {
		// 获取当前IP
		currentIP, err := d.getCurrentIP()
		if err != nil {
			log.Printf("获取当前IP失败: %v", err)
			time.Sleep(time.Duration(d.config.CheckInterval) * time.Second)
			continue
		}

		d.currentIP = currentIP
		log.Printf("当前IP: %s", currentIP)

		// 如果IP没有变化，跳过更新
		if currentIP == d.lastIP {
			log.Printf("IP未发生变化，跳过更新")
			time.Sleep(time.Duration(d.config.CheckInterval) * time.Second)
			continue
		}

		// 获取现有DNS记录
		record, err := d.getDNSRecord()
		if err != nil {
			log.Printf("获取DNS记录失败: %v", err)
			// 尝试添加新记录
			log.Printf("尝试添加新的DNS记录...")
			err = d.addDNSRecord(currentIP)
			if err != nil {
				log.Printf("添加DNS记录失败: %v", err)
			} else {
				log.Printf("成功添加DNS记录: %s.%s -> %s", d.config.SubDomain, d.config.DomainName, currentIP)
				d.lastIP = currentIP
			}
			time.Sleep(time.Duration(d.config.CheckInterval) * time.Second)
			continue
		}

		// 如果DNS记录的IP与当前IP相同，更新lastIP
		if record.Value == currentIP {
			log.Printf("DNS记录已是最新: %s", currentIP)
			d.lastIP = currentIP
			time.Sleep(time.Duration(d.config.CheckInterval) * time.Second)
			continue
		}

		// 更新DNS记录
		log.Printf("更新DNS记录: %s -> %s", record.Value, currentIP)
		err = d.updateDNSRecord(record.RecordId, currentIP)
		if err != nil {
			log.Printf("更新DNS记录失败: %v", err)
		} else {
			log.Printf("成功更新DNS记录: %s.%s -> %s", d.config.SubDomain, d.config.DomainName, currentIP)
			d.lastIP = currentIP
		}

		time.Sleep(time.Duration(d.config.CheckInterval) * time.Second)
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("用法: ./aliyun-ddns <配置文件路径>")
		fmt.Println("示例: ./aliyun-ddns config.json")
		os.Exit(1)
	}

	configPath := os.Args[1]
	ddns, err := NewDDNS(configPath)
	if err != nil {
		log.Fatalf("初始化DDNS客户端失败: %v", err)
	}

	err = ddns.Run()
	if err != nil {
		log.Fatalf("DDNS客户端运行失败: %v", err)
	}
}