package config

import "github.com/zeromicro/go-zero/zrpc"

type Config struct {
	zrpc.RpcServerConf
	SceneIdList    []int64
	MasterDBConfig struct {
		DataSource string
	}

	// AccessLog 请求/响应日志脱敏配置
	AccessLog AccessLogConf
}

// AccessLogConf 访问日志配置
type AccessLogConf struct {
	Enable          bool     `json:",default=true"`
	SensitiveFields []string `json:",optional"`
}
