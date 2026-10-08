package main

import (
	"disk-sync/src/internal/config"
	"fmt"
	"os"
)

func main() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load configuration: %v\n", err)
		return
	}

	fmt.Printf("Configuration loaded successfully, env: %s\n", cfg.Env)
}
