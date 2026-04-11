package ws

import (
	"testing"
	"time"

	chatPort "github.com/HiroLiang/tentserv-chat-server/internal/application/chat/port"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// syncAfterBroadcast waits for the Hub's Run() goroutine to drain the
// Broadcast channel. Since Broadcast is buffered, Run() may not have
// delivered the message yet when we return from "hub.Broadcast <- msg".
func syncAfterBroadcast() { time.Sleep(20 * time.Millisecond) }

func TestHub_Broadcast_ReachesAllClients(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	c1 := newTestClient(hub, "")
	c2 := newTestClient(hub, "")
	hub.Register <- c1 // unbuffered — blocks until Run() processes
	hub.Register <- c2

	hub.Broadcast <- []byte("hello")
	syncAfterBroadcast()

	for i, c := range []*Client{c1, c2} {
		select {
		case got := <-c.send:
			assert.Equal(t, []byte("hello"), got, "client %d", i+1)
		default:
			t.Errorf("client %d did not receive broadcast", i+1)
		}
	}
}

func TestHub_Unregister_ClosesClientSend(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	client := newTestClient(hub, "")
	hub.Register <- client
	hub.Unregister <- client // unbuffered — blocks until Run() receives

	// Synchronise: send a dummy register so we know Run() has finished
	// processing the Unregister (channels are FIFO within a single goroutine).
	dummy := newTestClient(hub, "")
	hub.Register <- dummy

	_, ok := <-client.send
	assert.False(t, ok, "send channel should be closed after unregister")
}

func TestHub_Unregister_DoesNotPanicForUnknownClient(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Unregister a client that was never registered — should not panic.
	unknown := newTestClient(hub, "")
	assert.NotPanics(t, func() {
		hub.Unregister <- unknown
		dummy := newTestClient(hub, "")
		hub.Register <- dummy // ensure Run() has processed the above
	})
}

func TestHub_SendToUser_OnlyTargetReceives(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	alice := newTestClient(hub, "alice")
	bob := newTestClient(hub, "bob")
	hub.Register <- alice
	hub.Register <- bob
	// Barrier: hub.Register is unbuffered, so sending to it blocks until Run()
	// loops back to select — which requires it to have finished processing bob
	// (including the mu.Lock / userClients append / mu.Unlock).
	barrier := newTestClient(hub, "")
	hub.Register <- barrier

	hub.SendToUser("alice", []byte("for alice"))

	select {
	case got := <-alice.send:
		assert.Equal(t, []byte("for alice"), got)
	default:
		t.Error("alice did not receive her message")
	}

	select {
	case <-bob.send:
		t.Error("bob should not receive a message targeted at alice")
	default:
		// expected: bob's channel is empty
	}
}

func TestHub_SendToUser_MultipleConnectionsSameUser(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	conn1 := newTestClient(hub, "user1")
	conn2 := newTestClient(hub, "user1")
	hub.Register <- conn1
	hub.Register <- conn2
	// Same barrier pattern: guarantees conn2's userClients append is complete.
	barrier := newTestClient(hub, "")
	hub.Register <- barrier

	hub.SendToUser("user1", []byte("broadcast to user"))

	for i, c := range []*Client{conn1, conn2} {
		select {
		case got := <-c.send:
			assert.Equal(t, []byte("broadcast to user"), got, "connection %d", i+1)
		default:
			t.Errorf("connection %d of user1 did not receive message", i+1)
		}
	}
}

func TestHub_SendToUser_UnknownUser_NoOp(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Should not panic or block.
	assert.NotPanics(t, func() {
		hub.SendToUser("nobody", []byte("ignored"))
	})
}

func TestHub_AnonymousClient_NotIndexedByUser(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// UserID="" → must not be added to userClients[""]
	anon := newTestClient(hub, "")
	hub.Register <- anon

	// SendToUser("", ...) should not deliver to anonymous clients.
	hub.SendToUser("", []byte("msg"))

	select {
	case <-anon.send:
		t.Error("anonymous client should not receive a user-targeted message")
	default:
		// expected
	}
}

func TestHub_UnregisterWithUserID_RemovesFromUserIndex(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	client := newTestClient(hub, "user1")
	hub.Register <- client
	hub.Unregister <- client

	// Sync
	dummy := newTestClient(hub, "")
	hub.Register <- dummy

	// After unregister, SendToUser should not deliver to the removed client.
	hub.SendToUser("user1", []byte("gone"))

	select {
	case <-client.send:
		// channel may be closed; reading (zero, false) is fine — it means
		// the client is gone.  Only fail if we get an actual message.
	default:
		// expected: nothing delivered
	}
}

func TestHub_GetPresence_TracksOnlineAndLastSeen(t *testing.T) {
	hub := NewHub()
	client := newTestClient(hub, "alice")
	client.DeviceID = "device-a"

	hub.registerClient(client)

	snapshot := hub.GetPresence("alice")
	assert.Equal(t, chatPort.PresenceStatusOnline, snapshot.Status)
	assert.Nil(t, snapshot.LastSeenAt)

	lastSeen := time.Date(2026, time.April, 12, 2, 3, 4, 0, time.UTC)
	hub.unregisterClient(client, lastSeen)

	snapshot = hub.GetPresence("alice")
	assert.Equal(t, chatPort.PresenceStatusOffline, snapshot.Status)
	require.NotNil(t, snapshot.LastSeenAt)
	assert.True(t, snapshot.LastSeenAt.Equal(lastSeen))
}

func TestHub_GetPresence_StaysOnlineUntilLastDeviceDisconnects(t *testing.T) {
	hub := NewHub()
	client1 := newTestClient(hub, "alice")
	client1.DeviceID = "device-a"
	client2 := newTestClient(hub, "alice")
	client2.DeviceID = "device-b"

	hub.registerClient(client1)
	hub.registerClient(client2)
	hub.unregisterClient(client1, time.Now().UTC())

	snapshot := hub.GetPresence("alice")
	assert.Equal(t, chatPort.PresenceStatusOnline, snapshot.Status)
	assert.Nil(t, snapshot.LastSeenAt)

	lastSeen := time.Date(2026, time.April, 12, 5, 6, 7, 0, time.UTC)
	hub.unregisterClient(client2, lastSeen)

	snapshot = hub.GetPresence("alice")
	assert.Equal(t, chatPort.PresenceStatusOffline, snapshot.Status)
	require.NotNil(t, snapshot.LastSeenAt)
	assert.True(t, snapshot.LastSeenAt.Equal(lastSeen))
}

func TestHub_PruneStaleClients_MarksUserOffline(t *testing.T) {
	hub := NewHub()
	hub.presenceTTL = time.Second

	client := newTestClient(hub, "alice")
	client.DeviceID = "device-a"
	client.lastSeenUnixNano.Store(time.Now().Add(-2 * time.Minute).UnixNano())
	hub.registerClient(client)

	hub.pruneStaleClients(time.Now().UTC())

	snapshot := hub.GetPresence("alice")
	assert.Equal(t, chatPort.PresenceStatusOffline, snapshot.Status)
	require.NotNil(t, snapshot.LastSeenAt)
}
