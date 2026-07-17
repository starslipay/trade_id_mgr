package mysql

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ TIdSegmentModel = (*customTIdSegmentModel)(nil)

type (
	TIdSegmentModel interface {
		tIdSegmentModel
		WithSession(session sqlx.Session) TIdSegmentModel
		FindOneForUpdate(ctx context.Context, sceneId int64) (*TIdSegment, error)
	}

	customTIdSegmentModel struct {
		*defaultTIdSegmentModel
	}
)

// NewTIdSegmentModel returns a model for the database table.
func NewTIdSegmentModel(conn sqlx.SqlConn) TIdSegmentModel {
	return &customTIdSegmentModel{
		defaultTIdSegmentModel: newTIdSegmentModel(conn),
	}
}

func (m *customTIdSegmentModel) WithSession(session sqlx.Session) TIdSegmentModel {
	return NewTIdSegmentModel(sqlx.NewSqlConnFromSession(session))
}

func (m *customTIdSegmentModel) FindOneForUpdate(ctx context.Context, sceneId int64) (*TIdSegment, error) {
	query := fmt.Sprintf("select %s from %s where `scene_id` = ? limit 1 for update", tIdSegmentRows, m.table)
	var resp TIdSegment
	err := m.conn.QueryRowCtx(ctx, &resp, query, sceneId)
	switch err {
	case nil:
		return &resp, nil
	case sqlx.ErrNotFound:
		return nil, ErrNotFound
	default:
		return nil, err
	}
}
