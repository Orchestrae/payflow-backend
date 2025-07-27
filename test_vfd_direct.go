package main

import (
	"context"
	"fmt"
	"log"
	"payflow/internal/platform/vfd"
	"time"
)

func main() {
	fmt.Println("=== Testing VFD Service Directly ===")

	// Create VFD client with the same credentials from .env
	vfdClient := vfd.NewClient(
		"https://api-devapps.vfdbank.systems",
		"xMjvHf7sY2q3YAxpPcDf8NrPzs8V",
		"5oUzlYqTOle35zY4gzs27k9TEvZY",
	)

	// Create VFD service
	vfdService := vfd.NewVFDService(vfdClient)

	// Test data from our business registration
	details := vfd.NewAccountDetails{
		RCNumber:          "RC1234567",
		CompanyName:       "TechCorp Solutions Limited",
		IncorporationDate: time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC),
		DirectorBVN:       "12345678901",
	}

	fmt.Printf("Testing with details: %+v\n", details)

	// Call the VFD service
	ctx := context.Background()
	account, err := vfdService.CreateNewCorporateAccount(ctx, details)

	if err != nil {
		log.Printf("VFD service error: %v", err)
		return
	}

	fmt.Printf("Success! Account created: %+v\n", account)
}
