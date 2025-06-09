# Fabric GUI

A modern graphical user interface for the Fabric AI pattern execution system, built with Go and the Fyne toolkit.

## Overview

Fabric GUI provides a user-friendly interface to interact with Fabric AI patterns. It allows users to:

- Browse and search through available patterns
- View pattern details including system and user prompts
- Execute patterns with custom inputs
- View and save execution results

## Architecture

The application follows a modern component-based architecture with clean separation of concerns:

### Core Components

- **AppState**: Centralized state management for the entire application
- **FabricConfig**: Interface to Fabric's configuration and database
- **ExecutionManager**: Handles pattern execution with proper error handling
- **PatternLoader**: Loads patterns from filesystem or Fabric's database

### UI Components

- **MainLayout**: Split view with sidebar and main content areas
- **SidebarPanel**: Pattern browsing, searching, and selection
- **MainContentPanel**: Tabbed interface for input, output, and pattern details
- **InputArea**: Text, file, URL inputs with source selection
- **OutputArea**: Result display with formatting options
- **PatternInfoArea**: Pattern metadata, prompts, and tags

## Current Implementation Status

As of June 2023:

- ✅ Core application structure implemented
- ✅ Pattern loading from Fabric's database
- ✅ Modern UI layout with sidebar and main content
- ✅ Pattern browsing and selection
- ✅ Search/filtering functionality
- ✅ Model and vendor selection
- ✅ Execution pipeline with Fabric core integration
- ✅ Configuration management and persistence
- ✅ Comprehensive error handling and logging
- ⚠️ Input handling for files and URLs partially implemented
- ⚠️ Output formatting options partially implemented

## Development Requirements

- Go 1.20 or later
- Fyne v2 toolkit
- Fabric CLI installed and configured
- C compiler for CGO
- Graphics drivers that support OpenGL

### Dependencies for Linux

On Debian/Ubuntu:
```bash
sudo apt-get install libgl1-mesa-dev xorg-dev
```

On Fedora/RHEL:
```bash
sudo dnf install mesa-libGL-devel xorg-x11-server-devel
```

## Running the Application

Standard run:
```bash
go run main.go
```

Run with pattern loading disabled (faster startup for development):
```bash
FABRIC_GUI_SKIP_PATTERNS=1 go run main.go
```

Or use the provided run script:
```bash
./run.sh --skip-patterns
```

## Project Structure

```
fabric-gui/
├── cmd/
│   └── gui/
│       └── main.go      # Application entry point
├── foundation/          # Core application components
│   ├── app.go           # Main application structure
│   ├── config.go        # Configuration management
│   ├── execution.go     # Pattern execution pipeline
│   ├── layouts.go       # UI components and layouts
│   ├── paths.go         # Filesystem path management
│   ├── pattern.go       # Pattern loading and management
│   └── types.go         # Data structures and state
├── assets/              # Application assets
└── run.sh               # Helper script
```

## Troubleshooting

See [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) for common issues and solutions.

## Future Development

Planned features:
- Enhanced input handling for files, URLs, and clipboard
- Advanced output formatting with markdown and syntax highlighting
- Save/load user sessions and execution history
- Batch pattern execution
- Enhanced visualization of execution metrics
- Export results in various formats (PDF, HTML, etc.)
- Keyboard shortcuts and accessibility improvements
- Dark mode and customizable themes