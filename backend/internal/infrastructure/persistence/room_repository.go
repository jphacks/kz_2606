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
	if err := tx.Commit(ctx); err != nil {
		return domain.Room{}, domain.Member{}, err
	}
	return room, ownerMember, nil
}

func (r *RoomRepository) FindByCode(ctx context.Context, code string) (domain.Room, error) {
	return findRoomByCode(ctx, r.pool, code)
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

type queryRower interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func findRoomByCode(ctx context.Context, q queryRower, code string) (domain.Room, error) {
	var room domain.Room
	var closedAt pgtype.Timestamptz
	err := q.QueryRow(ctx, `
		SELECT rooms.id, rooms.code, rooms.title, members.id, rooms.closed_at
		FROM rooms
		JOIN members ON members.room_id = rooms.id AND members.user_id = rooms.owner_user_id
		WHERE rooms.code = $1
	`, code).Scan(&room.ID, &room.Code, &room.Title, &room.OwnerMemberID, &closedAt)
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
