package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/google/uuid"
	"github.com/pchchv/go-icq/state"
)

const keyLength = 32 // 256 bits of entropy

func handleGenerate(args []string) {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing arguments: %v\n", err)
		os.Exit(1)
	}

	appName := fs.String("app-name", "", "Application name (required)")
	if *appName == "" {
		fmt.Fprintln(os.Stderr, "Error: --app-name is required")
		os.Exit(1)
	}

	// parse origins and capabilities
	var origins []string
	if originsStr := fs.String("origins", "", "Comma-separated list of allowed origins"); *originsStr != "" {
		origins = parseCSV(*originsStr)
	}

	var capabilities []string
	if capabilitiesStr := fs.String("capabilities", "", "Comma-separated list of capabilities"); *capabilitiesStr != "" {
		capabilities = parseCSV(*capabilitiesStr)
	}

	// generate secure random key
	keyBytes := make([]byte, keyLength)
	if _, err := rand.Read(keyBytes); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating key: %v\n", err)
		os.Exit(1)
	}

	devKey := hex.EncodeToString(keyBytes)
	// generate dev_id
	devID := fmt.Sprintf("dev_%s", uuid.New().String())
	rateLimit := fs.Int("rate-limit", 60, "Requests per minute")
	// create the API key record
	apiKey := state.WebAPIKey{
		DevID:          devID,
		DevKey:         devKey,
		AppName:        *appName,
		CreatedAt:      time.Now(),
		IsActive:       true,
		RateLimit:      *rateLimit,
		AllowedOrigins: origins,
		Capabilities:   capabilities,
	}

	// connect to database and insert the key
	store, err := connectToStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to database: %v\n", err)
		os.Exit(1)
	}

	if err := store.CreateAPIKey(context.Background(), apiKey); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating API key: %v\n", err)
		os.Exit(1)
	}

	// output the generated key details
	fmt.Println("Successfully generated Web API key:")
	fmt.Println("=====================================")
	fmt.Printf("Developer ID:  %s\n", devID)
	fmt.Printf("API Key:       %s\n", devKey)
	fmt.Printf("App Name:      %s\n", *appName)
	fmt.Printf("Rate Limit:    %d requests/minute\n", *rateLimit)
	if len(origins) > 0 {
		fmt.Printf("Origins:       %s\n", strings.Join(origins, ", "))
	}

	if len(capabilities) > 0 {
		fmt.Printf("Capabilities:  %s\n", strings.Join(capabilities, ", "))
	}

	fmt.Println("=====================================")
	fmt.Println("\nIMPORTANT: Save the API key securely. It cannot be retrieved later.")
}

func handleList(args []string) {
	store, err := connectToStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to database: %v\n", err)
		os.Exit(1)
	}

	keys, err := store.ListAPIKeys(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing API keys: %v\n", err)
		os.Exit(1)
	} else if len(keys) == 0 {
		fmt.Println("No API keys found.")
		return
	}

	// create a tabwriter for formatted output
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "DEV ID\tAPP NAME\tACTIVE\tRATE LIMIT\tCREATED\tLAST USED")
	fmt.Fprintln(w, "------\t--------\t------\t----------\t-------\t---------")

	for _, key := range keys {
		lastUsed := "Never"
		if key.LastUsed != nil {
			lastUsed = key.LastUsed.Format("2006-01-02 15:04")
		}

		fmt.Fprintf(w, "%s\t%s\t%v\t%d/min\t%s\t%s\n",
			truncateString(key.DevID, 20),
			truncateString(key.AppName, 20),
			key.IsActive,
			key.RateLimit,
			key.CreatedAt.Format("2006-01-02"),
			lastUsed,
		)
	}

	w.Flush()
}

func handleRevoke(args []string) {
	fs := flag.NewFlagSet("revoke", flag.ExitOnError)
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing arguments: %v\n", err)
		os.Exit(1)
	}

	devID := fs.String("dev-id", "", "Developer ID to revoke (required)")
	if *devID == "" {
		fmt.Fprintln(os.Stderr, "Error: --dev-id is required")
		os.Exit(1)
	}

	store, err := connectToStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to database: %v\n", err)
		os.Exit(1)
	}

	isActive := false
	update := state.WebAPIKeyUpdate{
		IsActive: &isActive,
	}
	err = store.UpdateAPIKey(context.Background(), *devID, update)
	switch err {
	case nil:
		fmt.Printf("Successfully revoked API key: %s\n", *devID)
	case state.ErrNoAPIKey:
		fmt.Fprintf(os.Stderr, "Error: API key not found for dev_id: %s\n", *devID)
		os.Exit(1)
	default:
		fmt.Fprintf(os.Stderr, "Error revoking API key: %v\n", err)
		os.Exit(1)
	}
}

func parseCSV(input string) []string {
	if input == "" {
		return []string{}
	}

	parts := strings.Split(input, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}

func connectToStore() (*state.SQLiteUserStore, error) {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "oscar.sqlite"
	}

	return state.NewSQLiteUserStore(dbPath)
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	return s[:maxLen-3] + "..."
}
