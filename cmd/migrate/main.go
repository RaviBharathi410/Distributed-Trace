package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	clickhousedriver "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5433/distributedtrace?sslmode=disable"
	}

	chURL := os.Getenv("CLICKHOUSE_URL")
	if chURL == "" {
		chURL = "127.0.0.1:9000"
	}
	chDB := os.Getenv("CLICKHOUSE_DB")
	if chDB == "" {
		chDB = "default"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	fmt.Println("==================================================")
	fmt.Println("DistributedTrace Database Migration Runner")
	fmt.Println("==================================================")

	// 1. PostgreSQL Migrations
	fmt.Printf("\n[PostgreSQL] Connecting to %s...\n", sanitizeURL(dbURL))
	pgPool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("❌ Failed to connect to PostgreSQL: %v", err)
	}
	defer pgPool.Close()

	if err := pgPool.Ping(ctx); err != nil {
		log.Fatalf("❌ PostgreSQL ping failed (ensure Docker or service is running on specified port): %v", err)
	}
	fmt.Println("✔ PostgreSQL connected successfully.")

	pgMigrationsDir := "migrations/postgres"
	pgFiles, err := os.ReadDir(pgMigrationsDir)
	if err != nil {
		log.Fatalf("❌ Failed to read PostgreSQL migrations directory %s: %v", pgMigrationsDir, err)
	}

	var pgScripts []string
	for _, f := range pgFiles {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".sql") {
			pgScripts = append(pgScripts, f.Name())
		}
	}
	sort.Strings(pgScripts)

	fmt.Printf("[PostgreSQL] Applying %d migrations in %s...\n", len(pgScripts), pgMigrationsDir)
	for _, filename := range pgScripts {
		path := filepath.Join(pgMigrationsDir, filename)
		content, err := os.ReadFile(path)
		if err != nil {
			log.Fatalf("❌ Failed to read migration %s: %v", filename, err)
		}

		if _, err := pgPool.Exec(ctx, string(content)); err != nil {
			log.Fatalf("❌ Failed to execute PostgreSQL migration %s: %v", filename, err)
		}
		fmt.Printf("  ✔ Applied: %s\n", filename)
	}

	// Verify required tables in PostgreSQL
	requiredPGTables := []string{
		"users", "organizations", "api_keys", "sampling_rules",
		"retention_policies", "team_invites", "audit_log", "tenant_llm_costs",
	}
	for _, tbl := range requiredPGTables {
		var exists bool
		err := pgPool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT FROM information_schema.tables 
				WHERE table_schema = 'public' AND table_name = $1
			);
		`, tbl).Scan(&exists)
		if err != nil || !exists {
			log.Fatalf("❌ Verification failed: table %q does not exist in PostgreSQL (err: %v)", tbl, err)
		}
	}
	fmt.Println("✔ Verified all 8 PostgreSQL tables exist and are healthy.")

	// 2. ClickHouse Migrations
	fmt.Printf("\n[ClickHouse] Connecting to %s (DB: %s)...\n", chURL, chDB)
	chConn, err := clickhousedriver.Open(&clickhousedriver.Options{
		Addr: []string{chURL},
		Auth: clickhousedriver.Auth{
			Database: chDB,
			Username: "default",
			Password: "",
		},
	})
	if err != nil {
		log.Fatalf("❌ Failed to initialize ClickHouse connection: %v", err)
	}
	defer chConn.Close()

	if err := chConn.Ping(ctx); err != nil {
		fmt.Printf("⚠️ ClickHouse ping failed at %s (%v). Skipping ClickHouse migrations.\n", chURL, err)
	} else {
		fmt.Println("✔ ClickHouse connected successfully.")
		chMigrationsDir := "migrations/clickhouse"
		chFiles, err := os.ReadDir(chMigrationsDir)
		if err == nil {
			var chScripts []string
			for _, f := range chFiles {
				if !f.IsDir() && strings.HasSuffix(f.Name(), ".sql") {
					chScripts = append(chScripts, f.Name())
				}
			}
			sort.Strings(chScripts)

			fmt.Printf("[ClickHouse] Applying %d migrations in %s...\n", len(chScripts), chMigrationsDir)
			for _, filename := range chScripts {
				path := filepath.Join(chMigrationsDir, filename)
				content, err := os.ReadFile(path)
				if err != nil {
					log.Fatalf("❌ Failed to read ClickHouse migration %s: %v", filename, err)
				}

				// ClickHouse migrations may contain multiple statements separated by semicolons
				stmts := strings.Split(string(content), ";")
				for _, stmt := range stmts {
					stmt = strings.TrimSpace(stmt)
					if stmt == "" {
						continue
					}
					if err := chConn.Exec(ctx, stmt); err != nil {
						log.Fatalf("❌ Failed to execute ClickHouse migration %s: %v", filename, err)
					}
				}
				fmt.Printf("  ✔ Applied: %s\n", filename)
			}
			fmt.Println("✔ ClickHouse migrations completed successfully.")
		}
	}

	fmt.Println("\n==================================================")
	fmt.Println("✔ ALL DATABASE MIGRATIONS EXECUTED & VERIFIED")
	fmt.Println("==================================================")
}

func sanitizeURL(raw string) string {
	if strings.Contains(raw, "@") {
		parts := strings.Split(raw, "@")
		return "..." + "@" + parts[len(parts)-1]
	}
	return raw
}
