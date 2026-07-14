package logic

import (
	"context"
	"strconv"
	"time"

	"trade_id_mgr/internal/svc"
	"trade_id_mgr/trade_id_mgr_pb"

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
	tradeId := in.SpId + // 商户id
		time.Now().Format("20060102") + // 时间（yyyymmdd）
		"000" + //  3位预留扩展标记： 比如1位社交/商业 + 2位业务标记
		l.svcCtx.OrderSet + // 订单集群编号
		strconv.FormatInt(int64(in.AccSet), 10) + // 资金账户集群编号
		l.svcCtx.MachineId + strconv.FormatInt(int64(l.svcCtx.GetLocalSeqNo()), 10) + // 业务序列号（机器号+机器自增序号）
		strconv.FormatInt(int64(in.Uid)%1000, 10) // 付款方uid尾号（取后3位）

	return &trade_id_mgr_pb.GenTradeIdRsp{
		TradeId: tradeId,
	}, nil
}
