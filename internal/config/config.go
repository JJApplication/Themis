package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config 应用配置结构
type Config struct {
	// 服务器配置
	Server ServerConfig `json:"server"`
	// 端口配置
	Port PortConfig `json:"port"`
	// 存储配置
	Storage StorageConfig `json:"storage"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	ListenType string `json:"listen_type"` // "unix" 或 "tcp"
	UnixSocket string `json:"unix_socket"` // Unix socket路径
	Host       string `json:"host"`        // TCP监听主机
	Port       int    `json:"port"`       // TCP监听端口
}

// PortConfig 端口配置
type PortConfig struct {
	MinPort int `json:"min_port"` // 最小端口号
	MaxPort int `json:"max_port"` // 最大端口号
}

// StorageConfig 存储配置
type StorageConfig struct {
	DataFile     string `json:"data_file"`     // 数据文件路径
	SyncInterval int    `json:"sync_interval"` // 同步间隔（秒）
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			ListenType: "unix",
			UnixSocket: "/var/run/Themis.sock",
			Host:       "localhost",
			Port:       9090,
		},
		Port: PortConfig{
			MinPort: 10000,
			MaxPort: 20000,
		},
		Storage: StorageConfig{
			DataFile:     "./data/ports.json",
			SyncInterval: 60, // 60秒
		},
	}
}

// LoadConfig 从文件加载配置
func LoadConfig(configPath string) (*Config, error) {
	// 如果配置文件不存在，返回默认配置
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return DefaultConfig(), nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %v", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}

	// 验证配置
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %v", err)
	}

	return &config, nil
}

// SaveConfig 保存配置到文件
func SaveConfig(config *Config, configPath string) error {
	// 确保目录存在
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %v", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %v", err)
	}

	return nil
}

// Validate 验证配置的有效性
func (c *Config) Validate() error {
	// 验证服务器配置
	if c.Server.ListenType != "unix" && c.Server.ListenType != "tcp" {
		return fmt.Errorf("无效的监听类型: %s，必须是 'unix' 或 'tcp'", c.Server.ListenType)
	}

	if c.Server.ListenType == "unix" && c.Server.UnixSocket == "" {
		return fmt.Errorf("Unix socket路径不能为空")
	}

	if c.Server.ListenType == "tcp" {
		if c.Server.Host == "" {
			return fmt.Errorf("TCP监听主机不能为空")
		}
		if c.Server.Port <= 0 || c.Server.Port > 65535 {
			return fmt.Errorf("无效的TCP端口: %d", c.Server.Port)
		}
	}

	// 验证端口配置
	if c.Port.MinPort <= 0 || c.Port.MaxPort <= 0 {
		return fmt.Errorf("端口范围必须大于0")
	}
	if c.Port.MinPort > c.Port.MaxPort {
		return fmt.Errorf("最小端口不能大于最大端口")
	}

	// 验证存储配置
	if c.Storage.DataFile == "" {
		return fmt.Errorf("数据文件路径不能为空")
	}
	if c.Storage.SyncInterval <= 0 {
		return fmt.Errorf("同步间隔必须大于0")
	}

	return nil
}

// GetListenAddress 获取监听地址
func (c *Config) GetListenAddress() string {
	if c.Server.ListenType == "unix" {
		return c.Server.UnixSocket
	}
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}