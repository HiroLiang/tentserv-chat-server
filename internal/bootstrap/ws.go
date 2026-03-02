package bootstrap

import (
	"net/http"

	"github.com/HiroLiang/goat-server/internal/application/shared"
	"github.com/HiroLiang/goat-server/internal/interface/http/middleware"
	"github.com/HiroLiang/goat-server/internal/interface/ws"
	wsChat "github.com/HiroLiang/goat-server/internal/interface/ws/handler/chat"
	wsGame "github.com/HiroLiang/goat-server/internal/interface/ws/handler/game"
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

// BuildWsComponents creates and starts the Hub, and wires all message handlers
// onto the router.
func BuildWsComponents(deps *Dependencies) (*ws.Hub, *ws.MessageRouter) {
	hub := ws.NewHub()
	go hub.Run()

	router := ws.NewMessageRouter()
	router.Register("chat.send", wsChat.NewMessageHandler())
	router.Register("game.move", wsGame.NewMoveHandler())

	return hub, router
}

// RegisterWsRoutes registers the single /ws upgrade endpoint.
func RegisterWsRoutes(r *gin.Engine, hub *ws.Hub, router *ws.MessageRouter, deps *Dependencies) {
	r.GET("/ws/",
		middleware.AuthMiddleware(deps.TokenService),
		func(c *gin.Context) {
			conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
			if err != nil {
				return
			}

			userID := ""
			if v, ok := c.Get("authContext"); ok {
				userID = v.(*shared.AuthContext).UserID
			}

			client := ws.NewClient(hub, conn, userID)
			hub.Register <- client

			go client.WritePump()
			go client.ReadPump(router)
		},
	)
}
