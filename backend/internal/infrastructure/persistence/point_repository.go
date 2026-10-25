package persistence

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jphacks/kz_2606/backend/internal/domain"
)

// usecase.PointRepository の Postgres 実装。
// 保存は pgx の CopyFrom で一括。1点ずつ INSERT しない。
func copyPoints(ctx context.Context, tx pgx.Tx, routeID int64, points []domain.Point) (int64, error) {
	rows, err := tx.CopyFrom(ctx, pgx.Identifier{"points"},
		[]string{"route_id", "latitude", "longitude", "accuracy", "recorded_at"},
		pgx.CopyFromSlice(len(points), func(i int) ([]any, error) {
			point := points[i]
			return []any{routeID, point.Lat, point.Lng, point.Acc, time.UnixMilli(point.TS).UTC()}, nil
		}),
	)
	if err != nil {
		return 0, err
	}
	if rows != int64(len(points)) {
		return rows, errors.New("point batch was not fully persisted")
	}
	return rows, nil
}
