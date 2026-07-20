package logic

import (
	"context"

	"github.com/starslipay/trade_id_mgr/internal/svc"
	"github.com/starslipay/trade_id_mgr/trade_id_mgr_pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GenUidLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGenUidLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GenUidLogic {
	return &GenUidLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GenUidLogic) GenUid(in *trade_id_mgr_pb.GenUidReq) (*trade_id_mgr_pb.GenUidRsp, error) {
	var sceneId int64 = 0 // UID
	uid, err := l.svcCtx.IDGenerator.GetID(l.ctx, sceneId)
	if err != nil {
		return nil, err
	}
	return &trade_id_mgr_pb.GenUidRsp{
		Uid: uid,
	}, nil
}
