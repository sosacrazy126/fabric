package foundation

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// FabricPaths holds important filesystem locations for the application
type FabricPaths struct {
	// Core directories
	HomeDir           string
	ConfigDir         string // ~/.config/fabric
	
	// Pattern-related paths
	PatternsDir       string // ~/.config/fabric/patterns
	DescriptionsPath  string // ~/.config/fabric/Pattern_Descriptions/pattern_descriptions.json
	
	// Configuration files
	EnvFile           string // ~/.config/fabric/.env
	
	// Cache directory
	CacheDir          string // ~/.config/fabric/cache
	
	// Temporary directory for file operations
	TempDir           string
}

// GetFabricPaths locates all necessary directories and files for Fabric
func GetFabricPaths() (*FabricPaths, error) {
	paths := &FabricPaths{}
	
	// Get user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	paths.HomeDir = homeDir
	
	// Set config directory based on OS
	// Fabric typically uses ~/.config/fabric on Unix systems
	if runtime.GOOS == "windows" {
		paths.ConfigDir = filepath.Join(homeDir, "AppData", "Local", "fabric")
	} else {
		paths.ConfigDir = filepath.Join(homeDir, ".config", "fabric")
	}
	
	// Check if we're in development mode
	devMode := false
	if _, err := os.Stat(paths.ConfigDir); os.IsNotExist(err) {
		log.Println("Config directory not found at", paths.ConfigDir)
		log.Println("Checking for development environment...")
		
		// Try to find patterns in the repository structure
		cwd, err := os.Getwd()
		if err == nil {
			// Look up the directory tree for the patterns folder
			for i := 0; i < 3; i++ { // Check up to 3 levels up
				repoRoot := filepath.Join(cwd, strings.Repeat("...", i))
				patternsDirInRepo := filepath.Join(repoRoot, "patterns")
				
				if _, err := os.Stat(patternsDirInRepo); !os.IsNotExist(err) {
					log.Println("Found patterns directory in repository at", patternsDirInRepo)
					paths.PatternsDir = patternsDirInRepo
					paths.DescriptionsPath = filepath.Join(repoRoot, "Pattern_Descriptions", "pattern_descriptions.json")
					devMode = true
					break
				}
			}
		}
	}
	
	// If not in dev mode, use standard Fabric paths
	if !devMode {
		paths.PatternsDir = filepath.Join(paths.ConfigDir, "patterns")
		paths.DescriptionsPath = filepath.Join(paths.ConfigDir, "Pattern_Descriptions", "pattern_descriptions.json")
	}
	
	// Set remaining paths
	paths.EnvFile = filepath.Join(paths.ConfigDir, ".env")
	paths.CacheDir = filepath.Join(paths.ConfigDir, "cache")
	
	// Create temporary directory
	tempDir := os.TempDir()
	paths.TempDir = filepath.Join(tempDir, "fabric-gui")
	
	// Ensure the temp directory exists
	if err := os.MkdirAll(paths.TempDir, 0755); err != nil {
		log.Printf("Warning: Failed to create temp directory: %v", err)
		// Continue anyway, temp dir will be created on demand
	}
	
	return paths, nil
}

// ValidatePaths checks if critical paths exist and logs warnings
func (p *FabricPaths) ValidatePaths() []string {
	var warnings []string
	
	// Check for patterns directory
	if _, err := os.Stat(p.PatternsDir); os.IsNotExist(err) {
		warning := "Patterns directory not found: " + p.PatternsDir
		log.Println("WARNING:", warning)
		warnings = append(warnings, warning)
	}
	
	// Check for pattern descriptions file
	if _, err := os.Stat(p.DescriptionsPath); os.IsNotExist(err) {
		warning := "Pattern descriptions file not found: " + p.DescriptionsPath
		log.Println("WARNING:", warning)
		warnings = append(warnings, warning)
	}
	
	// Check for .env file (not critical but useful to log)
	if _, err := os.Stat(p.EnvFile); os.IsNotExist(err) {
		warning := "Environment file not found: " + p.EnvFile
		log.Println("WARNING:", warning)
		warnings = append(warnings, warning)
	}
	
	return warnings
}