package provider

import (
	"fmt"
	"CloudFileSync/config"
)

// NewProvider 根据配置创建云盘提供商
func NewProvider(providerCfg config.ProviderConfig) (Provider, error) {
	switch providerCfg.Type {
	case "aliyun":
		return NewAliYunProvider(providerCfg.Tokens)
	case "baidu":
		return NewBaiduProvider(providerCfg.Tokens)
	default:
		return nil, fmt.Errorf("不支持的云盘类型: %s", providerCfg.Type)
	}
}
