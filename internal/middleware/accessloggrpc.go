package middleware

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const maskPlaceholder = "***"

type SensitiveFields map[string]bool

func NewSensitiveFields(fields []string) SensitiveFields {
	m := make(SensitiveFields, len(fields))
	for _, f := range fields {
		m[strings.ToLower(f)] = true
	}
	return m
}

func (sf SensitiveFields) IsSensitive(field string) bool {
	return sf[strings.ToLower(field)]
}

// AccessLogUnaryInterceptor gRPC 一元拦截器：记录请求和响应参数，并按字段名脱敏
func AccessLogUnaryInterceptor(sf SensitiveFields) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		// 调用前打印入参(脱敏)
		reqJSON := protoToJSON(req)
		maskedReq := maskJSON(reqJSON, sf)
		logx.WithContext(ctx).Infof("[ACCESS-IN] %s\n=> req: %s", info.FullMethod, maskedReq)

		// 执行下游 handler
		resp, err := handler(ctx, req)

		// 调用后打印出参(脱敏)
		if err != nil {
			logx.WithContext(ctx).Infof("[ACCESS-OUT] %s | duration=%s | error: %v",
				info.FullMethod, time.Since(start).String(), err)
		} else {
			respJSON := protoToJSON(resp)
			maskedResp := maskJSON(respJSON, sf)
			logx.WithContext(ctx).Infof("[ACCESS-OUT] %s | duration=%s\n<= resp: %s",
				info.FullMethod, time.Since(start).String(), maskedResp)
		}

		return resp, err
	}
}

func protoToJSON(m interface{}) []byte {
	if m == nil {
		return nil
	}
	if pm, ok := m.(proto.Message); ok {
		data, err := protojson.Marshal(pm)
		if err != nil {
			return nil
		}
		return data
	}
	data, err := json.Marshal(m)
	if err != nil {
		return nil
	}
	return data
}

func maskJSON(data []byte, sf SensitiveFields) string {
	if len(data) == 0 {
		return ""
	}
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return string(data)
	}
	masked := maskValue(v, sf)
	out, err := json.Marshal(masked)
	if err != nil {
		return string(data)
	}
	return string(out)
}

func maskValue(v interface{}, sf SensitiveFields) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{}, len(val))
		for k, vv := range val {
			if sf.IsSensitive(k) {
				result[k] = maskPlaceholder
			} else {
				result[k] = maskValue(vv, sf)
			}
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(val))
		for i, item := range val {
			result[i] = maskValue(item, sf)
		}
		return result
	default:
		return v
	}
}
