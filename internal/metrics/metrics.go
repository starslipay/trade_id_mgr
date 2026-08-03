package metrics

import (
	"strconv"

	"github.com/zeromicro/go-zero/core/metric"
)

// serviceName 作为 service 维度标签，标识指标来自哪个服务
const serviceName = "trade_id_mgr"

// moduleUnknownCode 兜底错误码: RPC 层无法解析出业务错误码时使用
// 取值为本模块 ModuleId*10000 (trade_id_mgr ModuleId=80000)
const moduleUnknownCode = int64(800000000)

const (
	resultSuccess = "success"
	resultError   = "error"
)

// codeCounter 自定义请求计数器：按 服务/接口/错误码/结果 维度累加
// 指标全名为 pay_biz_code_total
var codeCounter = metric.NewCounterVec(&metric.CounterVecOpts{
	Namespace: "pay",
	Subsystem: "biz",
	Name:      "code_total",
	Help:      "request counter by service/method/code/result.",
	Labels:    []string{"service", "method", "code", "result"},
})

// ReportCode 统一打点入口
// method: RPC 走 FullMethod
// code:   0 表示成功, 非 0 表示失败(错误码)
func ReportCode(method string, code int64) {
	result := resultSuccess
	if code != 0 {
		result = resultError
	}
	codeCounter.Inc(serviceName, method, strconv.FormatInt(code, 10), result)
}
