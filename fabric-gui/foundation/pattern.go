package foundation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

// PatternLoader handles loading patterns from filesystem
type PatternLoader struct {
	PatternsDir        string // Directory containing pattern folders
	DescriptionsPath   string // Path to pattern_descriptions.json
	descriptionsByName map[string]PatternDescription
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

// NewPatternLoader creates a new pattern loader with the given paths
func NewPatternLoader(patternsDir, descriptionsPath string) *PatternLoader {
	return &PatternLoader{
		PatternsDir:        patternsDir,
		DescriptionsPath:   descriptionsPath,
		descriptionsByName: make(map[string]PatternDescription),
	}
}

// LoadPatternDescriptions loads pattern descriptions from JSON file
func (pl *PatternLoader) LoadPatternDescriptions() error {
	// Read the descriptions file
	data, err := os.ReadFile(pl.DescriptionsPath)
	if err != nil {
		return fmt.Errorf("failed to read pattern descriptions: %w", err)
	}

	// Parse the JSON
	var descriptionsFile PatternDescriptionsFile
	if err := json.Unmarshal(data, &descriptionsFile); err != nil {
		return fmt.Errorf("failed to parse pattern descriptions: %w", err)
	}

	// Build lookup map
	for _, desc := range descriptionsFile.Patterns {
		pl.descriptionsByName[desc.PatternName] = desc
	}

	return nil
}

// LoadAllPatterns loads all patterns from the patterns directory
func (pl *PatternLoader) LoadAllPatterns() ([]Pattern, error) {
	log.Println("LoadAllPatterns: Starting to load patterns from", pl.PatternsDir)
	
	// Make sure descriptions are loaded
	if len(pl.descriptionsByName) == 0 {
		log.Println("LoadAllPatterns: Loading pattern descriptions")
		if err := pl.LoadPatternDescriptions(); err != nil {
			log.Printf("LoadAllPatterns: Failed to load pattern descriptions: %v", err)
			return nil, err
		}
		log.Printf("LoadAllPatterns: Loaded %d pattern descriptions", len(pl.descriptionsByName))
	}

	// List pattern directories
	log.Println("LoadAllPatterns: Reading pattern directory")
	entries, err := os.ReadDir(pl.PatternsDir)
	if err != nil {
		log.Printf("LoadAllPatterns: Failed to read patterns directory: %v", err)
		return nil, fmt.Errorf("failed to read patterns directory: %w", err)
	}
	log.Printf("LoadAllPatterns: Found %d entries in patterns directory", len(entries))

	// Load each pattern
	patterns := make([]Pattern, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue // Skip non-directories
		}

		patternID := entry.Name()
		log.Printf("LoadAllPatterns: Loading pattern %s", patternID)
		pattern, err := pl.LoadPattern(patternID)
		if err != nil {
			log.Printf("LoadAllPatterns: Warning: failed to load pattern %s: %v", patternID, err)
			continue // Skip patterns that fail to load
		}

		patterns = append(patterns, pattern)
	}

	log.Printf("LoadAllPatterns: Successfully loaded %d patterns", len(patterns))
	return patterns, nil
}

// LoadPattern loads a single pattern by ID
func (pl *PatternLoader) LoadPattern(patternID string) (Pattern, error) {
	log.Printf("LoadPattern: Loading pattern %s", patternID)
	
	pattern := Pattern{
		ID:   patternID,
		Name: formatPatternName(patternID),
	}

	// Load system.md
	systemPath := filepath.Join(pl.PatternsDir, patternID, "system.md")
	log.Printf("LoadPattern: Reading system.md from %s", systemPath)
	systemContent, err := os.ReadFile(systemPath)
	if err != nil {
		log.Printf("LoadPattern: Failed to read system.md: %v", err)
		return Pattern{}, fmt.Errorf("failed to read system.md: %w", err)
	}
	pattern.SystemMD = string(systemContent)
	log.Printf("LoadPattern: Successfully read system.md (%d bytes)", len(systemContent))

	// Try to load user.md (optional)
	userPath := filepath.Join(pl.PatternsDir, patternID, "user.md")
	userContent, err := os.ReadFile(userPath)
	if err == nil {
		pattern.UserMD = string(userContent)
		log.Printf("LoadPattern: Successfully read user.md (%d bytes)", len(userContent))
	} else {
		log.Printf("LoadPattern: No user.md found (or error: %v)", err)
	}

	// Add description and tags from pattern_descriptions.json if available
	if desc, ok := pl.descriptionsByName[patternID]; ok {
		pattern.Description = desc.Description
		pattern.Tags = desc.Tags
		log.Printf("LoadPattern: Found description in JSON: %s", pattern.Description[:min(30, len(pattern.Description))])
	} else {
		// Fallback: derive description from first line of system.md
		pattern.Description = deriveDescription(pattern.SystemMD)
		log.Printf("LoadPattern: Derived description: %s", pattern.Description[:min(30, len(pattern.Description))])
	}

	log.Printf("LoadPattern: Successfully loaded pattern %s", patternID)
	return pattern, nil
}

// Helper functions

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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