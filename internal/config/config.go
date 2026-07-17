package config

import "github.com/zeromicro/go-zero/zrpc"

type Config struct {
	zrpc.RpcServerConf
	SceneIdList    []int64
	MasterDBConfig struct {
		DataSource string
	}
}
