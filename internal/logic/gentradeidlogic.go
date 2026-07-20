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
	// 我的单号规则是 商户号（10位数字） + 时间（yyyyyymmdd）+ set 编号（2 位） + acc_set 编号（2 位） + bill_no（int64 数字，19 位向左补 0） + uid 后三位
	bill_no, err := l.svcCtx.IDGenerator.GetID(l.ctx, in.SceneId)
	if err != nil {
		return nil, err
	}
	tradeId := fmt.Sprintf("%s%s%03d%s%02d%019d%03d",
		in.SpId,
		time.Now().Format("20060102"),
		in.SceneId,
		l.svcCtx.OrderSet,
		in.AccSet,
		bill_no,
		in.Uid%1000)
	return &trade_id_mgr_pb.GenTradeIdRsp{
		TradeId: tradeId,
	}, nil
}
