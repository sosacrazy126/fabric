# Fabric GUI Implementation Details

This document provides technical details on the current implementation of the Fabric GUI application.

## Architecture Overview

The Fabric GUI follows a component-based architecture with clear separation of concerns:

1. **Core Application** (`FabricApp` in `foundation/app.go`)
   - Central coordination point
   - Manages application state
   - Loads patterns from filesystem
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
    CurrentPatternID   string
    CurrentInputText   string
    LastOutput         string
    LoadedPatterns     []Pattern
    FilteredPatterns   []Pattern
    InputSourceType    string // "Text", "Clipboard", "File", "URL", "YouTube"
    
    // Model parameters
    CurrentModelID     string
    CurrentModelName   string
    CurrentVendorID    string
    Temperature        float64
    // ... additional parameters
}
```

## Pattern Loading

Patterns are loaded from the filesystem using the `PatternLoader` in `foundation/pattern.go`. The loader:

1. Locates the Fabric data directory
2. Finds pattern folders and descriptions
3. Loads each pattern's system.md and user.md files
4. Populates the application state with loaded patterns

The pattern loading process includes error handling and timeouts to prevent the application from hanging.

## UI Implementation

The UI is built using the Fyne toolkit with a component-based approach:

- Each UI component implements the `FyneComponent` interface
- Components are arranged in a hierarchical structure
- Event handling is done through callbacks and state updates

The main layout is a horizontal split with:
- Left side: Pattern navigation sidebar
- Right side: Tabbed interface for input, output, and pattern details

## Current Limitations

1. **Pattern Filtering**: The search and tag filtering functionality is partially implemented.

2. **Core Integration**: The application currently does not integrate with Fabric's core processing capabilities. The execution function (`executePattern`) contains placeholder code that simulates pattern execution.

3. **Settings**: Configuration and settings management is not yet implemented.

4. **Data Persistence**: User inputs, results, and settings are not saved between sessions.

## Next Steps

### Immediate Tasks

1. **Complete Search/Filtering**: Implement the pattern filtering functionality in the sidebar panel.

2. **Integrate with Fabric Core**: Connect the execution logic to Fabric's actual pattern processing.

3. **Input Sources**: Complete the implementation of different input sources (clipboard, file, URL).

### Medium-Term Tasks

1. **Settings Panel**: Create a settings panel for configuring models, vendors, and parameters.

2. **Result Management**: Add functionality to save, load, and export results.

3. **Enhanced UI**: Add tooltips, keyboard shortcuts, and improved layout options.

### Long-Term Vision

1. **Extensibility**: Plugin system for custom pattern repositories and execution backends.

2. **Advanced Features**: Pattern comparison, batch processing, scheduled execution.

3. **Integration**: Connect with external tools and services for enhanced functionality.

## Coding Patterns and Conventions

The codebase follows these patterns and conventions:

1. **Composition Over Inheritance**: UI components are composed rather than inherited.

2. **Centralized State**: Application state is managed centrally and passed to components.

3. **Error Handling**: Errors are logged and, where possible, presented to the user.

4. **Progressive Enhancement**: The application can function with minimal pattern data, adding features as available.

5. **Separation of Concerns**: UI components, data loading, and business logic are separated.