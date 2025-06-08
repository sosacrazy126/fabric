# Fabric GUI Troubleshooting Guide

## Overview

This document outlines common issues and solutions for the Fabric GUI implementation, which uses the Fyne toolkit for Go.

## Common Issues

### 1. Application Loads But Doesn't Process

**Symptoms:**
- GUI window appears but remains unresponsive
- No error messages in console
- Application seems to be "stuck" in a loading state

**Potential Causes:**
- Missing system dependencies for Fyne
- OpenGL compatibility issues
- Goroutine deadlocks in event handling
- Pattern loading failures

### 2. Missing Libraries

Fyne applications require several system libraries to function properly:
- libGL.so.1 (OpenGL)
- libX11.so.6 (X11 window system)
- libXrandr.so.2 (X11 resize and rotate extension)
- libXcursor.so.1 (X11 cursor management)
- libXinerama.so.1 (X11 multi-monitor support)

### 3. Pattern Loading Issues

The application attempts to load patterns from either:
- The current working directory's `patterns` folder
- The `~/.config/fabric` directory

If patterns cannot be found in either location, the application may load but display empty pattern lists.

## Troubleshooting Steps

### 1. Verify System Dependencies

For Ubuntu/Debian:
```bash
sudo apt-get install libgl1-mesa-dev xorg-dev
```

For Fedora/RHEL:
```bash
sudo dnf install mesa-libGL-devel xorg-x11-server-devel
```

### 2. Check Go Version

Fyne requires Go 1.17 or later:
```bash
go version
```

### 3. Verify OpenGL Support

Check if your system supports OpenGL:
```bash
glxinfo | grep "OpenGL version"
```

### 4. Debug Application Launch

Run the application with verbose logging:
```bash
FYNE_DEBUG=1 go run cmd/gui/main.go
```

### 5. Test With Minimal Example

The repository includes a minimal test application (`test_fyne.go`) that can help determine if the issue is with Fyne itself or with the Fabric GUI implementation.

## Environment Variables

Fyne supports several environment variables that can help troubleshoot issues:

- `FYNE_THEME`: Set to "light" or "dark" to force a specific theme
- `FYNE_SCALE`: Control UI scaling (e.g., "1.2" for 120% scaling)
- `FYNE_DEBUG`: Enable debug logging when set to "1"
- `FYNE_FONT`: Specify a custom font path

## Next Steps

If the application still fails to run properly after following these steps, consider:

1. Running the application in a different environment (container or VM)
2. Trying an alternative GUI toolkit
3. Creating a simpler version of the application focusing on core functionality

## Further Resources

- [Fyne Documentation](https://developer.fyne.io/)
- [Go Packages Documentation for Fyne](https://pkg.go.dev/fyne.io/fyne/v2)
- [Fyne GitHub Issues](https://github.com/fyne-io/fyne/issues)