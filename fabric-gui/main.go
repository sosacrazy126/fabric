package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// Pattern represents a Fabric pattern for GUI display and management
type Pattern struct {
	ID          string   // Unique identifier (folder name)
	Name        string   // Display name (may be formatted version of ID)
	Description string   // Short description from pattern_descriptions.json
	SystemMD    string   // Content of system.md
	UserMD      string   // Content of user.md (if exists)
	Tags        []string // For filtering and categorization
}

// PatternDescription matches the structure in pattern_descriptions.json
type PatternDescription struct {
	PatternName string   `json:"patternName"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

// PatternDescriptionsFile represents the structure of pattern_descriptions.json
type PatternDescriptionsFile struct {
	Patterns []PatternDescription `json:"patterns"`
}

// AppState manages centralized application state
type AppState struct {
	CurrentPatternID string
	CurrentInput     string
	LastOutput       string
	LoadedPatterns   []Pattern
}

func main() {
	// Initialize Fyne app
	a := app.New()
	win := a.NewWindow("Fabric GUI")

	// Initialize app state
	state := &AppState{}

	// Load patterns
	patterns, err := loadPatterns()
	if err != nil {
		log.Printf("Failed to load patterns: %v", err)
	}
	state.LoadedPatterns = patterns

	// Create UI components
	statusBar := widget.NewLabel("Ready")
	
	// Create pattern list
	patternList := widget.NewList(
		func() int { return len(state.LoadedPatterns) },
		func() fyne.CanvasObject { 
			return container.NewVBox(
				widget.NewLabel("Pattern Name"),
				widget.NewLabel("Description"),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id >= len(state.LoadedPatterns) {
				return
			}
			container := obj.(*fyne.Container)
			nameLabel := container.Objects[0].(*widget.Label)
			descLabel := container.Objects[1].(*widget.Label)
			
			pattern := state.LoadedPatterns[id]
			nameLabel.SetText(pattern.Name)
			
			// Truncate description if needed
			desc := pattern.Description
			if len(desc) > 60 {
				desc = desc[:57] + "..."
			}
			descLabel.SetText(desc)
		},
	)

	// Create pattern details view
	systemText := widget.NewTextArea()
	systemText.Disable()
	
	userText := widget.NewTextArea()
	userText.Disable()
	
	// Tabs for system.md and user.md
	detailsTabs := container.NewAppTabs(
		container.NewTabItem("System Prompt", systemText),
		container.NewTabItem("User Prompt", userText),
	)

	// Create execute panel
	patternInfoLabel := widget.NewLabel("No pattern selected")
	inputArea := widget.NewMultiLineEntry()
	inputArea.SetPlaceHolder("Enter your input here...")
	
	outputArea := widget.NewTextArea()
	outputArea.Disable()
	
	executeBtn := widget.NewButton("Execute Pattern", func() {
		if state.CurrentPatternID == "" {
			statusBar.SetText("Error: No pattern selected")
			return
		}

		input := inputArea.Text
		if input == "" {
			statusBar.SetText("Error: No input provided")
			return
		}

		// Placeholder for actual execution
		// In a real implementation, this would use Fabric's core components
		go func() {
			statusBar.SetText("Executing pattern: " + state.CurrentPatternID)
			
			// Simulated output
			output := fmt.Sprintf("Pattern: %s\nInput: %s\n\nSimulated execution result.", 
				state.CurrentPatternID, input)
			
			outputArea.SetText(output)
			state.LastOutput = output
			statusBar.SetText("Pattern execution complete")
		}()
	})

	// Handle pattern selection
	patternList.OnSelected = func(id widget.ListItemID) {
		if id >= len(state.LoadedPatterns) {
			return
		}
		pattern := state.LoadedPatterns[id]
		state.CurrentPatternID = pattern.ID
		
		// Update preview
		systemText.SetText(pattern.SystemMD)
		userText.SetText(pattern.UserMD)
		patternInfoLabel.SetText(fmt.Sprintf("Selected pattern: %s", pattern.Name))
		
		// Update status
		statusBar.SetText(fmt.Sprintf("Selected pattern: %s", pattern.Name))
	}

	// Create main tabs
	patternPanel := container.NewBorder(
		nil, nil, nil, nil,
		container.NewHSplit(
			patternList,
			detailsTabs,
		),
	)
	
	executePanel := container.NewBorder(
		container.NewVBox(
			patternInfoLabel,
			executeBtn,
		), 
		nil, nil, nil,
		container.NewVSplit(inputArea, outputArea),
	)
	
	mainTabs := container.NewAppTabs(
		container.NewTabItem("Patterns", patternPanel),
		container.NewTabItem("Execute", executePanel),
	)

	// Create main layout
	mainContent := container.NewBorder(
		nil,         // top
		statusBar,   // bottom
		nil, nil,    // left, right
		mainTabs,    // center
	)

	win.SetContent(mainContent)
	win.Resize(fyne.NewSize(1200, 800))
	win.ShowAndRun()
}

// loadPatterns loads all patterns from the patterns directory
func loadPatterns() ([]Pattern, error) {
	// Locate Fabric data directory
	fabricDataDir, err := getFabricDataDir()
	if err != nil {
		return nil, fmt.Errorf("failed to locate Fabric data directory: %w", err)
	}

	// Initialize paths
	patternsDir := filepath.Join(fabricDataDir, "patterns")
	descriptionsPath := filepath.Join(fabricDataDir, "Pattern_Descriptions", "pattern_descriptions.json")

	// Load pattern descriptions
	descriptionsByName := make(map[string]PatternDescription)
	data, err := os.ReadFile(descriptionsPath)
	if err != nil {
		log.Printf("Warning: failed to read pattern descriptions: %v", err)
	} else {
		var descriptionsFile PatternDescriptionsFile
		if err := json.Unmarshal(data, &descriptionsFile); err != nil {
			log.Printf("Warning: failed to parse pattern descriptions: %v", err)
		} else {
			for _, desc := range descriptionsFile.Patterns {
				descriptionsByName[desc.PatternName] = desc
			}
		}
	}

	// List pattern directories
	entries, err := os.ReadDir(patternsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read patterns directory: %w", err)
	}

	// Load each pattern
	patterns := make([]Pattern, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue // Skip non-directories
		}

		patternID := entry.Name()
		
		// Create pattern with basic info
		pattern := Pattern{
			ID:   patternID,
			Name: formatPatternName(patternID),
		}

		// Load system.md
		systemPath := filepath.Join(patternsDir, patternID, "system.md")
		systemContent, err := os.ReadFile(systemPath)
		if err != nil {
			log.Printf("Warning: failed to read system.md for pattern %s: %v", patternID, err)
			continue
		}
		pattern.SystemMD = string(systemContent)

		// Try to load user.md (optional)
		userPath := filepath.Join(patternsDir, patternID, "user.md")
		userContent, err := os.ReadFile(userPath)
		if err == nil {
			pattern.UserMD = string(userContent)
		}

		// Add description and tags from pattern_descriptions.json if available
		if desc, ok := descriptionsByName[patternID]; ok {
			pattern.Description = desc.Description
			pattern.Tags = desc.Tags
		} else {
			// Fallback: derive description from first line of system.md
			pattern.Description = deriveDescription(pattern.SystemMD)
		}

		patterns = append(patterns, pattern)
	}

	// Sort patterns by name for better UX
	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].Name < patterns[j].Name
	})

	return patterns, nil
}

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

// formatPatternName converts pattern ID to a more readable display name
func formatPatternName(id string) string {
	// Replace underscores with spaces and capitalize words
	parts := strings.Split(id, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[0:1]) + part[1:]
		}
	}
	return strings.Join(parts, " ")
}

// deriveDescription extracts a short description from system.md content
func deriveDescription(systemMD string) string {
	// Find the first non-empty line that's not a heading
	lines := strings.Split(systemMD, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			// Truncate if needed
			if len(line) > 100 {
				return line[:97] + "..."
			}
			return line
		}
	}
	return "No description available"
}