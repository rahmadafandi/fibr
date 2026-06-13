// Copyright 2026 Rahmad Afandi. MIT License.

package ws_test

import (
	"net"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	gws "github.com/fasthttp/websocket"
	"github.com/gofiber/fiber/v2"
	fhredis "github.com/rahmadafandi/fibr/redis"
	"github.com/rahmadafandi/fibr/ws"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type chatMsg struct {
	Text string `json:"text"`
}

// serve starts app on a random port and returns ws:// base URL + a stop func.
func serve(t *testing.T, app *fiber.App) (string, func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	go func() { _ = app.Listener(ln) }()
	return "ws://" + ln.Addr().String(), func() { _ = app.Shutdown() }
}

func dial(t *testing.T, url string) *gws.Conn {
	t.Helper()
	var conn *gws.Conn
	var err error
	for i := 0; i < 50; i++ {
		conn, _, err = gws.DefaultDialer.Dial(url, nil)
		if err == nil {
			return conn
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("dial %s: %v", url, err)
	return nil
}

func readMsg(t *testing.T, c *gws.Conn) chatMsg {
	t.Helper()
	_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
	var m chatMsg
	require.NoError(t, c.ReadJSON(&m))
	return m
}

func TestRoomBroadcast(t *testing.T) {
	hub := ws.NewHub[chatMsg](ws.WithPingInterval(0))
	app := fiber.New()
	app.Get("/ws/:room", hub.Handle(ws.Handler[chatMsg]{
		OnConnect: func(c *ws.Conn[chatMsg]) error { c.Join(c.Params("room")); return nil },
	}))
	base, stop := serve(t, app)
	defer stop()

	a := dial(t, base+"/ws/r1")
	defer a.Close()
	b := dial(t, base+"/ws/r1")
	defer b.Close()
	other := dial(t, base+"/ws/r2")
	defer other.Close()

	require.Eventually(t, func() bool { return hub.Count() == 3 }, time.Second, 10*time.Millisecond)

	hub.ToRoom("r1", chatMsg{Text: "hi"})
	require.Equal(t, "hi", readMsg(t, a).Text)
	require.Equal(t, "hi", readMsg(t, b).Text)

	hub.Broadcast(chatMsg{Text: "all"})
	require.Equal(t, "all", readMsg(t, other).Text)
}

func TestOnMessageEcho(t *testing.T) {
	hub := ws.NewHub[chatMsg](ws.WithPingInterval(0))
	app := fiber.New()
	app.Get("/ws/:room", hub.Handle(ws.Handler[chatMsg]{
		OnConnect: func(c *ws.Conn[chatMsg]) error { c.Join(c.Params("room")); return nil },
		OnMessage: func(c *ws.Conn[chatMsg], m chatMsg) { hub.ToRoom(c.Params("room"), m) },
	}))
	base, stop := serve(t, app)
	defer stop()

	a := dial(t, base+"/ws/x")
	defer a.Close()
	require.Eventually(t, func() bool { return hub.Count() == 1 }, time.Second, 10*time.Millisecond)

	require.NoError(t, a.WriteJSON(chatMsg{Text: "ping"}))
	require.Equal(t, "ping", readMsg(t, a).Text)
}

func TestDisconnectRemoves(t *testing.T) {
	hub := ws.NewHub[chatMsg](ws.WithPingInterval(0))
	app := fiber.New()
	app.Get("/ws", hub.Handle(ws.Handler[chatMsg]{}))
	base, stop := serve(t, app)
	defer stop()

	a := dial(t, base+"/ws")
	require.Eventually(t, func() bool { return hub.Count() == 1 }, time.Second, 10*time.Millisecond)
	a.Close()
	require.Eventually(t, func() bool { return hub.Count() == 0 }, 2*time.Second, 10*time.Millisecond)
}

func TestUpgradeRequired(t *testing.T) {
	hub := ws.NewHub[chatMsg]()
	app := fiber.New()
	app.Get("/ws", hub.Handle(ws.Handler[chatMsg]{}))
	resp, err := app.Test(httptest.NewRequest("GET", "/ws", nil))
	require.NoError(t, err)
	require.Equal(t, fiber.StatusUpgradeRequired, resp.StatusCode)
}

func TestRedisBackplane(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()
	newR := func() *fhredis.Redis {
		return fhredis.New(goredis.NewClient(&goredis.Options{Addr: mr.Addr()}))
	}

	hubA := ws.NewHub[chatMsg](ws.WithPingInterval(0), ws.WithRedis(newR(), "rt"))
	defer hubA.Close()
	hubB := ws.NewHub[chatMsg](ws.WithPingInterval(0), ws.WithRedis(newR(), "rt"))
	defer hubB.Close()

	appB := fiber.New()
	appB.Get("/ws", hubB.Handle(ws.Handler[chatMsg]{}))
	base, stop := serve(t, appB)
	defer stop()

	client := dial(t, base+"/ws")
	defer client.Close()
	require.Eventually(t, func() bool { return hubB.Count() == 1 }, time.Second, 10*time.Millisecond)

	hubA.Broadcast(chatMsg{Text: "cross"})
	require.Equal(t, "cross", readMsg(t, client).Text)
}

func TestBackplaneErrSurfaced(t *testing.T) {
	// Point at a dead address so Subscribe's initial Receive fails fast.
	r := fhredis.New(goredis.NewClient(&goredis.Options{Addr: "127.0.0.1:1"}))
	hub := ws.NewHub[chatMsg](ws.WithPingInterval(0), ws.WithRedis(r, "rt"))
	defer hub.Close()
	require.Eventually(t, func() bool { return hub.BackplaneErr() != nil }, 3*time.Second, 20*time.Millisecond)
}

func TestBackplaneErrNilWhenHealthy(t *testing.T) {
	hub := ws.NewHub[chatMsg]() // no backplane configured
	require.NoError(t, hub.BackplaneErr())
}
