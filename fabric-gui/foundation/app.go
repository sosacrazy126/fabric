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

	// Initialize the application with default state
	app := &FabricApp{
		window: win,
		state:  NewAppState(),
	}
	
	// Initialize Fabric paths
	fabricPaths, err := GetFabricPaths()
	if err != nil {
		log.Printf("Warning: Failed to get Fabric paths: %v", err)
		// We'll create fallbacks later
	}
	app.fabricPaths = fabricPaths
	
	// Initialize Fabric config
	fabricConfig := NewFabricConfig(fabricPaths)
	if err := fabricConfig.Initialize(); err != nil {
		log.Printf("Warning: Failed to initialize Fabric config: %v", err)
		// We'll continue with defaults
	}
	app.fabricConfig = fabricConfig
	
	// Load settings from config to state
	appState := fabricConfig.GetDefaultAppState()
	app.state = appState
	
	// Initialize execution manager
	app.execManager = NewExecutionManager(app, fabricConfig)

	// Initialize pattern loader
	app.patternLoader = NewPatternLoader(fabricPaths.PatternsDir, fabricPaths.DescriptionsPath)

	// Load patterns (unless skipped)
	if !skipPatternLoading {
		if err := app.loadPatterns(); err != nil {
			log.Printf("Warning: failed to load patterns: %v", err)
			// Continue anyway with an empty pattern list
			app.state.LoadedPatterns = []Pattern{}
			app.state.FilteredPatterns = []Pattern{}
		}
	} else {
		// Create a simple test pattern for the UI
		testPattern := Pattern{
			ID:          "test_pattern",
			Name:        "Test Pattern",
			Description: "A test pattern for demonstration",
			SystemMD:    "# Test Pattern\n\nThis is a test pattern.",
			UserMD:      "",
			Tags:        []string{"test"},
		}
		app.state.LoadedPatterns = []Pattern{testPattern}
		app.state.FilteredPatterns = []Pattern{testPattern}
	}

	// Load models and vendors
	if err := app.loadModelsAndVendors(); err != nil {
		log.Printf("Warning: failed to load models and vendors: %v", err)
		// Continue with defaults
		app.state.LoadedVendors = []string{app.state.CurrentVendorID}
		app.state.LoadedModels = map[string][]string{
			app.state.CurrentVendorID: {app.state.CurrentModelID},
		}
	}

	// Initialize UI components
	app.mainLayout = NewMainLayout(app)
	app.StatusBar = app.mainLayout.StatusBar // Store direct reference to status bar
	
	// Set main window content
	app.window.SetContent(app.mainLayout.Container())
	
	return app, nil
}

// loadPatterns loads all patterns using Fabric's database or the patternLoader
func (app *FabricApp) loadPatterns() error {
	log.Println("Loading patterns...")
	// Initialize a temporary status message if StatusBar isn't ready yet
	if app.StatusBar != nil {
		app.StatusBar.ShowMessage("Loading patterns...")
	}
	
	// Use a timeout mechanism to prevent hanging indefinitely
	patternsChan := make(chan []Pattern, 1)
	errChan := make(chan error, 1)
	
	go func() {
		// First try loading from Fabric's database
		if app.fabricConfig != nil {
			patterns, err := app.fabricConfig.LoadPatterns()
			if err == nil && len(patterns) > 0 {
				patternsChan <- patterns
				return
			}
			log.Printf("Warning: Failed to load patterns from Fabric DB: %v", err)
			log.Println("Falling back to direct filesystem loading")
		}
		
		// Fallback to direct filesystem loading
		patterns, err := app.patternLoader.LoadAllPatterns()
		if err != nil {
			errChan <- err
			return
		}
		patternsChan <- patterns
	}()
	
	// Wait for patterns or timeout
	select {
	case patterns := <-patternsChan:
		log.Printf("Loaded %d patterns successfully", len(patterns))
		
		// Sort patterns by name for better UX
		sort.Slice(patterns, func(i, j int) bool {
			return patterns[i].Name < patterns[j].Name
		})
		
		app.state.LoadedPatterns = patterns
		app.state.FilteredPatterns = make([]Pattern, len(patterns)) // Initialize filtered list
		copy(app.state.FilteredPatterns, patterns)
		
		// If we have no patterns, create a fallback test pattern
		if len(patterns) == 0 {
			log.Println("No patterns found, creating test pattern")
			testPattern := createTestPattern()
			app.state.LoadedPatterns = []Pattern{testPattern}
			app.state.FilteredPatterns = []Pattern{testPattern}
		}
		
		if app.StatusBar != nil {
			app.StatusBar.ShowMessage(fmt.Sprintf("Loaded %d patterns", len(app.state.LoadedPatterns)))
		}
		return nil
		
	case err := <-errChan:
		log.Printf("Error loading patterns: %v", err)
		if app.StatusBar != nil {
			app.StatusBar.ShowError(err)
		}
		return err
		
	case <-time.After(15 * time.Second):
		log.Println("Pattern loading timed out after 15 seconds")
		// Create a test pattern so the UI can still function
		testPattern := createTestPattern()
		app.state.LoadedPatterns = []Pattern{testPattern}
		app.state.FilteredPatterns = []Pattern{testPattern}
		
		if app.StatusBar != nil {
			app.StatusBar.ShowError(fmt.Errorf("pattern loading timed out"))
		}
		return fmt.Errorf("pattern loading timed out")
	}
}


// Run starts the application
func (app *FabricApp) Run() {
	app.window.Resize(fyne.NewSize(1200, 800))
	app.window.ShowAndRun()
	log.Println("Fabric GUI application event loop finished.")
}

// updateRunButtonText updates the text of the Run Pattern button in the Execute tab
func (app *FabricApp) updateRunButtonText(patternName string) {
	// This logic is now handled by MainContentPanel and update its RunButton
	app.mainLayout.MainContent.UpdateRunButton(patternName)
}

// getPatternNameByID safely retrieves the pattern's display name from LoadedPatterns.
func (app *FabricApp) getPatternNameByID(patternID string) string {
	for _, p := range app.state.LoadedPatterns {
		if p.ID == patternID {
			return p.Name
		}
	}
	return "" // Pattern not found
}

// Helper functions

// ShowMessage displays a message in the status bar
func (app *FabricApp) ShowMessage(message string) {
	if app.StatusBar != nil {
		app.StatusBar.ShowMessage(message)
	}
}

// ShowError displays an error message in the status bar
func (app *FabricApp) ShowError(err error) {
	if app.StatusBar != nil {
		app.StatusBar.ShowError(err)
	}
}

// StarOutput saves the current output as a favorite
func (app *FabricApp) StarOutput(customName string) {
	// Check if we have an output to star
	if app.state.LastOutput == "" {
		app.ShowMessage("No output to star")
		return
	}
	
	// Generate a unique ID for this starred output
	id := fmt.Sprintf("star_%d", len(app.state.StarredOutputs) + 1)
	
	// Get pattern information
	patternName := app.getPatternNameByID(app.state.CurrentPatternID)
	
	// Create the output snapshot
	starred := OutputSnapshot{
		ID:           id,
		PatternID:    app.state.CurrentPatternID,
		PatternName:  patternName,
		Timestamp:    time.Now(),
		InputText:    app.state.CurrentInputText,
		OutputText:   app.state.LastOutput,
		Model:        app.state.CurrentModelID,
		Vendor:       app.state.CurrentVendorID,
		CustomName:   customName,
	}
	
	// Add to starred outputs
	app.state.StarredOutputs = append(app.state.StarredOutputs, starred)
	
	app.ShowMessage(fmt.Sprintf("Output starred as '%s'", customName))
}

// executePattern is the public method for running a pattern
func (app *FabricApp) executePattern(config ExecutionConfig) (*ExecutionResult, error) {
	if app.execManager == nil {
		return nil, fmt.Errorf("execution manager not initialized")
	}
	return app.execManager.ExecutePattern(config)
}

// createTestPattern generates a hardcoded pattern for fallback.
func createTestPattern() Pattern {
	return Pattern{
		ID:          "test_pattern",
		Name:        "Test Pattern",
		Description: "A simple test pattern for demonstration purposes when no Fabric patterns are found.",
		SystemMD:    "# Test Pattern\n\nThis is a simulated test pattern. If you see this, Fabric patterns could not be loaded from your system.\n\n## STEPS\n1. Acknowledge the input.\n2. Provide a simulated response.\n\n## OUTPUT INSTRUCTIONS\nRespond concisely.",
		UserMD:      "",
		Tags:        []string{"test", "simulation", "fallback"},
	}
}

// loadModelsAndVendors loads available vendors (but not their models yet)
func (app *FabricApp) loadModelsAndVendors() error {
	log.Println("Loading AI vendors...")
	if app.StatusBar != nil {
		app.StatusBar.ShowMessage("Loading AI vendors...")
	}
	
	if app.fabricConfig == nil {
		return fmt.Errorf("fabric config not initialized")
	}
	
	// Only load vendors initially (not models)
	vendors, err := app.fabricConfig.LoadVendors()
	if err != nil {
		return fmt.Errorf("failed to load vendors: %w", err)
	}
	
	// Store in app state
	app.state.LoadedVendors = vendors
	app.state.LoadedModels = make(map[string][]string) // Initialize empty model map
	
	log.Printf("Loaded %d AI vendors", len(vendors))
	if app.StatusBar != nil {
		app.StatusBar.ShowMessage(fmt.Sprintf("Loaded %d AI vendors", len(vendors)))
	}
	
	return nil
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
		log.Printf("Warning: failed to load models for %s: %v", vendorName, err)
		// Set empty array for this vendor to avoid retrying constantly
		app.state.LoadedModels[vendorName] = []string{}
		app.state.VendorModelCounts[vendorName] = 0
		return fmt.Errorf("failed to load models: %w", err)
	}
	
	// Store in app state
	app.state.LoadedModels[vendorName] = models
	app.state.VendorModelCounts[vendorName] = len(models)
	
	log.Printf("Loaded %d models for %s", len(models), vendorName)
	if app.StatusBar != nil {
		app.StatusBar.ShowMessage(fmt.Sprintf("Loaded %d models for %s", len(models), vendorName))
	}
	
	return nil
}