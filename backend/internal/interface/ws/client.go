package ws

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jphacks/kz_2606/backend/internal/domain"
	"github.com/jphacks/kz_2606/backend/internal/usecase"
)

const (
	maxMessageSize = 64 * 1024
	sendBufferSize = 64
	pingInterval   = 30 * time.Second
	pongWait       = 70 * time.Second
	writeWait      = 10 * time.Second
	sessionTimeout = 10 * time.Second
)

type recordingSession interface {
	StartRoute(context.Context, string, int64) (domain.Room, domain.Member, domain.Route, error)
	AppendPoints(context.Context, string, int64, []domain.Point) (int64, domain.Member, error)
	StopRoute(context.Context, string, int64) (domain.Room, domain.Member, int64, time.Time, error)
	UpdateMember(context.Context, string, int64, *string, *string) (domain.Member, bool, error)
	Leave(context.Context, string, int64) (domain.Member, int64, time.Time, bool, error)
}

type client struct {
	conn          *websocket.Conn
	out           chan []byte
	done          chan struct{}
	closeAfter    chan struct{}
	registry      *Registry
	sessions      recordingSession
	roomCode      string
	memberID      int64
	onLeave       func()
	closeOne      sync.Once
	closeAfterOne sync.Once
}

func (c *client) enqueue(message []byte) {
	message = append([]byte(nil), message...)
	select {
	case <-c.done:
		return
	case <-c.closeAfter:
		return
	default:
	}
	select {
	case c.out <- message:
	case <-c.done:
	case <-c.closeAfter:
	default:
		slog.Warn("websocket send buffer full; disconnecting client", "room", c.roomCode, "member", c.memberID)
		c.shutdown()
	}
}

func (c *client) disconnectAfter(message []byte) {
	c.enqueue(message)
	if c.closeAfter == nil {
		c.shutdown()
		return
	}
	c.closeAfterOne.Do(func() {
		close(c.closeAfter)
	})
}

func (c *client) shutdown() {
	c.closeOne.Do(func() {
		close(c.done)
		lastConnection := c.registry.Remove(c)
		if err := c.conn.Close(); err != nil {
			slog.Debug("close websocket connection failed", "err", err)
		}
		if lastConnection {
			if c.onLeave != nil {
				go c.onLeave()
			} else {
				c.registry.BroadcastLeave(c.roomCode, c.memberID)
			}
		}
	})
}

func (c *client) readPump() {
	defer c.shutdown()
	c.conn.SetReadLimit(maxMessageSize)
	if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		slog.Debug("set websocket read deadline failed", "err", err)
		return
	}
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})
	for {
		messageType, payload, err := c.conn.ReadMessage()
		if err != nil {
			slog.Debug("read websocket message failed", "err", err)
			return
		}
		if messageType != websocket.TextMessage {
			c.sendError("invalid_format", "テキストメッセージを送信してください")
			continue
		}
		c.handleInbound(payload)
	}
}

func (c *client) writePump(initial []byte) {
	ticker := time.NewTicker(pingInterval)
	defer func() {
		ticker.Stop()
		c.shutdown()
	}()

	if !c.writeMessage(initial) {
		return
	}
	for {
		select {
		case <-c.done:
			return
		case <-c.closeAfter:
			for {
				select {
				case message := <-c.out:
					if !c.writeMessage(message) {
						return
					}
				default:
					return
				}
			}
		case message := <-c.out:
			if !c.writeMessage(message) {
				return
			}
		case <-ticker.C:
			if err := c.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(writeWait)); err != nil {
				slog.Debug("send websocket ping failed", "err", err)
				return
			}
		}
	}
}

func (c *client) writeMessage(message []byte) bool {
	if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
		slog.Debug("set websocket write deadline failed", "err", err)
		return false
	}
	if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
		slog.Debug("write websocket message failed", "err", err)
		return false
	}
	return true
}

func (c *client) handleInbound(payload []byte) {
	var message struct {
		Type string `json:"t"`
	}
	if err := json.Unmarshal(payload, &message); err != nil {
		c.sendError("invalid_format", "メッセージ形式が正しくありません")
		return
	}
	switch message.Type {
	case "start":
		c.handleStart()
	case "points":
		c.handlePoints(payload)
	case "stop":
		c.handleStop()
	case "member":
		c.handleMember(payload)
	}
}

func (c *client) handleStart() {
	if c.sessions == nil {
		c.sendError("service_unavailable", "記録機能を利用できません")
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), sessionTimeout)
	defer cancel()
	_, member, route, err := c.sessions.StartRoute(ctx, c.roomCode, c.memberID)
	if err != nil {
		c.sendSessionError(err)
		return
	}
	c.registry.BroadcastRoute(c.roomCode, route, member, 0)
	c.registry.BroadcastMember(c.roomCode, member, true, 0)
}

func (c *client) handlePoints(payload []byte) {
	if c.sessions == nil {
		c.sendError("service_unavailable", "記録機能を利用できません")
		return
	}
	var message struct {
		Points [][]json.Number `json:"pts"`
	}
	if err := json.Unmarshal(payload, &message); err != nil {
		c.sendError("invalid_format", "メッセージ形式が正しくありません")
		return
	}
	if len(message.Points) == 0 || len(message.Points) > usecase.MaxPointsPerMessage {
		c.sendError("invalid_points", "点の数は1〜500点で指定してください")
		return
	}
	points := make([]domain.Point, 0, len(message.Points))
	for _, tuple := range message.Points {
		if len(tuple) != 4 {
			c.sendError("invalid_points", "点は [lat, lng, ts, acc] の形式で指定してください")
			return
		}
		lat, errLat := tuple[0].Float64()
		lng, errLng := tuple[1].Float64()
		ts, errTS := tuple[2].Int64()
		acc, errAcc := tuple[3].Float64()
		if errLat != nil || errLng != nil || errTS != nil || errAcc != nil ||
			math.IsNaN(lat) || math.IsInf(lat, 0) || math.IsNaN(lng) || math.IsInf(lng, 0) ||
			math.IsNaN(acc) || math.IsInf(acc, 0) || math.Abs(acc) > math.MaxFloat32 {
			c.sendError("invalid_points", "点の値が正しくありません")
			return
		}
		points = append(points, domain.Point{Lat: lat, Lng: lng, TS: ts, Acc: float32(acc)})
	}
	ctx, cancel := context.WithTimeout(context.Background(), sessionTimeout)
	defer cancel()
	routeID, _, err := c.sessions.AppendPoints(ctx, c.roomCode, c.memberID, points)
	if err != nil {
		c.sendSessionError(err)
		return
	}
	c.registry.BroadcastPoints(c.roomCode, routeID, points, c.memberID)
}

func (c *client) handleStop() {
	if c.sessions == nil {
		c.sendError("service_unavailable", "記録機能を利用できません")
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), sessionTimeout)
	defer cancel()
	_, member, routeID, endedAt, err := c.sessions.StopRoute(ctx, c.roomCode, c.memberID)
	if err != nil {
		c.sendSessionError(err)
		return
	}
	c.registry.BroadcastRouteEnd(c.roomCode, routeID, c.memberID, endedAt, 0)
	c.registry.BroadcastMember(c.roomCode, member, false, 0)
}

func (c *client) handleMember(payload []byte) {
	if c.sessions == nil {
		c.sendError("service_unavailable", "記録機能を利用できません")
		return
	}
	var message struct {
		DisplayName *string `json:"displayName"`
		Color       *string `json:"color"`
	}
	if err := json.Unmarshal(payload, &message); err != nil {
		c.sendError("invalid_format", "メッセージ形式が正しくありません")
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), sessionTimeout)
	defer cancel()
	member, recording, err := c.sessions.UpdateMember(ctx, c.roomCode, c.memberID, message.DisplayName, message.Color)
	if err != nil {
		c.sendSessionError(err)
		return
	}
	c.registry.BroadcastMember(c.roomCode, member, recording, 0)
}

func (c *client) handleLeave() {
	if c.sessions == nil {
		c.registry.BroadcastLeave(c.roomCode, c.memberID)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), sessionTimeout)
	defer cancel()
	member, routeID, endedAt, ended, err := c.sessions.Leave(ctx, c.roomCode, c.memberID)
	if err != nil {
		slog.Error("close route after websocket disconnect failed", "room", c.roomCode, "member", c.memberID, "err", err)
		c.registry.BroadcastLeave(c.roomCode, c.memberID)
		return
	}
	if ended {
		c.registry.BroadcastRouteEnd(c.roomCode, routeID, c.memberID, endedAt, 0)
		c.registry.BroadcastMember(c.roomCode, member, false, 0)
	}
	c.registry.BroadcastLeave(c.roomCode, c.memberID)
}

func (c *client) sendSessionError(err error) {
	switch {
	case errors.Is(err, domain.ErrRoomClosed):
		c.sendError("room_closed", "この部屋は締め切られています")
	case errors.Is(err, domain.ErrNoActiveRoute):
		c.sendError("no_active_route", "記録が開始されていません")
	case errors.Is(err, domain.ErrRouteInUse):
		c.sendError("route_in_use", "他のメンバーが記録中です")
	case errors.Is(err, domain.ErrInvalidPoints):
		c.sendError("invalid_points", "点の値が正しくありません")
	case errors.Is(err, domain.ErrMemberBanned):
		c.sendError("banned", "この部屋への参加は禁止されています")
	case errors.Is(err, domain.ErrMemberNotFound):
		c.sendError("member_not_found", "メンバーが見つかりません")
	case errors.Is(err, domain.ErrRoomNotFound), errors.Is(err, domain.ErrInvalidRoomCode):
		c.sendError("room_not_found", "部屋が見つかりません")
	default:
		slog.Error("websocket recording operation failed", "room", c.roomCode, "member", c.memberID, "err", err)
		c.sendError("internal_error", "サーバーエラーが発生しました")
	}
}

func (c *client) sendError(code, message string) {
	payload, err := json.Marshal(struct {
		Type    string `json:"t"`
		Code    string `json:"code"`
		Message string `json:"message"`
	}{Type: "error", Code: code, Message: message})
	if err != nil {
		slog.Error("encode websocket error failed", "err", err)
		return
	}
	c.enqueue(payload)
}
