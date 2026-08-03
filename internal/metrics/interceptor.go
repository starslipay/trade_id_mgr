package metrics

import (
	"context"

	"github.com/starslipay/paycomm/xerror"
	"google.golang.org/grpc"
)

// UnaryMetricInterceptor gRPC 服务端一元拦截器: 统计每个 RPC 方法的错误码
// 成功(err==nil) 打 code=0; 失败时用 xerror.ParseBizError 解出业务错误码,
// 解析不出则记为本模块兜底系统错误码。
func UnaryMetricInterceptor(ctx context.Context, req any,
	info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	resp, err := handler(ctx, req)
	if err == nil {
		ReportCode(info.FullMethod, 0)
		return resp, err
	}
	if bizErr, ok := xerror.ParseBizError(err); ok {
		ReportCode(info.FullMethod, bizErr.Code)
	} else {
		ReportCode(info.FullMethod, moduleUnknownCode)
	}
	return resp, err
}
