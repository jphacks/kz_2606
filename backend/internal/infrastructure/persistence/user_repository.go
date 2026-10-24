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

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) CreateWithPersonalRoom(
	ctx context.Context,
	username, passwordHash, roomCode, roomTitle string,
) (domain.User, domain.Room, domain.Member, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.User{}, domain.Room{}, domain.Member{}, err
	}
	defer func() {
		_ = tx.Rollback(context.Background())
	}()

	var user domain.User
	err = tx.QueryRow(ctx, `
		INSERT INTO users (username, password_hash)
		VALUES ($1, $2)
		RETURNING id, username, created_at
	`, username, passwordHash).Scan(&user.ID, &user.Username, &user.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		// users currently has only one UNIQUE constraint: username.
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.User{}, domain.Room{}, domain.Member{}, domain.ErrUsernameTaken
		}
		return domain.User{}, domain.Room{}, domain.Member{}, err
	}

	var room domain.Room
	err = tx.QueryRow(ctx, `
		INSERT INTO rooms (code, title, owner_user_id, is_private)
		VALUES ($1, $2, $3, TRUE)
		RETURNING id, code, title, is_private
	`, roomCode, roomTitle, user.ID).Scan(&room.ID, &room.Code, &room.Title, &room.IsPrivate)
	if err != nil {
		var pgErr *pgconn.PgError
		// rooms currently has only one UNIQUE constraint: code.
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.User{}, domain.Room{}, domain.Member{}, domain.ErrRoomCodeTaken
		}
		return domain.User{}, domain.Room{}, domain.Member{}, err
	}
	room.OwnerUserID = user.ID

	member, err := insertMember(ctx, tx, room.ID, user.ID, user.Username, domain.FirstMemberColor)
	if err != nil {
		return domain.User{}, domain.Room{}, domain.Member{}, err
	}
	room.OwnerMemberID = member.ID
	if err := tx.Commit(ctx); err != nil {
		return domain.User{}, domain.Room{}, domain.Member{}, err
	}
	return user, room, member, nil
}

func (r *UserRepository) FindByUsername(ctx context.Context, username string) (domain.User, string, error) {
	var user domain.User
	var passwordHash string
	err := r.pool.QueryRow(ctx, `
		SELECT id, username, password_hash, created_at
		FROM users
		WHERE username = $1
	`, username).Scan(&user.ID, &user.Username, &passwordHash, &user.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, "", domain.ErrUserNotFound
	}
	if err != nil {
		return domain.User{}, "", err
	}
	return user, passwordHash, nil
}

func (r *UserRepository) FindByID(ctx context.Context, id int64) (domain.User, error) {
	var user domain.User
	err := r.pool.QueryRow(ctx, `
		SELECT id, username, created_at
		FROM users
		WHERE id = $1
	`, id).Scan(&user.ID, &user.Username, &user.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, domain.ErrUserNotFound
	}
	if err != nil {
		return domain.User{}, err
	}
	return user, nil
}

func (r *UserRepository) FindPersonalRoom(ctx context.Context, ownerUserID int64) (domain.Room, error) {
	var room domain.Room
	var closedAt pgtype.Timestamptz
	err := r.pool.QueryRow(ctx, `
		SELECT rooms.id, rooms.code, rooms.title, rooms.owner_user_id,
		       members.id, rooms.is_private, rooms.closed_at
		FROM rooms
		JOIN members ON members.room_id = rooms.id AND members.user_id = rooms.owner_user_id
		WHERE rooms.owner_user_id = $1 AND rooms.is_private = TRUE
		ORDER BY rooms.id
		LIMIT 1
	`, ownerUserID).Scan(
		&room.ID, &room.Code, &room.Title, &room.OwnerUserID,
		&room.OwnerMemberID, &room.IsPrivate, &closedAt,
	)
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
