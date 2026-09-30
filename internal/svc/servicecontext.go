package svc

import (
	"database/sql"
	"os"
	"strconv"
	"time"

	"github.com/starslipay/trade_id_mgr/internal/config"
	"github.com/starslipay/trade_id_mgr/internal/id_generator"
	"github.com/starslipay/trade_id_mgr/model/mysql"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ServiceContext struct {
	Config      config.Config
	OrderSet    int
	IDGenerator *id_generator.IDGenerator
}

// newDBConn 自建 *sql.DB 以支持自定义连接池参数
func newDBConn(dataSource string, maxOpen, maxIdle, lifetimeSec int) sqlx.SqlConn {
	db, err := sql.Open("mysql", dataSource)
	if err != nil {
		logx.Must(err)
	}
	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)
	db.SetConnMaxLifetime(time.Duration(lifetimeSec) * time.Second)
	if err = db.Ping(); err != nil {
		logx.Must(err)
	}
	// 等效关闭 sqlx 层熔断：所有错误都视为"可接受"，熔断器永不累计失败
	// 避免号段预取被 circuit breaker is open 快速失败，放大缓存耗尽风险
	return sqlx.NewSqlConnFromDB(db, sqlx.WithAcceptable(func(err error) bool { return true }))
}

func NewServiceContext(c config.Config) *ServiceContext {
	conn := newDBConn(c.MasterDBConfig.DataSource,
		c.MasterDBConfig.MaxOpenConns, c.MasterDBConfig.MaxIdleConns, c.MasterDBConfig.ConnMaxLifetimeSec)
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
