package main

import (
	"log"
	"os"
	"runtime"
	"time"
	
	"fabric-gui/foundation"
)

func main() {
	// Configure logging
	log.SetFlags(log.Ltime | log.Lmicroseconds)
	
	// Display startup information
	log.Println("==== Fabric GUI Starting ====")
	log.Printf("OS: %s, Architecture: %s", runtime.GOOS, runtime.GOARCH)
	log.Printf("Go version: %s", runtime.Version())
	log.Printf("GOPATH: %s", os.Getenv("GOPATH"))
	log.Printf("Working directory: %s", getWD())
	
	// Check if we should skip patterns
	skipPatterns := os.Getenv("FABRIC_GUI_SKIP_PATTERNS") == "1"
	if skipPatterns {
		log.Println("Pattern loading disabled (FABRIC_GUI_SKIP_PATTERNS=1)")
	}
	
	startTime := time.Now()
	
	// Initialize the main application
	log.Println("Initializing application...")
	app, err := foundation.NewFabricApp()
	if err != nil {
		log.Fatalf("Failed to initialize Fabric GUI: %v", err)
	}
	
	initDuration := time.Since(startTime)
	log.Printf("Initialization completed in %v", initDuration)
	
	// Run the application
	log.Println("Starting GUI event loop...")
	app.Run()
	
	log.Println("==== Fabric GUI Exiting ====")
}

// getWD returns the current working directory or an error message
func getWD() string {
	dir, err := os.Getwd()
	if err != nil {
		return "Error getting working directory: " + err.Error()
	}
	return dir
}