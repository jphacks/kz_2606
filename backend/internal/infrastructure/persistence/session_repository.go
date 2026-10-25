package persistence

import (
	"context"
	"errors"

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
	if memberID != room.OwnerMemberID {
		var ownerRecording bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM routes WHERE room_id = $1 AND member_id = $2 AND ended_at IS NULL
			)
		`, room.ID, room.OwnerMemberID).Scan(&ownerRecording); err != nil {
			return domain.Room{}, domain.Member{}, domain.Route{}, err
		}
		if !ownerRecording {
			return domain.Room{}, domain.Member{}, domain.Route{}, domain.ErrOwnerNotRecording
		}
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

func (r *SessionRepository) StopRoute(ctx context.Context, code string, memberID int64) (domain.Room, domain.Member, []domain.RouteEnd, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Room{}, domain.Member{}, nil, err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	room, err := lockSessionRoom(ctx, tx, code)
	if err != nil {
		return domain.Room{}, domain.Member{}, nil, err
	}
	member, err := lockSessionMember(ctx, tx, room.ID, memberID)
	if err != nil {
		return domain.Room{}, domain.Member{}, nil, err
	}
	if member.BannedAt != nil {
		return domain.Room{}, domain.Member{}, nil, domain.ErrMemberBanned
	}

	routeEnds := make([]domain.RouteEnd, 0, 1)
	if memberID == room.OwnerMemberID {
		rows, queryErr := tx.Query(ctx, `
			UPDATE routes SET ended_at = now()
			WHERE room_id = $1 AND ended_at IS NULL
			RETURNING id, member_id, ended_at
		`, room.ID)
		if queryErr != nil {
			return domain.Room{}, domain.Member{}, nil, queryErr
		}
		for rows.Next() {
			var routeEnd domain.RouteEnd
			if scanErr := rows.Scan(&routeEnd.ID, &routeEnd.MemberID, &routeEnd.EndedAt); scanErr != nil {
				rows.Close()
				return domain.Room{}, domain.Member{}, nil, scanErr
			}
			routeEnds = append(routeEnds, routeEnd)
		}
		if rowsErr := rows.Err(); rowsErr != nil {
			rows.Close()
			return domain.Room{}, domain.Member{}, nil, rowsErr
		}
		rows.Close()
	} else {
		var routeEnd domain.RouteEnd
		err = tx.QueryRow(ctx, `
			UPDATE routes SET ended_at = now()
			WHERE id = (
				SELECT id FROM routes
				WHERE room_id = $1 AND member_id = $2 AND ended_at IS NULL
				FOR UPDATE
			)
			RETURNING id, member_id, ended_at
		`, room.ID, memberID).Scan(&routeEnd.ID, &routeEnd.MemberID, &routeEnd.EndedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Room{}, domain.Member{}, nil, domain.ErrNoActiveRoute
		}
		if err != nil {
			return domain.Room{}, domain.Member{}, nil, err
		}
		routeEnds = append(routeEnds, routeEnd)
	}
	if len(routeEnds) == 0 {
		return domain.Room{}, domain.Member{}, nil, domain.ErrNoActiveRoute
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Room{}, domain.Member{}, nil, err
	}
	return room, member, routeEnds, nil
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

func (r *SessionRepository) Leave(ctx context.Context, code string, memberID int64) (domain.Room, domain.Member, []domain.RouteEnd, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Room{}, domain.Member{}, nil, err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	room, err := lockSessionRoom(ctx, tx, code)
	if err != nil {
		return domain.Room{}, domain.Member{}, nil, err
	}
	member, err := lockSessionMember(ctx, tx, room.ID, memberID)
	if err != nil {
		return domain.Room{}, domain.Member{}, nil, err
	}
	routeEnds := make([]domain.RouteEnd, 0, 1)
	if memberID == room.OwnerMemberID {
		rows, queryErr := tx.Query(ctx, `
			UPDATE routes SET ended_at = now()
			WHERE room_id = $1 AND ended_at IS NULL
			RETURNING id, member_id, ended_at
		`, room.ID)
		if queryErr != nil {
			return domain.Room{}, domain.Member{}, nil, queryErr
		}
		for rows.Next() {
			var routeEnd domain.RouteEnd
			if scanErr := rows.Scan(&routeEnd.ID, &routeEnd.MemberID, &routeEnd.EndedAt); scanErr != nil {
				rows.Close()
				return domain.Room{}, domain.Member{}, nil, scanErr
			}
			routeEnds = append(routeEnds, routeEnd)
		}
		if rowsErr := rows.Err(); rowsErr != nil {
			rows.Close()
			return domain.Room{}, domain.Member{}, nil, rowsErr
		}
		rows.Close()
	} else {
		var routeEnd domain.RouteEnd
		err = tx.QueryRow(ctx, `
			UPDATE routes SET ended_at = now()
			WHERE id = (
				SELECT id FROM routes
				WHERE room_id = $1 AND member_id = $2 AND ended_at IS NULL
				FOR UPDATE
			)
			RETURNING id, member_id, ended_at
		`, room.ID, memberID).Scan(&routeEnd.ID, &routeEnd.MemberID, &routeEnd.EndedAt)
		if err == nil {
			routeEnds = append(routeEnds, routeEnd)
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return domain.Room{}, domain.Member{}, nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Room{}, domain.Member{}, nil, err
	}
	return room, member, routeEnds, nil
}

func lockSessionRoom(ctx context.Context, tx pgx.Tx, code string) (domain.Room, error) {
	var room domain.Room
	var closedAt pgtype.Timestamptz
	err := tx.QueryRow(ctx, `
		SELECT rooms.id, rooms.code, rooms.title, rooms.owner_user_id,
		       members.id, rooms.is_private, rooms.closed_at
		FROM rooms
		JOIN members ON members.room_id = rooms.id AND members.user_id = rooms.owner_user_id
		WHERE rooms.code = $1
		FOR UPDATE OF rooms
	`, code).Scan(&room.ID, &room.Code, &room.Title, &room.OwnerUserID, &room.OwnerMemberID, &room.IsPrivate, &closedAt)
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
