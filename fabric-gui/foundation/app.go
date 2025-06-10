package foundation

import (
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

// FabricApp represents the main application structure
type FabricApp struct {
	// Core Components
	window        fyne.Window
	patternLoader *PatternLoader
	state         *AppState
	fabricPaths   *FabricPaths
	fabricConfig  *FabricConfig
	execManager   *ExecutionManager

	// UI Components
	mainLayout *MainLayout // The new main layout structure
	
	// Direct reference to status bar for easier access
	StatusBar    *StatusBar
}

// NewFabricApp creates and initializes the main application
func NewFabricApp() (*FabricApp, error) {
	// Initialize Fyne app
	a := app.New()
	win := a.NewWindow("Fabric GUI")
	
	// In a real implementation, set the application icon
	// This would be done with an imported resource from assets.go
	// a.SetIcon(appIcon)
	
	// Configure logging
	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Lshortfile)
	log.Println("==== Fabric GUI Starting ====")
	
	// Check if we should skip pattern loading (faster startup for testing)
	skipPatternLoading := os.Getenv("FABRIC_GUI_SKIP_PATTERNS") == "1"
	if skipPatternLoading {
		log.Println("FABRIC_GUI_SKIP_PATTERNS=1, skipping pattern loading")
	}

	// Create app instance
	fabricApp := &FabricApp{
		window: win,
		state:  NewAppState(),
	}
	
	// Initialize paths
	var err error
	fabricApp.fabricPaths, err = GetFabricPaths()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Fabric paths: %w", err)
	}
	
	log.Printf("Using config dir: %s", fabricApp.fabricPaths.ConfigDir)
	log.Printf("Using patterns dir: %s", fabricApp.fabricPaths.PatternsDir)
	
	// Initialize config
	fabricApp.fabricConfig = NewFabricConfig(fabricApp.fabricPaths)
	if err := fabricApp.fabricConfig.Initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize Fabric configuration: %w", err)
	}
	
	// Initialize state with config values
	fabricApp.state = fabricApp.fabricConfig.GetDefaultAppState()
	
	// Create main layout
	fabricApp.mainLayout = NewMainLayout(fabricApp)
	
	// Store reference to status bar for easier access
	fabricApp.StatusBar = fabricApp.mainLayout.StatusBar
	
	// Set content and configure window
	win.SetContent(fabricApp.mainLayout.Container())
	win.Resize(fyne.NewSize(1024, 768))
	win.SetMaster() // This is the main window
	
	// Load patterns if not skipped
	if !skipPatternLoading {
		go fabricApp.loadPatterns()
	}
	
	return fabricApp, nil
}

// Run starts the application main loop
func (app *FabricApp) Run() {
	app.window.ShowAndRun()
}

// ShowMessage displays a message in the status bar
func (app *FabricApp) ShowMessage(message string) {
	if app.StatusBar != nil {
		app.StatusBar.ShowMessage(message)
	}
}

// loadPatterns loads all patterns from Fabric's database
func (app *FabricApp) loadPatterns() {
	startTime := time.Now()
	app.ShowMessage("Loading patterns...")
	
	// Create a channel to signal completion
	done := make(chan bool)
	var patterns []Pattern
	var err error
	
	// Load patterns in a goroutine
	go func() {
		patterns, err = app.fabricConfig.LoadPatterns()
		done <- true
	}()
	
	// Wait for completion or timeout
	select {
	case <-done:
		if err != nil {
			log.Printf("Error loading patterns: %v", err)
			app.StatusBar.ShowError(err.Error())
			return
		}
		
		app.processLoadedPatterns(patterns, startTime)
		
	case <-time.After(30 * time.Second):
		log.Println("Pattern loading timed out after 30 seconds")
		app.StatusBar.ShowError("Pattern loading timed out")
	}
}

// processLoadedPatterns handles successfully loaded patterns
func (app *FabricApp) processLoadedPatterns(patterns []Pattern, startTime time.Time) {
	// Update app state
	app.state.LoadedPatterns = patterns
	app.state.FilteredPatterns = patterns // Initially, filtered = all
	
	// Sort patterns by name for better UX
	sort.Slice(app.state.LoadedPatterns, func(i, j int) bool {
		return app.state.LoadedPatterns[i].Name < app.state.LoadedPatterns[j].Name
	})
	
	// Update UI
	if app.mainLayout != nil && app.mainLayout.Sidebar != nil {
		app.mainLayout.Sidebar.patternList.Refresh()
	}
	
	// Update status
	loadTime := time.Since(startTime)
	app.ShowMessage(fmt.Sprintf("Loaded %d patterns in %v", len(patterns), loadTime.Round(time.Millisecond)))
	log.Printf("Loaded %d patterns in %v", len(patterns), loadTime.Round(time.Millisecond))
}

// getPatternNameByID returns the name of a pattern given its ID
func (app *FabricApp) getPatternNameByID(id string) string {
	for _, pattern := range app.state.LoadedPatterns {
		if pattern.ID == id {
			return pattern.Name
		}
	}
	return ""
}

// loadModelsForVendor loads models for a specific vendor on demand
func (app *FabricApp) loadModelsForVendor(vendorName string) error {
	log.Printf("Loading models for vendor: %s", vendorName)
	if app.StatusBar != nil {
		app.StatusBar.ShowMessage(fmt.Sprintf("Loading models for %s...", vendorName))
	}
	
	if app.fabricConfig == nil {
		return fmt.Errorf("fabric config not initialized")
	}
	
	// Check if already cached
	if models, ok := app.state.LoadedModels[vendorName]; ok && len(models) > 0 {
		log.Printf("Using cached models for %s (%d models)", vendorName, len(models))
		// Update the model count cache if not already set
		app.state.VendorModelCounts[vendorName] = len(models)
		return nil
	}
	
	// Load models for this vendor
	models, err := app.fabricConfig.LoadModelsForVendor(vendorName)
	if err != nil {
		log.Printf("Error loading models for vendor %s: %v", vendorName, err)
		return err
	}
	
	// Cache models in app state
	app.state.LoadedModels[vendorName] = models
	app.state.VendorModelCounts[vendorName] = len(models)
	
	log.Printf("Loaded %d models for vendor %s", len(models), vendorName)
	if app.StatusBar != nil {
		app.StatusBar.ShowMessage(fmt.Sprintf("Loaded %d models for %s", len(models), vendorName))
	}
	
	return nil
}

// ShowError displays an error message in the status bar
func (app *FabricApp) ShowError(err error) {
	log.Printf("Error: %v", err)
	if app.StatusBar != nil {
		app.StatusBar.ShowError(err.Error())
	}
}

// ShowErrorStr with string parameter for direct string errors
func (app *FabricApp) ShowErrorStr(message string) {
	log.Printf("Error: %s", message)
	if app.StatusBar != nil {
		app.StatusBar.ShowError(message)
	}
}
