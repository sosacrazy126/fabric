package foundation

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"

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
	
	// In a real implementation, set the application icon
	// This would be done with an imported resource from assets.go
	// a.SetIcon(appIcon)
	
	// Check if we should skip pattern loading (faster startup for testing)
	skipPatternLoading := os.Getenv("FABRIC_GUI_SKIP_PATTERNS") == "1"
	if skipPatternLoading {
		log.Println("FABRIC_GUI_SKIP_PATTERNS=1, skipping pattern loading")
	}

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

	// Load patterns (unless skipped)
	if !skipPatternLoading {
		if err := app.loadPatterns(); err != nil {
			log.Printf("Warning: failed to load patterns: %v", err)
			// Continue anyway with an empty pattern list
			app.state.LoadedPatterns = []Pattern{}
		}
	} else {
		// Create a simple test pattern for the UI
		app.state.LoadedPatterns = []Pattern{
			{
				ID:          "test_pattern",
				Name:        "Test Pattern",
				Description: "A test pattern for demonstration",
				SystemMD:    "# Test Pattern\n\nThis is a test pattern.",
				UserMD:      "",
				Tags:        []string{"test"},
			},
		}
	}

	// Initialize UI components
	app.setupUI()
	return app, nil
}

// loadPatterns loads all patterns using the patternLoader
func (app *FabricApp) loadPatterns() error {
	log.Println("Loading patterns...")
	
	// Use a timeout mechanism to prevent hanging indefinitely
	patternsChan := make(chan []Pattern, 1)
	errChan := make(chan error, 1)
	
	go func() {
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
		return nil
		
	case err := <-errChan:
		log.Printf("Error loading patterns: %v", err)
		return err
		
	case <-time.After(10 * time.Second):
		log.Println("Pattern loading timed out after 10 seconds")
		// Create an empty pattern list so the UI can still load
		app.state.LoadedPatterns = []Pattern{}
		return fmt.Errorf("pattern loading timed out")
	}
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
	
	// Set up tab change handler for clearing input/output
	app.mainTabs.OnChanged = func(tab *container.TabItem) {
		// When switching to Execute tab, update pattern info and run button
		if tab.Text == "Execute" && app.state.CurrentPatternID != "" {
			// Find selected pattern to get name
			for _, p := range app.state.LoadedPatterns {
				if p.ID == app.state.CurrentPatternID {
					app.updateRunButtonText(p.Name)
					app.execute.patternInfo.SetText(fmt.Sprintf("Selected pattern: %s", p.Name))
					break
				}
			}
		}
	}

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

// updateRunButtonText updates the text of the Run Pattern button in the Execute tab
func (app *FabricApp) updateRunButtonText(patternName string) {
	if app.execute != nil && app.execute.runBtn != nil {
		if patternName == "" {
			app.execute.runBtn.SetText("Run Pattern")
			app.execute.runBtn.Importance = widget.MediumImportance
		} else {
			app.execute.runBtn.SetText(fmt.Sprintf("Run '%s'", patternName))
			app.execute.runBtn.Importance = widget.HighImportance
		}
		app.execute.runBtn.Refresh()
	}
}

// Helper functions

// getFabricDataDir returns the location of Fabric's data directory
func getFabricDataDir() (string, error) {
	log.Println("Searching for Fabric data directory...")
	
	// First, check if we're running from within the Fabric repository
	currentDir, err := os.Getwd()
	if err != nil {
		log.Printf("Error getting current directory: %v", err)
		return "", err
	}
	log.Printf("Current directory: %s", currentDir)

	// First check current directory
	patternsDir := filepath.Join(currentDir, "patterns")
	if _, err := os.Stat(patternsDir); err == nil {
		log.Printf("Found patterns directory at: %s", patternsDir)
		return currentDir, nil
	}
	log.Printf("No patterns directory found at: %s", patternsDir)
	
	// Check parent directory (we might be in a subdirectory of the fabric repo)
	parentDir := filepath.Dir(currentDir)
	parentPatternsDir := filepath.Join(parentDir, "patterns")
	if _, err := os.Stat(parentPatternsDir); err == nil {
		log.Printf("Found patterns directory at: %s", parentPatternsDir)
		return parentDir, nil
	}
	log.Printf("No patterns directory found at: %s", parentPatternsDir)

	// Check ~/.config/fabric
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Error getting home directory: %v", err)
		return "", fmt.Errorf("couldn't determine user home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config", "fabric")
	if _, err := os.Stat(configDir); err == nil {
		log.Printf("Found config directory at: %s", configDir)
		return configDir, nil
	}
	log.Printf("No config directory found at: %s", configDir)

	// As a fallback, create a minimal patterns directory in the current location
	log.Println("No Fabric data directory found, creating a minimal one in current directory")
	err = os.MkdirAll(patternsDir, 0755)
	if err != nil {
		log.Printf("Error creating patterns directory: %v", err)
		return "", fmt.Errorf("couldn't create patterns directory: %w", err)
	}
	
	// Create a simple test pattern
	testPatternDir := filepath.Join(patternsDir, "test_pattern")
	err = os.MkdirAll(testPatternDir, 0755)
	if err != nil {
		log.Printf("Error creating test pattern directory: %v", err)
		return "", fmt.Errorf("couldn't create test pattern directory: %w", err)
	}
	
	// Create a simple system.md file
	systemMDPath := filepath.Join(testPatternDir, "system.md")
	err = os.WriteFile(systemMDPath, []byte("# Test Pattern\n\nThis is a test pattern created by the Fabric GUI."), 0644)
	if err != nil {
		log.Printf("Error creating system.md: %v", err)
		return "", fmt.Errorf("couldn't create system.md: %w", err)
	}
	
	log.Println("Created minimal Fabric data directory with test pattern")
	return currentDir, nil
}