package usecase

import (
	"context"
	"math"
	"testing"

	"github.com/jphacks/kz_2606/backend/internal/domain"
)

type recordingSessionRepository struct {
	member      domain.Member
	recording   bool
	points      []domain.Point
	displayName *string
	color       *string
}

func (r *recordingSessionRepository) StartRoute(context.Context, string, int64) (domain.Room, domain.Member, domain.Route, error) {
	return domain.Room{}, r.member, domain.Route{}, nil
}

func (r *recordingSessionRepository) AppendPoints(_ context.Context, _ string, _ int64, points []domain.Point) (int64, domain.Member, error) {
	r.points = append(r.points, points...)
	return 8, r.member, nil
}

func (r *recordingSessionRepository) StopRoute(context.Context, string, int64) (domain.Room, domain.Member, int64, error) {
	return domain.Room{}, r.member, 8, nil
}

func (r *recordingSessionRepository) UpdateMember(_ context.Context, _ string, _ int64, displayName, color *string) (domain.Member, bool, error) {
	r.displayName = displayName
	r.color = color
	if displayName != nil {
		r.member.DisplayName = *displayName
	}
	if color != nil {
		r.member.Color = *color
	}
	return r.member, r.recording, nil
}

func (r *recordingSessionRepository) Leave(context.Context, string, int64) (domain.Member, int64, bool, error) {
	return r.member, 0, false, nil
}

func TestSessionServiceValidatesAndNormalizesPoints(t *testing.T) {
	repository := &recordingSessionRepository{}
	service := NewSessionService(repository)
	points := []domain.Point{{Lat: 43.06, Lng: 141.35, TS: 1761278400000, Acc: 8.2}}
	if _, _, err := service.AppendPoints(context.Background(), "KZ26AB", 3, points); err != nil {
		t.Fatalf("append valid points: %v", err)
	}
	if len(repository.points) != 1 || repository.points[0] != points[0] {
		t.Fatalf("stored points = %+v", repository.points)
	}

	maxBatch := make([]domain.Point, MaxPointsPerMessage)
	for i := range maxBatch {
		maxBatch[i] = domain.Point{Lat: 43.06, Lng: 141.35, TS: int64(i), Acc: 8.2}
	}
	if _, _, err := service.AppendPoints(context.Background(), "KZ26AB", 3, maxBatch); err != nil {
		t.Fatalf("append maximum batch: %v", err)
	}
	tooMany := append(maxBatch, domain.Point{Lat: 43.06, Lng: 141.35, TS: 501})
	if _, _, err := service.AppendPoints(context.Background(), "KZ26AB", 3, tooMany); err != ErrInvalidPoints {
		t.Fatalf("append oversized batch error = %v, want %v", err, ErrInvalidPoints)
	}

	for name, invalid := range map[string]domain.Point{
		"latitude":   {Lat: 91, Lng: 0, TS: 1},
		"longitude":  {Lat: 0, Lng: 181, TS: 1},
		"timestamp":  {Lat: 0, Lng: 0, TS: -1},
		"accuracy":   {Lat: 0, Lng: 0, TS: 1, Acc: -1},
		"not finite": {Lat: math.NaN(), Lng: 0, TS: 1},
	} {
		t.Run(name, func(t *testing.T) {
			if _, _, err := service.AppendPoints(context.Background(), "KZ26AB", 3, []domain.Point{invalid}); err != ErrInvalidPoints {
				t.Fatalf("append invalid point error = %v, want %v", err, ErrInvalidPoints)
			}
		})
	}
}

func TestSessionServiceIgnoresInvalidMemberValuesAndTruncatesLongName(t *testing.T) {
	repository := &recordingSessionRepository{
		member:    domain.Member{ID: 3, DisplayName: "walker", Color: "#00e5ff"},
		recording: true,
	}
	service := NewSessionService(repository)
	longName := "12345678901234567"
	invalidColor := "red"
	member, recording, err := service.UpdateMember(context.Background(), "KZ26AB", 3, &longName, &invalidColor)
	if err != nil {
		t.Fatalf("update member: %v", err)
	}
	if member.DisplayName != "1234567890123456" || member.Color != "#00e5ff" || !recording {
		t.Fatalf("member = %+v, recording = %v", member, recording)
	}
	if repository.color != nil {
		t.Fatalf("invalid color was sent to repository: %q", *repository.color)
	}

	emptyName := "   "
	member, _, err = service.UpdateMember(context.Background(), "KZ26AB", 3, &emptyName, nil)
	if err != nil {
		t.Fatalf("update member with empty name: %v", err)
	}
	if repository.displayName != nil || member.DisplayName != "1234567890123456" {
		t.Fatalf("empty name changed the member: %+v", member)
	}
}
