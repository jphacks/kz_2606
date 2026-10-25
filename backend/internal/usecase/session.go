package usecase

import (
	"context"
	"math"
	"strings"
	"time"

	"github.com/jphacks/kz_2606/backend/internal/domain"
)

var (
	ErrRoomClosed    = domain.ErrRoomClosed
	ErrNoActiveRoute = domain.ErrNoActiveRoute
	ErrRouteInUse    = domain.ErrRouteInUse
	ErrInvalidPoints = domain.ErrInvalidPoints
)

const MaxPointsPerMessage = 500

type SessionService struct {
	sessions SessionRepository
}

func NewSessionService(sessions SessionRepository) *SessionService {
	return &SessionService{sessions: sessions}
}

func (s *SessionService) StartRoute(ctx context.Context, code string, memberID int64) (domain.Room, domain.Member, domain.Route, error) {
	normalizedCode, err := domain.NormalizeRoomCode(code)
	if err != nil {
		return domain.Room{}, domain.Member{}, domain.Route{}, err
	}
	return s.sessions.StartRoute(ctx, normalizedCode, memberID)
}

func (s *SessionService) AppendPoints(ctx context.Context, code string, memberID int64, points []domain.Point) (int64, domain.Member, error) {
	normalizedCode, err := domain.NormalizeRoomCode(code)
	if err != nil {
		return 0, domain.Member{}, err
	}
	if len(points) == 0 || len(points) > MaxPointsPerMessage {
		return 0, domain.Member{}, ErrInvalidPoints
	}
	for _, point := range points {
		if math.IsNaN(point.Lat) || math.IsInf(point.Lat, 0) || point.Lat < -90 || point.Lat > 90 ||
			math.IsNaN(point.Lng) || math.IsInf(point.Lng, 0) || point.Lng < -180 || point.Lng > 180 ||
			point.TS < 0 || math.IsNaN(float64(point.Acc)) || math.IsInf(float64(point.Acc), 0) || point.Acc < 0 {
			return 0, domain.Member{}, ErrInvalidPoints
		}
	}
	return s.sessions.AppendPoints(ctx, normalizedCode, memberID, points)
}

func (s *SessionService) StopRoute(ctx context.Context, code string, memberID int64) (domain.Room, domain.Member, int64, time.Time, error) {
	normalizedCode, err := domain.NormalizeRoomCode(code)
	if err != nil {
		return domain.Room{}, domain.Member{}, 0, time.Time{}, err
	}
	return s.sessions.StopRoute(ctx, normalizedCode, memberID)
}

func (s *SessionService) UpdateMember(ctx context.Context, code string, memberID int64, displayName, color *string) (domain.Member, bool, error) {
	normalizedCode, err := domain.NormalizeRoomCode(code)
	if err != nil {
		return domain.Member{}, false, err
	}
	var normalizedName, normalizedColor *string
	if displayName != nil {
		if name := strings.TrimSpace(*displayName); name != "" {
			trimmedName, nameErr := domain.NormalizeDisplayName(name)
			if nameErr == nil {
				normalizedName = &trimmedName
			} else if len([]rune(name)) > 16 {
				truncated := string([]rune(name)[:16])
				normalizedName = &truncated
			}
		}
	}
	if color != nil {
		if normalized, colorErr := domain.NormalizeColor(*color); colorErr == nil {
			normalizedColor = &normalized
		}
	}
	return s.sessions.UpdateMember(ctx, normalizedCode, memberID, normalizedName, normalizedColor)
}

func (s *SessionService) Leave(ctx context.Context, code string, memberID int64) (domain.Member, int64, time.Time, bool, error) {
	normalizedCode, err := domain.NormalizeRoomCode(code)
	if err != nil {
		return domain.Member{}, 0, time.Time{}, false, err
	}
	return s.sessions.Leave(ctx, normalizedCode, memberID)
}
