package main

import (
	backendapp "backend-go/app"
	"log"
)

func main() {
	app, err := backendapp.New()
	if err != nil {
		log.Fatalf("failed to initialize backend app: %v", err)
	}
	defer func() { _ = app.Close() }()

	if err := app.Run(); err != nil {
		log.Fatalf("backend server failed: %v", err)
	}
}
