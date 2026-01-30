package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
)

func main() {
	fix := flag.Bool("fix", false, "reset processing messages to new")
	flag.Parse()

	connStr := "postgres://user:password@localhost:5433/wb_tech"
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(ctx)

	if *fix {
		tag, err := conn.Exec(ctx, "UPDATE outbox SET status = 'new' WHERE status = 'processing'")
		if err != nil {
			fmt.Printf("Fix failed: %v\n", err)
		} else {
			fmt.Printf("Fixed %d messages\n", tag.RowsAffected())
		}
	}

	fmt.Println("--- Orders ---")
	rows, _ := conn.Query(ctx, "SELECT id, status, updated_at FROM orders ORDER BY created_at DESC LIMIT 5")
	for rows.Next() {
		var id, status string
		var updatedAt interface{}
		rows.Scan(&id, &status, &updatedAt)
		fmt.Printf("ID: %s | Status: %s | Updated: %v\n", id, status, updatedAt)
	}

	fmt.Println("\n--- Outbox ---")
	rows, _ = conn.Query(ctx, "SELECT id, status, event_type FROM outbox ORDER BY created_at DESC LIMIT 5")
	for rows.Next() {
		var id, status, eventType string
		rows.Scan(&id, &status, &eventType)
		fmt.Printf("ID: %s | Status: %s | Type: %s\n", id, status, eventType)
	}
}
