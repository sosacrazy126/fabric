package foundation

import (
    "time"

    "fyne.io/fyne/v2"
)

// Pattern represents a Fabric pattern with metadata and prompts.
type Pattern struct {
    ID          string   // Unique pattern identifier
    Name        string   // Human-readable name
    Path        string   // Filesystem path to the pattern directory
    SystemMD    string   // Content of system.md
    UserMD      string   // Content of user.md (optional)
    Description string   // Short description of the pattern
    Tags        []string // Associated tags for filtering and categorization
}

// AppState manages centralized application state
type AppState struct {
    // Pattern Selection
    CurrentPatternID   string
    CurrentInputText   string // The text content of the current input (from any source)
    LastOutput         string
    LoadedPatterns     []Pattern
    FilteredPatterns   []Pattern // Patterns currently displayed after search/filter
    
    // Model Configuration (loaded from .env/settings)
    CurrentModelID     string // The actual model ID used by Fabric (e.g., "gpt-4o")
    CurrentModelName   string // User-friendly model name (e.g., "GPT-4o")
    CurrentVendorID    string // The vendor ID for the model (e.g., "openai")
    Temperature        float64
    TopP               float64
    PresencePenalty    float64
    FrequencyPenalty   float64
    Seed               int
    ContextLength      int // Model context length (e.g., for Ollama)
    Strategy           string // For prompt strategies (e.g., "cot", "tot")
    
    // UI State
    LastActiveTab      string // To remember last active tab (e.g., "Input", "Output", "Pattern Info")
    InputSourceType    string // "Text", "Clipboard", "File", "URL", "YouTube"
    OutputFormat       string // "Text", "Markdown", "JSON"
    SearchQuery        string // Current pattern search query
    SelectedTags       []string // Currently selected filter tags
    
    // Data Caches (from Fabric's fsdb)
    LoadedVendors      []string  // Cache of available vendor names
    LoadedModels       map[string][]string // Cache of models by vendor (map[vendor] -> []model_names)
    VendorModelCounts  map[string]int // Cache of model counts by vendor for UI hints
    LoadedStrategies   []string // Cache of available strategy names
    StarredOutputs     []OutputSnapshot // User's favorited outputs
    
    // Session History
    LastUsedPatterns   []string // IDs of recently used patterns
    LastInputs         []string // Recent inputs (limit to 10)
    LastRun            time.Time
}

// OutputSnapshot stores a saved output from a pattern execution
type OutputSnapshot struct {
    ID           string
    PatternID    string
    PatternName  string
    Timestamp    time.Time
    InputText    string
    OutputText   string
    Model        string
    Vendor       string
    CustomName   string // User-provided name
}

// NewAppState initializes AppState with default values.
func NewAppState() *AppState {
    return &AppState{
        // Default model parameters
        Temperature:      0.7,
        TopP:             0.9,
        PresencePenalty:  0.0,
        FrequencyPenalty: 0.0,
        Seed:             0,
        ContextLength:    4096, // Default for many models
        Strategy:         "standard",
        
        // Default UI state
        InputSourceType:  "Text",
        OutputFormat:     "Text",
        LastActiveTab:    "Execute", // Default to Execute tab
        
        // Initialize collections
        LoadedPatterns:   []Pattern{},
        FilteredPatterns: []Pattern{},
        SelectedTags:     []string{},
        LastUsedPatterns: []string{},
        LastInputs:       []string{},
        LastRun:          time.Now(),
        
        // Initialize caches
        LoadedModels:     make(map[string][]string),
        VendorModelCounts: make(map[string]int),
        StarredOutputs:   []OutputSnapshot{},
    }
}

// ExecutionConfig wraps parameters for a single pattern execution.
type ExecutionConfig struct {
    PatternID         string
    Input             string
    // Fabric Core parameters, taken from AppState
    Model             string
    Vendor            string
    Temperature       float64
    TopP              float64
    PresencePenalty   float64
    FrequencyPenalty  float64
    Seed              int
    ContextLength     int
    Strategy          string
    Stream            bool
    DryRun            bool
}

// ExecutionResult wraps the outcome of a pattern execution.
type ExecutionResult struct {
    Output         string
    PatternID      string
    Timestamp      time.Time
    TokensUsed     int
    ExecutionTime  time.Duration
    Success        bool
    Error          error
}

// FyneComponent is a base interface for all custom Fyne components/tabs.
type FyneComponent interface {
    Container() fyne.CanvasObject // Method to return the root Fyne container for the component
}