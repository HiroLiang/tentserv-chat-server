package bootstrap

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	chatPort "github.com/HiroLiang/tentserv-chat-server/internal/application/chat/port"
	"github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	domainfriendship "github.com/HiroLiang/tentserv-chat-server/internal/domain/friendship"
	domainshared "github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	infraPush "github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/push"
	"github.com/HiroLiang/tentserv-chat-server/internal/interface/http/middleware"
	"github.com/HiroLiang/tentserv-chat-server/internal/interface/ws"
	wsChat "github.com/HiroLiang/tentserv-chat-server/internal/interface/ws/handler/chat"
	wsGame "github.com/HiroLiang/tentserv-chat-server/internal/interface/ws/handler/game"
	wsSystem "github.com/HiroLiang/tentserv-chat-server/internal/interface/ws/handler/system"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Origin validation is handled by the CORS middleware on the Gin engine.
		return true
	},
}

// [EN] BuildWsComponents registers WS message handlers (chat.send, game.move, system.ack) and sets
//      the onConnect hook that replays pending sender key requests when a user reconnects.
// [中] BuildWsComponents 註冊 WS 訊息處理器（chat.send、game.move、system.ack），
//      並設定 onConnect 鉤子，在使用者重新連線時重播待處理的 sender key 請求。
// [日] BuildWsComponents は WS メッセージハンドラ（chat.send、game.move、system.ack）を登録し、
//      ユーザー再接続時に保留中の sender key リクエストを再配信する onConnect フックを設定する。

// BuildWsComponents wires all message handlers onto the router.
// The Hub is already created and started in BuildDeps.
func BuildWsComponents(deps *Dependencies, useCases *UseCases) (*ws.Hub, *ws.MessageRouter) {
	router := ws.NewMessageRouter()
	router.Register("chat.send", wsChat.NewMessageHandler(useCases.SendMessageUseCase))
	router.Register("game.move", wsGame.NewMoveHandler())
	router.Register("system.ack", wsSystem.NewAckHandler(deps.Hub))

	deps.Hub.SetOnConnect(func(userID string) {
		ctx := context.Background()
		useCases.NotifyPendingSenderKeyRequestsUseCase.Execute(ctx, userID)
		useCases.NotifyPendingSenderKeyDistributionsUseCase.Execute(ctx, userID)
		parsedUserID, err := domainshared.ParseUserID(userID)
		if err != nil {
			return
		}
		infraPush.ReplayPendingForUser(ctx, deps.DeliveryQueueRepo, deps.Hub, parsedUserID)
	})
	deps.Hub.SetOnPresenceChanged(func(userID string, snapshot chatPort.PresenceSnapshot) {
		notifyPresenceStatusChanged(deps, userID, snapshot)
	})

	return deps.Hub, router
}

type presenceStatusChangedPayload struct {
	UserID     int64      `json:"user_id"`
	Status     string     `json:"status"`
	LastSeenAt *time.Time `json:"last_seen_at,omitempty"`
}

func notifyPresenceStatusChanged(deps *Dependencies, userID string, snapshot chatPort.PresenceSnapshot) {
	parsedUserID, err := domainshared.ParseUserID(userID)
	if err != nil {
		return
	}

	friendships, err := deps.FriendshipRepo.FindAllByUserID(context.Background(), parsedUserID)
	if err != nil {
		return
	}

	recipients := make([]string, 0, len(friendships))
	for _, row := range friendships {
		if row.Status != domainfriendship.StatusAccepted {
			continue
		}

		friendID := row.UserID
		if row.UserID == parsedUserID {
			friendID = row.FriendID
		}
		recipients = append(recipients, strconv.FormatInt(int64(friendID), 10))
	}
	if len(recipients) == 0 {
		return
	}

	data, err := json.Marshal(struct {
		Type    string                       `json:"type"`
		Payload presenceStatusChangedPayload `json:"payload"`
	}{
		Type: "presence.user_status_changed",
		Payload: presenceStatusChangedPayload{
			UserID:     int64(parsedUserID),
			Status:     string(snapshot.Status),
			LastSeenAt: snapshot.LastSeenAt,
		},
	})
	if err != nil {
		return
	}

	deps.Hub.SendToUsers(recipients, data)
}

// [EN] RegisterWsRoutes: upgrades GET /ws/ to WebSocket after AuthMiddleware + RequireAuth validation,
//
//	extracts userID from authContext, creates a Client, registers it with the Hub,
//	and starts ReadPump + WritePump goroutines.
//
// [中] RegisterWsRoutes：AuthMiddleware + RequireAuth 驗證後，將 GET /ws/ 升級為 WebSocket，
//
//	從 authContext 取得 userID，建立 Client，向 Hub 註冊，並啟動 ReadPump + WritePump goroutine。
//
// [日] RegisterWsRoutes：AuthMiddleware + RequireAuth 検証後、GET /ws/ を WebSocket にアップグレードし、
//
//	authContext から userID を取得して Client を作成、Hub に登録し、ReadPump + WritePump goroutine を起動する。
//
// RegisterWsRoutes registers the single /ws upgrade endpoint.
func RegisterWsRoutes(r *gin.Engine, hub *ws.Hub, router *ws.MessageRouter, deps *Dependencies) {
	r.GET("/ws/",
		middleware.AuthMiddleware(deps.SessionManager, deps.UserRepo),
		middleware.RequireAuthMiddleware(),
		func(c *gin.Context) {
			conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
			if err != nil {
				return
			}

			v, _ := c.Get("authContext")
			userID := strconv.FormatInt(int64(v.(*shared.AuthContext).UserID), 10)
			deviceID := c.GetHeader("X-Device-ID")
			if deviceID == "" {
				deviceID = c.Query("device_id")
			}

			client := ws.NewClient(hub, conn, userID, deviceID)
			hub.Register <- client

			go client.WritePump()
			go client.ReadPump(router)
		},
	)
}
