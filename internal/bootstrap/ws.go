package bootstrap

import (
	"net/http"
	"strconv"

	"github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
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

// BuildWsComponents wires all message handlers onto the router.
// The Hub is already created and started in BuildDeps.
func BuildWsComponents(deps *Dependencies, useCases *UseCases) (*ws.Hub, *ws.MessageRouter) {
	router := ws.NewMessageRouter()
	router.Register("chat.send", wsChat.NewMessageHandler(useCases.SendMessageUseCase))
	router.Register("game.move", wsGame.NewMoveHandler())
	router.Register("system.ack", wsSystem.NewAckHandler(deps.Hub))

	return deps.Hub, router
}

// RegisterWsRoutes registers the single /ws upgrade endpoint.
func RegisterWsRoutes(r *gin.Engine, hub *ws.Hub, router *ws.MessageRouter, deps *Dependencies) {
	r.GET("/ws/",
		middleware.AuthMiddleware(deps.SessionManager, deps.UserRepo),
		func(c *gin.Context) {
			conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
			if err != nil {
				return
			}

			userID := ""
			if v, ok := c.Get("authContext"); ok {
				userID = strconv.FormatInt(int64(v.(*shared.AuthContext).UserID), 10)
			}

			client := ws.NewClient(hub, conn, userID)
			hub.Register <- client

			go client.WritePump()
			go client.ReadPump(router)
		},
	)
}
