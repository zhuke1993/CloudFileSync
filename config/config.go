package config

import (
	"encoding/json"
	"os"
	"time"
)

// Config 主配置结构
type Config struct {
	WatchDir    string            `json:"watch_dir"`    // 监听的目录
	DelayTime   int               `json:"delay_time"`   // 延迟上传时间（秒）
	Providers   []ProviderConfig  `json:"providers"`    // 云盘配置列表
}

// ProviderConfig 云盘提供商配置
type ProviderConfig struct {
	Type   string            `json:"type"`   // 类型: "aliyun" 或 "baidu"
	Name   string            `json:"name"`   // 配置名称
	Enable bool              `json:"enable"` // 是否启用
	Tokens map[string]string `json:"tokens"` // 认证令牌
	Target string            `json:"target"` // 目标目录
}

// LoadConfig 从文件加载配置
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// GetDelayDuration 获取延迟时间
func (c *Config) GetDelayDuration() time.Duration {
	return time.Duration(c.DelayTime) * time.Second
}
