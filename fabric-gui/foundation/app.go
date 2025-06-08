package foundation

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// FabricApp represents the main application structure
type FabricApp struct {
	// Core Components
	window        fyne.Window
	patternLoader *PatternLoader
	state         *AppState

	// UI Components
	mainTabs      *container.AppTabs
	patterns      *PatternsPanel
	execute       *ExecutePanel
	status        *StatusBar
}

// AppState manages centralized application state
type AppState struct {
	CurrentPatternID string
	CurrentInput     string
	LastOutput       string
	LoadedPatterns   []Pattern
}

// NewFabricApp creates and initializes the main application
func NewFabricApp() (*FabricApp, error) {
	// Initialize Fyne app
	a := app.New()
	win := a.NewWindow("Fabric GUI")

	// Locate Fabric data directory
	fabricDataDir, err := getFabricDataDir()
	if err != nil {
		return nil, fmt.Errorf("failed to locate Fabric data directory: %w", err)
	}

	// Initialize pattern loader
	patternsDir := filepath.Join(fabricDataDir, "patterns")
	descriptionsPath := filepath.Join(fabricDataDir, "Pattern_Descriptions", "pattern_descriptions.json")
	patternLoader := NewPatternLoader(patternsDir, descriptionsPath)

	app := &FabricApp{
		window:        win,
		patternLoader: patternLoader,
		state:         &AppState{},
	}

	// Load patterns
	if err := app.loadPatterns(); err != nil {
		return nil, fmt.Errorf("failed to load patterns: %w", err)
	}

	// Initialize UI components
	app.setupUI()
	return app, nil
}

// loadPatterns loads all patterns using the patternLoader
func (app *FabricApp) loadPatterns() error {
	patterns, err := app.patternLoader.LoadAllPatterns()
	if err != nil {
		return err
	}

	// Sort patterns by name for better UX
	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].Name < patterns[j].Name
	})

	app.state.LoadedPatterns = patterns
	return nil
}

// setupUI initializes all UI components
func (app *FabricApp) setupUI() {
	// Create main components
	app.patterns = NewPatternsPanel(app)
	app.execute = NewExecutePanel(app)
	app.status = NewStatusBar()

	// Create tabs
	app.mainTabs = container.NewAppTabs(
		container.NewTabItem("Patterns", app.patterns.Container()),
		container.NewTabItem("Execute", app.execute.Container()),
	)

	// Create main layout
	mainContent := container.NewBorder(
		nil,                // top
		app.status.content, // bottom
		nil, nil,           // left, right
		app.mainTabs,       // center
	)

	app.window.SetContent(mainContent)
}

// Run starts the application
func (app *FabricApp) Run() {
	app.window.Resize(fyne.NewSize(1200, 800))
	app.window.ShowAndRun()
}

// Helper functions

// getFabricDataDir returns the location of Fabric's data directory
func getFabricDataDir() (string, error) {
	// First, check if we're running from within the Fabric repository
	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// If we're inside the Fabric repo, use the patterns directory directly
	patternsDir := filepath.Join(currentDir, "patterns")
	if _, err := os.Stat(patternsDir); err == nil {
		log.Println("Using local Fabric data directory")
		return currentDir, nil
	}

	// Otherwise, check ~/.config/fabric
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("couldn't determine user home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config", "fabric")
	if _, err := os.Stat(configDir); err == nil {
		log.Println("Using ~/.config/fabric data directory")
		return configDir, nil
	}

	return "", fmt.Errorf("couldn't locate Fabric data directory")
}