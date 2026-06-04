// Copyright 2025 Rahmad Afandi
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package health

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/uptrace/bun"
)

const (
	defaultLivez        = "/livez"
	defaultReadyz       = "/readyz"
	defaultCheckTimeout = 5 * time.Second
)

// NamedCheck is a readiness check with a name.
type NamedCheck struct {
	Name string
	Fn   func(ctx context.Context) error
}

// Check builds a NamedCheck.
func Check(name string, fn func(ctx context.Context) error) NamedCheck {
	return NamedCheck{Name: name, Fn: fn}
}

// PingBun is a convenience readiness check that pings a Bun database.
func PingBun(db *bun.DB) NamedCheck {
	return Check("db", func(ctx context.Context) error {
		return db.PingContext(ctx)
	})
}

// Register mounts /livez and /readyz on app.
func Register(app *fiber.App, checks ...NamedCheck) {
	RegisterAt(app, defaultLivez, defaultReadyz, checks...)
}

// RegisterAt mounts liveness and readiness endpoints at custom paths.
func RegisterAt(app *fiber.App, livez, readyz string, checks ...NamedCheck) {
	app.Get(livez, func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	app.Get(readyz, func(c *fiber.Ctx) error {
		ok, detail := runChecks(c.UserContext(), checks)
		status := "ok"
		code := fiber.StatusOK
		if !ok {
			status = "error"
			code = fiber.StatusServiceUnavailable
		}
		return c.Status(code).JSON(fiber.Map{"status": status, "checks": detail})
	})
}

func runChecks(ctx context.Context, checks []NamedCheck) (bool, map[string]string) {
	detail := make(map[string]string, len(checks))
	if len(checks) == 0 {
		return true, detail
	}

	type result struct {
		name string
		msg  string
		ok   bool
	}
	ch := make(chan result, len(checks))

	for _, chk := range checks {
		chk := chk
		go func() {
			cctx, cancel := context.WithTimeout(ctx, defaultCheckTimeout)
			defer cancel()

			r := result{name: chk.Name, ok: true, msg: "ok"}
			func() {
				defer func() {
					if rec := recover(); rec != nil {
						r.ok = false
						r.msg = fmt.Sprintf("panic: %v", rec)
					}
				}()
				if err := chk.Fn(cctx); err != nil {
					r.ok = false
					r.msg = err.Error()
				}
			}()
			ch <- r
		}()
	}

	allOK := true
	for i := 0; i < len(checks); i++ {
		r := <-ch
		detail[r.name] = r.msg
		if !r.ok {
			allOK = false
		}
	}
	return allOK, detail
}
