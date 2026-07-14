package logic

import (
	"context"
	"fmt"
	"time"

	"github.com/starslipay/trade_id_mgr/internal/svc"
	"github.com/starslipay/trade_id_mgr/trade_id_mgr_pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GenTradeIdLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGenTradeIdLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GenTradeIdLogic {
	return &GenTradeIdLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GenTradeIdLogic) GenTradeId(in *trade_id_mgr_pb.GenTradeIdReq) (*trade_id_mgr_pb.GenTradeIdRsp, error) {
	// 10位商户号 + 8位时间（yyyymmdd） + 3位预留扩展标记 + 2位订单集群编号 + 2位资金账户集群编号 + 10位业务序列号 + 3位付款方uid尾号
	tradeId := fmt.Sprintf("%s%s%s%s%02d%s%08d%03d",
		in.SpId,
		time.Now().Format("20060102"),
		"000",
		l.svcCtx.OrderSet,
		in.AccSet,
		l.svcCtx.MachineId,
		l.svcCtx.GetLocalSeqNo(),
		in.Uid%1000)
	// TODO 单号待优化,机器重启会出现重复的情况
	return &trade_id_mgr_pb.GenTradeIdRsp{
		TradeId: tradeId,
	}, nil
}
