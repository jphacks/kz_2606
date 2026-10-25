package persistence

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jphacks/kz_2606/backend/internal/domain"
)

// Activities は、ユーザー本人が記録した route を新しい順に返す。
// 距離は domain.Route.Stats で仕様どおり算出できるよう、点列も時刻順に読む。
func (r *RoomRepository) Activities(ctx context.Context, userID int64) ([]domain.Activity, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT routes.id, routes.member_id, routes.started_at, routes.ended_at,
		       rooms.code, rooms.title, rooms.is_private,
		       (SELECT COUNT(*)
		          FROM members AS room_members
		         WHERE room_members.room_id = rooms.id
		           AND room_members.banned_at IS NULL),
		       points.latitude, points.longitude,
		       (EXTRACT(EPOCH FROM points.recorded_at) * 1000)::BIGINT,
		       COALESCE(points.accuracy, 0)::REAL
		FROM routes
		JOIN members ON members.id = routes.member_id
		JOIN rooms ON rooms.id = routes.room_id
		LEFT JOIN points ON points.route_id = routes.id
		WHERE members.user_id = $1
		ORDER BY routes.started_at DESC, routes.id DESC, points.recorded_at, points.id
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	activities := make([]domain.Activity, 0)
	activityIndexes := make(map[int64]int)
	for rows.Next() {
		var routeID, memberID int64
		var startedAt pgtype.Timestamptz
		var endedAt pgtype.Timestamptz
		var roomCode, roomTitle string
		var isPrivate bool
		var memberCount int
		var latitude, longitude pgtype.Float8
		var timestamp pgtype.Int8
		var accuracy pgtype.Float4
		if err := rows.Scan(
			&routeID, &memberID, &startedAt, &endedAt,
			&roomCode, &roomTitle, &isPrivate, &memberCount,
			&latitude, &longitude, &timestamp, &accuracy,
		); err != nil {
			return nil, err
		}

		index, exists := activityIndexes[routeID]
		if !exists {
			activity := domain.Activity{
				RoomCode:    roomCode,
				RoomTitle:   roomTitle,
				IsPrivate:   isPrivate,
				MemberCount: memberCount,
				Route: domain.Route{
					ID:        routeID,
					MemberID:  memberID,
					StartedAt: startedAt.Time,
					Points:    make([]domain.Point, 0),
				},
			}
			if endedAt.Valid {
				activity.Route.EndedAt = timePointer(endedAt.Time)
			}
			activities = append(activities, activity)
			index = len(activities) - 1
			activityIndexes[routeID] = index
		}

		if latitude.Valid {
			activities[index].Route.Points = append(activities[index].Route.Points, domain.Point{
				Lat: latitude.Float64,
				Lng: longitude.Float64,
				TS:  timestamp.Int64,
				Acc: accuracy.Float32,
			})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return activities, nil
}
