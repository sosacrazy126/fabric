package foundation

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Pattern definition moved to types.go for centralized type management

// GetShortDescription moved to types.go for centralized type management

// PatternLoader handles loading patterns from filesystem
type PatternLoader struct {
	PatternsDir        string // Directory containing pattern folders
	DescriptionsPath   string // Path to pattern_descriptions.json
	descriptionsByName map[string]PatternDescription
	mutex              sync.RWMutex // Protects map during concurrent operations
	lastRefreshTime    time.Time // Tracks when descriptions were last refreshed
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
	// Use mutex to protect the map during update
	pl.mutex.Lock()
	defer pl.mutex.Unlock()
	
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

	// Create a new map (don't reuse existing one to avoid partial updates)
	newDescMap := make(map[string]PatternDescription)
	for _, desc := range descriptionsFile.Patterns {
		newDescMap[desc.PatternName] = desc
	}
	
	// Replace the map atomically
	pl.descriptionsByName = newDescMap
	pl.lastRefreshTime = time.Now()
	
	log.Printf("Loaded %d pattern descriptions", len(pl.descriptionsByName))
	return nil
}

// LoadAllPatterns loads all patterns from the patterns directory
func (pl *PatternLoader) LoadAllPatterns() ([]Pattern, error) {
	log.Println("LoadAllPatterns: Starting to load patterns from", pl.PatternsDir)
	
	// Make sure descriptions are loaded (thread-safe check)
	pl.mutex.RLock()
	descCount := len(pl.descriptionsByName)
	refreshNeeded := time.Since(pl.lastRefreshTime) > 1*time.Hour // Refresh once per hour
	pl.mutex.RUnlock()
	
	if descCount == 0 || refreshNeeded {
		log.Println("LoadAllPatterns: Loading pattern descriptions")
		if err := pl.LoadPatternDescriptions(); err != nil {
			log.Printf("LoadAllPatterns: Failed to load pattern descriptions: %v", err)
			// Continue anyway - we'll use derived descriptions as fallback
		}
	}

	// List pattern directories
	log.Println("LoadAllPatterns: Reading pattern directory")
	entries, err := os.ReadDir(pl.PatternsDir)
	if err != nil {
		log.Printf("LoadAllPatterns: Failed to read patterns directory: %v", err)
		return nil, fmt.Errorf("failed to read patterns directory: %w", err)
	}
	log.Printf("LoadAllPatterns: Found %d entries in patterns directory", len(entries))

	// Use worker pool to load patterns in parallel for better performance
	type patternResult struct {
		pattern Pattern
		err     error
	}
	
	// Create a buffered channel for results
	resultChan := make(chan patternResult, len(entries))
	
	// Start workers (limit to 8 concurrent goroutines to avoid overwhelming the system)
	workerCount := 8
	if len(entries) < workerCount {
		workerCount = len(entries)
	}
	
	// Create a channel for distributing work
	jobChan := make(chan string, len(entries))
	
	// Start worker pool
	for i := 0; i < workerCount; i++ {
		go func() {
			for patternID := range jobChan {
				pattern, err := pl.LoadPattern(patternID)
				resultChan <- patternResult{pattern, err}
			}
		}()
	}
	
	// Queue up all pattern directories for processing
	patternCount := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue // Skip non-directories
		}
		patternCount++
		jobChan <- entry.Name()
	}
	close(jobChan) // No more jobs to add
	
	// Collect results
	patterns := make([]Pattern, 0, patternCount)
	for i := 0; i < patternCount; i++ {
		result := <-resultChan
		if result.err != nil {
			log.Printf("LoadAllPatterns: Warning: failed to load pattern: %v", result.err)
			continue
		}
		patterns = append(patterns, result.pattern)
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
		Path: filepath.Join(pl.PatternsDir, patternID),
	}

	// Load system.md
	systemPath := filepath.Join(pattern.Path, "system.md")
	log.Printf("LoadPattern: Reading system.md from %s", systemPath)
	systemContent, err := os.ReadFile(systemPath)
	if err != nil {
		log.Printf("LoadPattern: Failed to read system.md: %v", err)
		return Pattern{}, fmt.Errorf("failed to read system.md for pattern '%s': %w", patternID, err)
	}
	pattern.SystemMD = string(systemContent)
	log.Printf("LoadPattern: Successfully read system.md (%d bytes)", len(systemContent))

	// Try to load user.md (optional)
	userPath := filepath.Join(pattern.Path, "user.md")
	userContent, err := os.ReadFile(userPath)
	if err == nil {
		pattern.UserMD = string(userContent)
		log.Printf("LoadPattern: Successfully read user.md (%d bytes)", len(userContent))
	} else {
		// Not having user.md is normal for many patterns
		pattern.UserMD = ""
	}

	// Add description and tags from pattern_descriptions.json if available
	pl.mutex.RLock() // Thread-safe read from the map
	desc, ok := pl.descriptionsByName[patternID]
	pl.mutex.RUnlock()
	
	if ok {
		pattern.Description = desc.Description
		pattern.Tags = desc.Tags
		if len(pattern.Description) > 0 {
			truncDesc := pattern.Description
			if len(truncDesc) > 30 {
				truncDesc = truncDesc[:30] + "..."
			}
			log.Printf("LoadPattern: Found description in JSON: %s", truncDesc)
		}
	} else {
		// Fallback: derive description from first line of system.md
		pattern.Description = deriveDescription(pattern.SystemMD)
		pattern.Tags = deriveTagsFromContent(pattern.SystemMD, patternID)
		log.Printf("LoadPattern: Derived description (no JSON entry found)")
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

// deriveTagsFromContent extracts potential tags from the pattern content and ID
func deriveTagsFromContent(systemMD string, patternID string) []string {
	tagSet := make(map[string]struct{})
	
	// Add tags from pattern ID (e.g., "analyze_threat_report" -> ["analyze", "threat", "report"])
	parts := strings.Split(patternID, "_")
	for _, part := range parts {
		if len(part) > 2 { // Avoid very short words
			tagSet[part] = struct{}{}
		}
	}
	
	// Look for common keywords in system.md
	keywords := []string{
		"analyze", "summarize", "extract", "create", "generate", "explain", 
		"write", "review", "evaluate", "translate", "convert", "recommend",
		"security", "threat", "report", "code", "article", "email", "paper",
		"academic", "business", "technical", "creative", "visualization",
	}
	
	for _, keyword := range keywords {
		if strings.Contains(strings.ToLower(systemMD), keyword) {
			tagSet[keyword] = struct{}{}
		}
	}
	
	// Convert set to slice
	tags := make([]string, 0, len(tagSet))
	for tag := range tagSet {
		tags = append(tags, tag)
	}
	
	return tags
}