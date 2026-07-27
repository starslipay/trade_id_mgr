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
	// 从环境变量中获取订单集群编号,默认值为16
	orderSetStr := os.Getenv("ORDER_SET")
	if orderSetStr == "" {
		orderSetStr = "16"
	}
	orderSet, err := strconv.Atoi(orderSetStr)
	if err != nil {
		panic(err)
	}
	return orderSet
}
