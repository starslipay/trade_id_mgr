package svc

import (
	"os"
	"strconv"

	"github.com/starslipay/trade_id_mgr/internal/config"
	"github.com/starslipay/trade_id_mgr/internal/id_generator"
	"github.com/starslipay/trade_id_mgr/model/mysql"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ServiceContext struct {
	Config      config.Config
	OrderSet    int
	IDGenerator *id_generator.IDGenerator
}

func NewServiceContext(c config.Config) *ServiceContext {
	conn := sqlx.NewMysql(c.MasterDBConfig.DataSource)
	ig := id_generator.MustNewIDGenerator(mysql.NewTIdSegmentModel(conn), conn, c.SceneIdList)
	return &ServiceContext{
		Config:      c,
		OrderSet:    GetOrderSet(),
		IDGenerator: ig,
	}
}

func GetOrderSet() int {
	// 从环境变量中获取订单集群编号
	// 如果环境变量中没有配置，默认使用配置文件中的值
	orderSet, err := strconv.Atoi(os.Getenv("ORDER_SET"))
	if err != nil {
		panic(err)
	}
	if orderSet == 0 {
		orderSet = 0
	}
	return orderSet
}
