// Copyright 2026 Rahmad Afandi. MIT License.

package outbox_test

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"

	"github.com/rahmadafandi/fibr/outbox"
)

// recordingPublisher prints each event it would publish.
type recordingPublisher struct{}

func (recordingPublisher) Publish(_ context.Context, topic string, payload []byte) error {
	fmt.Printf("publish %s %s\n", topic, payload)
	return nil
}

// Write an event in the same transaction as the business data, then let the
// relay publish it. If the transaction rolls back, no event is ever published.
func ExampleRelay_Process() {
	sqldb, _ := sql.Open(sqliteshim.ShimName, "file::memory:?cache=shared")
	db := bun.NewDB(sqldb, sqlitedialect.New())
	defer db.Close()
	ctx := context.Background()
	_ = outbox.Migrate(ctx, db)

	// Business write + event, atomically.
	tx, _ := db.BeginTx(ctx, nil)
	_ = outbox.Enqueue(ctx, tx, "order.created", map[string]int{"order_id": 42})
	_ = tx.Commit()

	// Background relay publishes committed events at-least-once.
	relay := outbox.NewRelay(db, recordingPublisher{})
	n, _ := relay.Process(ctx)
	fmt.Println("published:", n)
	// Output:
	// publish order.created {"order_id":42}
	// published: 1
}
