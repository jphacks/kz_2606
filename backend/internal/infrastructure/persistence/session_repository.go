package persistence

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jphacks/kz_2606/backend/internal/domain"
)

type SessionRepository struct {
	pool *pgxpool.Pool
}

func NewSessionRepository(pool *pgxpool.Pool) *SessionRepository {
	return &SessionRepository{pool: pool}
}

func (r *SessionRepository) StartRoute(ctx context.Context, code string, memberID int64) (domain.Room, domain.Member, domain.Route, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Room{}, domain.Member{}, domain.Route{}, err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	room, err := lockSessionRoom(ctx, tx, code)
	if err != nil {
		return domain.Room{}, domain.Member{}, domain.Route{}, err
	}
	if room.ClosedAt != nil {
		return domain.Room{}, domain.Member{}, domain.Route{}, domain.ErrRoomClosed
	}
	member, err := lockSessionMember(ctx, tx, room.ID, memberID)
	if err != nil {
		return domain.Room{}, domain.Member{}, domain.Route{}, err
	}
	if member.BannedAt != nil {
		return domain.Room{}, domain.Member{}, domain.Route{}, domain.ErrMemberBanned
	}
	var active bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM routes WHERE room_id = $1 AND ended_at IS NULL
		)
	`, room.ID).Scan(&active); err != nil {
		return domain.Room{}, domain.Member{}, domain.Route{}, err
	}
	if active {
		return domain.Room{}, domain.Member{}, domain.Route{}, domain.ErrRouteInUse
	}
	var route domain.Route
	route.MemberID = memberID
	err = tx.QueryRow(ctx, `
		INSERT INTO routes (room_id, member_id) VALUES ($1, $2)
		RETURNING id, started_at
	`, room.ID, memberID).Scan(&route.ID, &route.StartedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.Room{}, domain.Member{}, domain.Route{}, domain.ErrRouteInUse
		}
		return domain.Room{}, domain.Member{}, domain.Route{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Room{}, domain.Member{}, domain.Route{}, err
	}
	return room, member, route, nil
}

func (r *SessionRepository) AppendPoints(ctx context.Context, code string, memberID int64, points []domain.Point) (int64, domain.Member, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, domain.Member{}, err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	room, err := lockSessionRoom(ctx, tx, code)
	if err != nil {
		return 0, domain.Member{}, err
	}
	if room.ClosedAt != nil {
		return 0, domain.Member{}, domain.ErrRoomClosed
	}
	member, err := lockSessionMember(ctx, tx, room.ID, memberID)
	if err != nil {
		return 0, domain.Member{}, err
	}
	if member.BannedAt != nil {
		return 0, domain.Member{}, domain.ErrMemberBanned
	}
	var routeID int64
	err = tx.QueryRow(ctx, `
		SELECT id FROM routes
		WHERE room_id = $1 AND member_id = $2 AND ended_at IS NULL
	`, room.ID, memberID).Scan(&routeID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, domain.Member{}, domain.ErrNoActiveRoute
	}
	if err != nil {
		return 0, domain.Member{}, err
	}
	if _, err := copyPoints(ctx, tx, routeID, points); err != nil {
		return 0, domain.Member{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, domain.Member{}, err
	}
	return routeID, member, nil
}

func (r *SessionRepository) StopRoute(ctx context.Context, code string, memberID int64) (domain.Room, domain.Member, int64, time.Time, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Room{}, domain.Member{}, 0, time.Time{}, err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	room, err := lockSessionRoom(ctx, tx, code)
	if err != nil {
		return domain.Room{}, domain.Member{}, 0, time.Time{}, err
	}
	member, err := lockSessionMember(ctx, tx, room.ID, memberID)
	if err != nil {
		return domain.Room{}, domain.Member{}, 0, time.Time{}, err
	}
	if member.BannedAt != nil {
		return domain.Room{}, domain.Member{}, 0, time.Time{}, domain.ErrMemberBanned
	}
	var routeID int64
	var endedAt time.Time
	err = tx.QueryRow(ctx, `
		UPDATE routes SET ended_at = now()
		WHERE id = (
			SELECT id FROM routes
			WHERE room_id = $1 AND member_id = $2 AND ended_at IS NULL
			FOR UPDATE
		)
		RETURNING id, ended_at
	`, room.ID, memberID).Scan(&routeID, &endedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Room{}, domain.Member{}, 0, time.Time{}, domain.ErrNoActiveRoute
	}
	if err != nil {
		return domain.Room{}, domain.Member{}, 0, time.Time{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Room{}, domain.Member{}, 0, time.Time{}, err
	}
	return room, member, routeID, endedAt, nil
}

func (r *SessionRepository) UpdateMember(ctx context.Context, code string, memberID int64, displayName, color *string) (domain.Member, bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Member{}, false, err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	room, err := lockSessionRoom(ctx, tx, code)
	if err != nil {
		return domain.Member{}, false, err
	}
	member, err := lockSessionMember(ctx, tx, room.ID, memberID)
	if err != nil {
		return domain.Member{}, false, err
	}
	if member.BannedAt != nil {
		return domain.Member{}, false, domain.ErrMemberBanned
	}
	if displayName != nil || color != nil {
		err = tx.QueryRow(ctx, `
			UPDATE members
			SET display_name = COALESCE($3, display_name),
			    color = COALESCE($4, color)
			WHERE id = $1 AND room_id = $2
			RETURNING id, display_name, color
		`, memberID, room.ID, displayName, color).Scan(&member.ID, &member.DisplayName, &member.Color)
		if err != nil {
			return domain.Member{}, false, err
		}
	}
	var recording bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM routes WHERE room_id = $1 AND member_id = $2 AND ended_at IS NULL
		)
	`, room.ID, memberID).Scan(&recording); err != nil {
		return domain.Member{}, false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Member{}, false, err
	}
	return member, recording, nil
}

func (r *SessionRepository) Leave(ctx context.Context, code string, memberID int64) (domain.Member, int64, time.Time, bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Member{}, 0, time.Time{}, false, err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	room, err := lockSessionRoom(ctx, tx, code)
	if err != nil {
		return domain.Member{}, 0, time.Time{}, false, err
	}
	member, err := lockSessionMember(ctx, tx, room.ID, memberID)
	if err != nil {
		return domain.Member{}, 0, time.Time{}, false, err
	}
	var routeID int64
	var endedAt time.Time
	err = tx.QueryRow(ctx, `
		UPDATE routes SET ended_at = now()
		WHERE id = (
			SELECT id FROM routes
			WHERE room_id = $1 AND member_id = $2 AND ended_at IS NULL
			FOR UPDATE
		)
		RETURNING id, ended_at
	`, room.ID, memberID).Scan(&routeID, &endedAt)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return domain.Member{}, 0, time.Time{}, false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Member{}, 0, time.Time{}, false, err
	}
	return member, routeID, endedAt, err == nil, nil
}

func lockSessionRoom(ctx context.Context, tx pgx.Tx, code string) (domain.Room, error) {
	var room domain.Room
	var closedAt pgtype.Timestamptz
	err := tx.QueryRow(ctx, `
		SELECT id, code, title, owner_user_id, is_private, closed_at
		FROM rooms WHERE code = $1 FOR UPDATE
	`, code).Scan(&room.ID, &room.Code, &room.Title, &room.OwnerUserID, &room.IsPrivate, &closedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Room{}, domain.ErrRoomNotFound
	}
	if err != nil {
		return domain.Room{}, err
	}
	if closedAt.Valid {
		room.ClosedAt = &closedAt.Time
	}
	return room, nil
}

func lockSessionMember(ctx context.Context, tx pgx.Tx, roomID, memberID int64) (domain.Member, error) {
	var member domain.Member
	var bannedAt pgtype.Timestamptz
	err := tx.QueryRow(ctx, `
		SELECT id, display_name, color, banned_at
		FROM members WHERE id = $1 AND room_id = $2 FOR UPDATE
	`, memberID, roomID).Scan(&member.ID, &member.DisplayName, &member.Color, &bannedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Member{}, domain.ErrMemberNotFound
	}
	if err != nil {
		return domain.Member{}, err
	}
	if bannedAt.Valid {
		member.BannedAt = &bannedAt.Time
	}
	return member, nil
}
