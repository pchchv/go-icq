package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"strings"
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
