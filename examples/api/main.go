// Copyright 2026 Rahmad Afandi. MIT License.

// Command api shows several fibr packages working together in one small API:
// API-key auth, feature flags, an in-memory cache, an in-process event bus, and
// an audit log — all dependency-free (in-memory SQLite, in-process).
//
//	go run ./api
//	curl -H "X-API-Key: <printed-at-startup>" localhost:3000/widgets/1
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/rahmadafandi/fibr/apierror"
	"github.com/rahmadafandi/fibr/apikey"
	"github.com/rahmadafandi/fibr/audit"
	"github.com/rahmadafandi/fibr/bootstrap"
	"github.com/rahmadafandi/fibr/cache"
	"github.com/rahmadafandi/fibr/database"
	"github.com/rahmadafandi/fibr/events"
	"github.com/rahmadafandi/fibr/featureflag"
	"github.com/rahmadafandi/fibr/health"
	"github.com/rahmadafandi/fibr/response"
)

// Widget is the "expensive to load" resource the cache fronts.
type Widget struct {
	ID   string
	Name string
}

// WidgetViewed is a domain event published when a widget is read.
type WidgetViewed struct {
	WidgetID string
	Actor    string
}

func main() {
	ctx := context.Background()

	db, err := database.NewBun("file::memory:?cache=shared")
	if err != nil {
		log.Fatal(err)
	}
	if err := audit.Migrate(ctx, db); err != nil {
		log.Fatal(err)
	}

	// Issue a demo API key (in real life you persist only the hash).
	key, hash, err := apikey.Generate()
	if err != nil {
		log.Fatal(err)
	}
	keys := apikey.New(apikey.Config{Store: apikey.MapStore(map[string]apikey.Identity{
		hash: {ID: "demo-user", Scopes: []string{"widgets:read"}},
	})})

	actorOf := func(c *fiber.Ctx) string {
		if id, ok := apikey.FromContext(c); ok {
			return id.ID
		}
		return ""
	}

	flags := featureflag.New(featureflag.Rules(map[string]featureflag.Rule{
		"fancy_widget_name": {Enabled: true},
	}))

	rec := audit.New(audit.NewBunSink(db), audit.WithActor(actorOf))

	widgets := cache.New[Widget](cache.WithDefaultTTL(time.Minute), cache.WithMaxSize(1024))

	bus := events.New()
	events.Subscribe(bus, func(_ context.Context, e WidgetViewed) error {
		log.Printf("event: widget %s viewed by %s", e.WidgetID, e.Actor)
		return nil
	})

	app := bootstrap.New(bootstrap.Options{
		DB:           db,
		EnableCORS:   true,
		HealthChecks: []health.NamedCheck{health.PingBun(db)},
	})

	app.Get("/public", func(c *fiber.Ctx) error {
		return response.SendSuccess(c, fiber.Map{"ok": true}, "public")
	})

	// Everything below requires a valid API key.
	api := app.Group("/", keys.Middleware(), flags.Middleware(func(c *fiber.Ctx) featureflag.Eval {
		return featureflag.Eval{UserID: actorOf(c)}
	}))

	api.Get("/me", func(c *fiber.Ctx) error {
		id, _ := apikey.FromContext(c)
		return response.SendSuccess(c, id, "identity")
	})

	api.Get("/widgets/:id", keys.RequireScope("widgets:read"), func(c *fiber.Ctx) error {
		id := c.Params("id")

		w, err := widgets.GetOrLoad(c.UserContext(), "widget:"+id, func() (Widget, error) {
			// Pretend this is a slow DB/API call.
			name := "widget-" + id
			if featureflag.Enabled(c, "fancy_widget_name") {
				name = "✨ " + name
			}
			return Widget{ID: id, Name: name}, nil
		})
		if err != nil {
			return apierror.Internal("load widget")
		}

		// Fan out a domain event and record an audit entry.
		_ = events.Publish(c.UserContext(), bus, WidgetViewed{WidgetID: id, Actor: actorOf(c)})
		e := rec.FromRequest(c)
		e.Action, e.Target, e.TargetID = "widget.view", "widget", id
		_ = rec.Record(c.UserContext(), e)

		return response.SendSuccess(c, w, "widget")
	})

	fmt.Println("demo API key:", key)
	fmt.Println("listening on :3000  (try: curl -H \"X-API-Key: <key>\" localhost:3000/widgets/1)")
	if err := app.Run(":3000"); err != nil {
		log.Fatal(err)
	}
}
