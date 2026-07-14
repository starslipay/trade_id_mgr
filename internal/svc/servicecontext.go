package svc

import (
	"os"
	"sync"

	"github.com/starslipay/trade_id_mgr/internal/config"
)

type ServiceContext struct {
	Config     config.Config
	OrderSet   string
	MachineId  string
	localSeqNo int32
	mu         sync.Mutex
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:     c,
		OrderSet:   GetOrderSet(),
		MachineId:  GetMachineId(),
		localSeqNo: 0,
	}
}

func GetOrderSet() string {
	// 从环境变量中获取订单集群编号
	// 如果环境变量中没有配置，默认使用配置文件中的值
	orderSet := os.Getenv("ORDER_SET")
	if orderSet == "" {
		orderSet = "00"
	}
	return orderSet
}

func GetMachineId() string {
	// 从环境变量中获取机器编号
	// 如果环境变量中没有配置，默认使用配置文件中的值
	machineId := os.Getenv("MACHINE_ID")
	if machineId == "" {
		machineId = "00"
	}
	return machineId
}

// 8位自增序号，用于生成交易id
func (g *ServiceContext) GetLocalSeqNo() int32 {
	g.mu.Lock()
	defer g.mu.Unlock()
	ret := g.localSeqNo
	g.localSeqNo += 1
	if g.localSeqNo > 99999999 {
		g.localSeqNo = 0
	}
	return ret
}
