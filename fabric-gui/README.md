# Fabric GUI

A modern graphical user interface for the Fabric AI pattern tool, built with Go and the Fyne toolkit.

## Overview

Fabric GUI provides a user-friendly interface to interact with Fabric AI patterns. It allows users to:

- Browse and search through available patterns
- View pattern details including system and user prompts
- Execute patterns with custom inputs
- View and save execution results

## Architecture

The application follows a component-based architecture:

- **Main Application**: Central coordination point
- **Layout System**: Modern split-view design with sidebar and content area
- **Patterns Panel**: Browse, search, and select patterns
- **Input/Output Areas**: Work with pattern inputs and view results
- **Pattern Info**: View detailed pattern information

## Current Implementation Status

As of June 2023:

- ✅ Core application structure implemented
- ✅ Pattern loading from file system
- ✅ Modern UI layout with sidebar and main content
- ✅ Pattern browsing and selection
- ✅ Basic execution flow
- ✅ Error handling and logging
- ⚠️ Search/filtering partially implemented
- ❌ API integration with Fabric core not yet implemented
- ❌ Settings and configuration not yet implemented

## Development Requirements

- Go 1.17 or later
- Fyne v2 toolkit
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
│   └── gui/             # Command-line entry point
├── foundation/          # Core application components
│   ├── app.go           # Main application
│   ├── layouts.go       # UI layouts and components
│   ├── pattern.go       # Pattern loading and management
│   └── types.go         # Type definitions
├── assets/              # Application assets
├── main.go              # Main entry point
└── run.sh               # Helper script
```

## Troubleshooting

See [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) for common issues and solutions.

## Future Development

Planned features:
- Complete search and filtering functionality
- Integration with Fabric's core processing capabilities
- Save/load user inputs and results
- Model and vendor selection
- Parameter adjustment for different AI models
- Export results in various formats