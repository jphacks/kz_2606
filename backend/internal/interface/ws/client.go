package ws

import (
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	maxMessageSize = 64 * 1024
	sendBufferSize = 64
	pingInterval   = 30 * time.Second
	pongWait       = 70 * time.Second
	writeWait      = 10 * time.Second
)

type client struct {
	conn          *websocket.Conn
	out           chan []byte
	done          chan struct{}
	closeAfter    chan struct{}
	registry      *Registry
	roomCode      string
	memberID      int64
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
		if c.registry.Remove(c) {
			c.registry.BroadcastLeave(c.roomCode, c.memberID)
		}
		if err := c.conn.Close(); err != nil {
			slog.Debug("close websocket connection failed", "err", err)
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
	case "start", "points", "stop", "member":
		c.sendError("not_implemented", "この操作はまだ利用できません")
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
