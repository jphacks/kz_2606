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

type RoomRepository struct {
	pool *pgxpool.Pool
}

func NewRoomRepository(pool *pgxpool.Pool) *RoomRepository {
	return &RoomRepository{pool: pool}
}

func (r *RoomRepository) Create(ctx context.Context, owner domain.User, code, title string) (domain.Room, domain.Member, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Room{}, domain.Member{}, err
	}
	defer func() {
		_ = tx.Rollback(context.Background())
	}()

	var room domain.Room
	err = tx.QueryRow(ctx, `
		INSERT INTO rooms (code, title, owner_user_id)
		VALUES ($1, $2, $3)
		RETURNING id, code, title
	`, code, title, owner.ID).Scan(&room.ID, &room.Code, &room.Title)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.Room{}, domain.Member{}, domain.ErrRoomCodeTaken
		}
		return domain.Room{}, domain.Member{}, err
	}

	ownerMember, err := insertMember(ctx, tx, room.ID, owner.ID, owner.Username, domain.FirstMemberColor)
	if err != nil {
		return domain.Room{}, domain.Member{}, err
	}
	room.OwnerMemberID = ownerMember.ID
	room.OwnerUserID = owner.ID
	if err := tx.Commit(ctx); err != nil {
		return domain.Room{}, domain.Member{}, err
	}
	return room, ownerMember, nil
}

func (r *RoomRepository) FindByCode(ctx context.Context, code string) (domain.Room, error) {
	return findRoomByCode(ctx, r.pool, code)
}

func (r *RoomRepository) Snapshot(ctx context.Context, code string, userID int64) (domain.RoomSnapshot, domain.Member, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel:   pgx.RepeatableRead,
		AccessMode: pgx.ReadOnly,
	})
	if err != nil {
		return domain.RoomSnapshot{}, domain.Member{}, err
	}
	defer func() {
		_ = tx.Rollback(context.Background())
	}()

	room, err := findRoomByCode(ctx, tx, code)
	if err != nil {
		return domain.RoomSnapshot{}, domain.Member{}, err
	}

	if room.IsPrivate && room.OwnerUserID != userID {
		return domain.RoomSnapshot{}, domain.Member{}, domain.ErrPrivateRoom
	}
	you, err := readMember(ctx, tx, room.ID, userID)
	if err != nil {
		return domain.RoomSnapshot{}, domain.Member{}, err
	}
	if you.BannedAt != nil {
		return domain.RoomSnapshot{}, domain.Member{}, domain.ErrMemberBanned
	}

	snapshot := domain.RoomSnapshot{Room: room}
	memberRows, err := tx.Query(ctx, `
		SELECT id, display_name, color, banned_at
		FROM members
		WHERE room_id = $1
		ORDER BY id
	`, room.ID)
	if err != nil {
		return domain.RoomSnapshot{}, domain.Member{}, err
	}
	for memberRows.Next() {
		var member domain.Member
		var bannedAt pgtype.Timestamptz
		if err := memberRows.Scan(&member.ID, &member.DisplayName, &member.Color, &bannedAt); err != nil {
			memberRows.Close()
			return domain.RoomSnapshot{}, domain.Member{}, err
		}
		if bannedAt.Valid {
			member.BannedAt = timePointer(bannedAt.Time)
		}
		snapshot.Members = append(snapshot.Members, member)
	}
	if err := memberRows.Err(); err != nil {
		memberRows.Close()
		return domain.RoomSnapshot{}, domain.Member{}, err
	}
	memberRows.Close()

	routeRows, err := tx.Query(ctx, `
		SELECT routes.id, routes.member_id, routes.started_at, routes.ended_at,
		       points.latitude, points.longitude,
		       (EXTRACT(EPOCH FROM points.recorded_at) * 1000)::BIGINT,
		       COALESCE(points.accuracy, 0)::REAL
		FROM routes
		LEFT JOIN points ON points.route_id = routes.id
		WHERE routes.room_id = $1
		ORDER BY routes.id, points.recorded_at
	`, room.ID)
	if err != nil {
		return domain.RoomSnapshot{}, domain.Member{}, err
	}
	routeIndexes := make(map[int64]int)
	for routeRows.Next() {
		var routeID, memberID int64
		var startedAt time.Time
		var endedAt pgtype.Timestamptz
		var latitude, longitude pgtype.Float8
		var timestamp pgtype.Int8
		var accuracy pgtype.Float4
		if err := routeRows.Scan(
			&routeID, &memberID, &startedAt, &endedAt,
			&latitude, &longitude, &timestamp, &accuracy,
		); err != nil {
			routeRows.Close()
			return domain.RoomSnapshot{}, domain.Member{}, err
		}
		index, exists := routeIndexes[routeID]
		if !exists {
			route := domain.Route{
				ID:        routeID,
				MemberID:  memberID,
				StartedAt: startedAt,
				Points:    make([]domain.Point, 0),
			}
			if endedAt.Valid {
				route.EndedAt = timePointer(endedAt.Time)
			}
			snapshot.Routes = append(snapshot.Routes, route)
			index = len(snapshot.Routes) - 1
			routeIndexes[routeID] = index
		}
		if latitude.Valid {
			snapshot.Routes[index].Points = append(snapshot.Routes[index].Points, domain.Point{
				Lat: latitude.Float64,
				Lng: longitude.Float64,
				TS:  timestamp.Int64,
				Acc: accuracy.Float32,
			})
		}
	}
	if err := routeRows.Err(); err != nil {
		routeRows.Close()
		return domain.RoomSnapshot{}, domain.Member{}, err
	}
	routeRows.Close()
	if err := tx.Commit(ctx); err != nil {
		return domain.RoomSnapshot{}, domain.Member{}, err
	}
	return snapshot, you, nil
}

func (r *RoomRepository) Join(ctx context.Context, code string, userID int64, displayName, color string) (domain.Room, domain.Member, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Room{}, domain.Member{}, err
	}
	defer func() {
		_ = tx.Rollback(context.Background())
	}()

	room, err := findRoomByCode(ctx, tx, code)
	if err != nil {
		return domain.Room{}, domain.Member{}, err
	}
	if room.IsPrivate && room.OwnerUserID != userID {
		return domain.Room{}, domain.Member{}, domain.ErrPrivateRoom
	}

	member, err := insertMember(ctx, tx, room.ID, userID, displayName, color)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return domain.Room{}, domain.Member{}, err
	}
	if errors.Is(err, pgx.ErrNoRows) {
		member, err = findMember(ctx, tx, room.ID, userID)
		if err != nil {
			return domain.Room{}, domain.Member{}, err
		}
	}
	if member.BannedAt != nil {
		return domain.Room{}, domain.Member{}, domain.ErrMemberBanned
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Room{}, domain.Member{}, err
	}
	return room, member, nil
}

func readMember(ctx context.Context, tx pgx.Tx, roomID, userID int64) (domain.Member, error) {
	var member domain.Member
	var bannedAt pgtype.Timestamptz
	err := tx.QueryRow(ctx, `
		SELECT id, display_name, color, banned_at
		FROM members
		WHERE room_id = $1 AND user_id = $2
	`, roomID, userID).Scan(&member.ID, &member.DisplayName, &member.Color, &bannedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Member{}, domain.ErrMemberNotFound
	}
	if err != nil {
		return domain.Member{}, err
	}
	if bannedAt.Valid {
		member.BannedAt = timePointer(bannedAt.Time)
	}
	return member, nil
}

func (r *RoomRepository) Close(ctx context.Context, code string, userID int64) (time.Time, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return time.Time{}, err
	}
	defer func() {
		_ = tx.Rollback(context.Background())
	}()

	roomID, ownerUserID, err := lockRoomOwner(ctx, tx, code)
	if err != nil {
		return time.Time{}, err
	}
	if ownerUserID != userID {
		return time.Time{}, domain.ErrForbidden
	}
	var closedAt time.Time
	err = tx.QueryRow(ctx, `
		UPDATE rooms
		SET closed_at = COALESCE(closed_at, now())
		WHERE id = $1
		RETURNING closed_at
	`, roomID).Scan(&closedAt)
	if err != nil {
		return time.Time{}, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE routes
		SET ended_at = $2
		WHERE room_id = $1 AND ended_at IS NULL
	`, roomID, closedAt); err != nil {
		return time.Time{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return time.Time{}, err
	}
	return closedAt, nil
}

func (r *RoomRepository) Reopen(ctx context.Context, code string, userID int64) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(context.Background())
	}()

	roomID, ownerUserID, err := lockRoomOwner(ctx, tx, code)
	if err != nil {
		return err
	}
	if ownerUserID != userID {
		return domain.ErrForbidden
	}
	if _, err := tx.Exec(ctx, `UPDATE rooms SET closed_at = NULL WHERE id = $1`, roomID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *RoomRepository) Ban(ctx context.Context, code string, userID, memberID int64) error {
	return r.setBanned(ctx, code, userID, memberID, true)
}

func (r *RoomRepository) Unban(ctx context.Context, code string, userID, memberID int64) error {
	return r.setBanned(ctx, code, userID, memberID, false)
}

func (r *RoomRepository) setBanned(ctx context.Context, code string, userID, memberID int64, banned bool) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(context.Background())
	}()

	roomID, ownerUserID, err := lockRoomOwner(ctx, tx, code)
	if err != nil {
		return err
	}
	if ownerUserID != userID {
		return domain.ErrForbidden
	}
	var targetUserID int64
	err = tx.QueryRow(ctx, `
		SELECT user_id
		FROM members
		WHERE id = $1 AND room_id = $2
		FOR UPDATE
	`, memberID, roomID).Scan(&targetUserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrMemberNotFound
	}
	if err != nil {
		return err
	}
	if targetUserID == ownerUserID {
		return domain.ErrCannotBanOwner
	}
	if banned {
		_, err = tx.Exec(ctx, `
			UPDATE members SET banned_at = COALESCE(banned_at, now())
			WHERE id = $1
		`, memberID)
	} else {
		_, err = tx.Exec(ctx, `UPDATE members SET banned_at = NULL WHERE id = $1`, memberID)
	}
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *RoomRepository) DeleteRoute(ctx context.Context, code string, userID, routeID int64) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(context.Background())
	}()

	roomID, ownerUserID, err := lockRoomOwner(ctx, tx, code)
	if err != nil {
		return err
	}
	var routeUserID int64
	err = tx.QueryRow(ctx, `
		SELECT members.user_id
		FROM routes
		JOIN members ON members.id = routes.member_id
		WHERE routes.id = $1 AND routes.room_id = $2
		FOR UPDATE OF routes
	`, routeID, roomID).Scan(&routeUserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrRouteNotFound
	}
	if err != nil {
		return err
	}
	if userID != ownerUserID && userID != routeUserID {
		return domain.ErrForbidden
	}
	if _, err := tx.Exec(ctx, `DELETE FROM routes WHERE id = $1 AND room_id = $2`, routeID, roomID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

type queryRower interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func lockRoomOwner(ctx context.Context, tx pgx.Tx, code string) (int64, int64, error) {
	var roomID, ownerUserID int64
	err := tx.QueryRow(ctx, `
		SELECT id, owner_user_id
		FROM rooms
		WHERE code = $1
		FOR UPDATE
	`, code).Scan(&roomID, &ownerUserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, 0, domain.ErrRoomNotFound
	}
	if err != nil {
		return 0, 0, err
	}
	return roomID, ownerUserID, nil
}

func findRoomByCode(ctx context.Context, q queryRower, code string) (domain.Room, error) {
	var room domain.Room
	var closedAt pgtype.Timestamptz
	err := q.QueryRow(ctx, `
		SELECT rooms.id, rooms.code, rooms.title, rooms.owner_user_id, members.id, rooms.is_private, rooms.closed_at
		FROM rooms
		JOIN members ON members.room_id = rooms.id AND members.user_id = rooms.owner_user_id
		WHERE rooms.code = $1
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

func insertMember(ctx context.Context, tx pgx.Tx, roomID, userID int64, displayName, color string) (domain.Member, error) {
	var member domain.Member
	var bannedAt pgtype.Timestamptz
	err := tx.QueryRow(ctx, `
		INSERT INTO members (room_id, user_id, display_name, color)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (room_id, user_id) DO NOTHING
		RETURNING id, display_name, color, banned_at
	`, roomID, userID, displayName, color).Scan(&member.ID, &member.DisplayName, &member.Color, &bannedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Member{}, pgx.ErrNoRows
	}
	if err != nil {
		return domain.Member{}, err
	}
	if bannedAt.Valid {
		member.BannedAt = timePointer(bannedAt.Time)
	}
	return member, nil
}

func findMember(ctx context.Context, tx pgx.Tx, roomID, userID int64) (domain.Member, error) {
	var member domain.Member
	var bannedAt pgtype.Timestamptz
	err := tx.QueryRow(ctx, `
		SELECT id, display_name, color, banned_at
		FROM members
		WHERE room_id = $1 AND user_id = $2
		FOR UPDATE
	`, roomID, userID).Scan(&member.ID, &member.DisplayName, &member.Color, &bannedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Member{}, domain.ErrMemberNotFound
	}
	if err != nil {
		return domain.Member{}, err
	}
	if bannedAt.Valid {
		member.BannedAt = timePointer(bannedAt.Time)
	}
	return member, nil
}

func timePointer(value time.Time) *time.Time {
	return &value
}
