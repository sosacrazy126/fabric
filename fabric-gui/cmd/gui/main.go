package main

import (
	"log"
	"path/filepath"

	"fabric-gui/foundation"
)

func main() {
	app, err := foundation.NewFabricApp()
	if err != nil {
		log.Fatalf("Failed to initialize Fabric GUI: %v", err)
	}
	
	app.Run()
}