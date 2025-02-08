package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/hashicorp/vault/api"
)

func init() {
	// Set up log
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)
}

func checkVaultConnectivity() error {
	log.Println("Checking Vault connectivity...")
	client, err := api.NewClient(&api.Config{Address: vaultAddr})
	if err != nil {
		return fmt.Errorf("failed to create Vault client: %v", err)
	}

	client.SetToken(vaultToken)

	// Check Vault health
	health, err := client.Sys().Health()
	if err != nil {
		return fmt.Errorf("failed to check Vault health: %v", err)
	}

	if !health.Initialized {
		return fmt.Errorf("vault is not initialized")
	}

	if health.Sealed {
		return fmt.Errorf("vault is sealed")
	}

	log.Println("Vault connectivity check passed")
	return nil
}

func main() {
	// Check Vault connectivity
	for i := 0; i < 5; i++ {
		if err := checkVaultConnectivity(); err != nil {
			log.Printf("Vault connectivity check failed: %v. Retrying in 5 seconds...\n", err)
			time.Sleep(5 * time.Second)
		} else {
			break
		}
	}

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/wrap", wrapHandler)
	http.HandleFunc("/unwrap", unwrapHandler)

	// Serve static files
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	log.Println("Server starting on :3001")
	if err := http.ListenAndServe(":3001", nil); err != nil {
		log.Fatal(err)
	}
}
