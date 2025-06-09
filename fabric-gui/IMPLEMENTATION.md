# Fabric GUI Implementation Details

This document provides technical details on the current implementation of the Fabric GUI application.

## Architecture Overview

The Fabric GUI follows a component-based architecture with clear separation of concerns:

1. **Core Application** (`FabricApp` in `foundation/app.go`)
   - Central coordination point
   - Manages application state
   - Loads patterns from Fabric's database with filesystem fallback
   - Integrates with Fabric's execution pipeline
   - Handles configuration management
   - Initializes UI components

2. **Layout System** (`MainLayout` in `foundation/layouts.go`)
   - Modern split-view design with sidebar and content area
   - Manages tab switching and component interaction
   - Handles responsive layout adjustments

3. **Components**
   - `SidebarPanel`: Pattern navigation and filtering
   - `MainContentPanel`: Tabbed interface for input, output, and pattern details
   - `PatternInfoArea`: Displays pattern details
   - `InputArea`: Handles different input sources
   - `OutputArea`: Displays execution results

## State Management

Application state is centralized in the `AppState` structure:

```go
type AppState struct {
    // Pattern Selection
    CurrentPatternID   string
    CurrentInputText   string
    LastOutput         string
    LoadedPatterns     []Pattern
    FilteredPatterns   []Pattern
    
    // Model Configuration
    CurrentModelID     string
    CurrentModelName   string
    CurrentVendorID    string
    Temperature        float64
    TopP               float64
    PresencePenalty    float64
    FrequencyPenalty   float64
    Seed               int
    ContextLength      int
    Strategy           string
    
    // UI State
    LastActiveTab      string
    InputSourceType    string // "Text", "Clipboard", "File", "URL"
    OutputFormat       string // "Text", "Markdown", "JSON"
    SearchQuery        string
    SelectedTags       []string
    
    // Data Caches
    LoadedVendors      []string
    LoadedModels       map[string][]string
    LoadedStrategies   []string
    
    // Session History
    LastUsedPatterns   []string
    LastInputs         []string
    LastRun            time.Time
}
```

## Pattern Loading

Patterns are loaded using a multi-tiered approach:

1. **Primary Source**: Fabric's `fsdb` database via `foundation/config.go`
   - Uses Fabric's internal APIs to access patterns
   - Retrieves metadata, tags, and content
   - Handles pattern organization and grouping

2. **Fallback Mechanism**: Direct filesystem loading via `PatternLoader` in `foundation/pattern.go`
   - Locates the Fabric data directory using `foundation/paths.go`
   - Finds pattern folders and descriptions
   - Loads each pattern's system.md and user.md files
   - Uses worker pools to load patterns in parallel for better performance

Both approaches include comprehensive error handling, timeouts, and graceful degradation to prevent the application from hanging or crashing.

```go
// Sample from ExecutionManager in foundation/execution.go
func (em *ExecutionManager) ExecutePattern(config ExecutionConfig) (*ExecutionResult, error) {
    // Create execution options from config
    execOptions := common.Options{
        Model:           config.Model,
        Vendor:          config.Vendor,
        Temperature:     config.Temperature,
        TopP:            config.TopP,
        PresencePenalty: config.PresencePenalty,
        // Other options...
    }
    
    // Create session using Fabric's BuildSession
    session, err := cli.BuildSession(ctx, db, pattern.ID, config.Input, &execOptions)
    if err != nil {
        return nil, fmt.Errorf("failed to build session: %w", err)
    }
    
    // Execute session
    response, err := session.Execute()
    if err != nil {
        return nil, fmt.Errorf("execution failed: %w", err)
    }
    
    // Return results
    return &ExecutionResult{
        Output:        response,
        PatternID:     config.PatternID,
        Timestamp:     time.Now(),
        ExecutionTime: time.Since(startTime),
        Success:       true,
    }, nil
}
```

## UI Implementation

The UI is built using the Fyne toolkit with a component-based approach:

- Each UI component implements the `FyneComponent` interface
- Components are arranged in a hierarchical structure
- Event handling is done through callbacks and state updates

The main layout is a horizontal split with:
- Left side: Pattern navigation sidebar
- Right side: Tabbed interface for input, output, and pattern details

## Current Features

1. **Pattern Management**: Full pattern loading from Fabric's database with filesystem fallback and parallel loading for performance.

2. **Search and Filtering**: Complete search by name, description, and content, with tag-based filtering.

3. **Core Integration**: Full integration with Fabric's core processing capabilities through the `ExecutionManager` in `foundation/execution.go`.

4. **Configuration Management**: Settings are loaded from and saved to Fabric's .env file via `FabricConfig` in `foundation/config.go`.

5. **Model Selection**: Support for selecting models and vendors, with parameter adjustment.

## Current Limitations

1. **Input Sources**: Advanced input handling for files and URLs is partially implemented.

2. **Output Formatting**: Advanced output formatting (markdown, syntax highlighting) is not fully implemented.

3. **Data Persistence**: While configuration is saved, execution history and pattern favorites are not persisted between sessions.

## Next Steps

### Immediate Tasks

1. **Complete Input Sources**: Finish the implementation of file uploading and URL fetching.

2. **Enhance Output Formatting**: Add markdown rendering and syntax highlighting for code blocks.

3. **Session Management**: Implement saving and loading of execution history and user preferences.

### Medium-Term Tasks

1. **Advanced Settings Panel**: Enhance the settings panel with more configuration options and presets.

2. **Result Management**: Add functionality to export results in various formats (PDF, HTML, JSON).

3. **Enhanced UI**: Add tooltips, keyboard shortcuts, and accessibility improvements.

4. **Dark Mode**: Implement dark mode and customizable themes.

### Long-Term Vision

1. **Extensibility**: Plugin system for custom pattern repositories and execution backends.

2. **Advanced Features**: Pattern comparison, batch processing, scheduled execution.

3. **Integration**: Connect with external tools and services for enhanced functionality.

## Coding Patterns and Conventions

The codebase follows these patterns and conventions:

1. **Composition Over Inheritance**: UI components are composed rather than inherited, following Fyne's container model.

2. **Centralized State Management**: Application state is managed centrally in `AppState` with clear update patterns.

3. **Comprehensive Error Handling**: Multi-tiered error handling with logging, user feedback, and graceful fallbacks.

4. **Progressive Enhancement**: The application functions with minimal data, adding features as available.

5. **Separation of Concerns**: Clear separation between UI components, data management, and business logic.

6. **Concurrency Patterns**: Safe goroutine usage with proper synchronization and timeout handling.

7. **Configuration Management**: Structured approach to loading, validating, and saving configuration.

8. **Path Resolution**: Flexible path handling for different environments (development, user installation).