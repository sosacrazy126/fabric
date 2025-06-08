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
	// Make sure descriptions are loaded
	if len(pl.descriptionsByName) == 0 {
		if err := pl.LoadPatternDescriptions(); err != nil {
			return nil, err
		}
	}

	// List pattern directories
	entries, err := os.ReadDir(pl.PatternsDir)
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
		pattern, err := pl.LoadPattern(patternID)
		if err != nil {
			fmt.Printf("Warning: failed to load pattern %s: %v\n", patternID, err)
			continue // Skip patterns that fail to load
		}

		patterns = append(patterns, pattern)
	}

	return patterns, nil
}

// LoadPattern loads a single pattern by ID
func (pl *PatternLoader) LoadPattern(patternID string) (Pattern, error) {
	pattern := Pattern{
		ID:   patternID,
		Name: formatPatternName(patternID),
	}

	// Load system.md
	systemPath := filepath.Join(pl.PatternsDir, patternID, "system.md")
	systemContent, err := os.ReadFile(systemPath)
	if err != nil {
		return Pattern{}, fmt.Errorf("failed to read system.md: %w", err)
	}
	pattern.SystemMD = string(systemContent)

	// Try to load user.md (optional)
	userPath := filepath.Join(pl.PatternsDir, patternID, "user.md")
	userContent, err := os.ReadFile(userPath)
	if err == nil {
		pattern.UserMD = string(userContent)
	}

	// Add description and tags from pattern_descriptions.json if available
	if desc, ok := pl.descriptionsByName[patternID]; ok {
		pattern.Description = desc.Description
		pattern.Tags = desc.Tags
	} else {
		// Fallback: derive description from first line of system.md
		pattern.Description = deriveDescription(pattern.SystemMD)
	}

	return pattern, nil
}

// Helper functions

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