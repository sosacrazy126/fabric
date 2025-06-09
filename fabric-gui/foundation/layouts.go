package foundation

import (
    "fmt"
    "sort"
    "strings"
    "time"

    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/container"
    "fyne.io/fyne/v2/dialog"
    "fyne.io/fyne/v2/theme"
    "fyne.io/fyne/v2/widget"
)

// MainLayout defines the primary UI structure mirroring Streamlit's layout.
type MainLayout struct {
    app *FabricApp // Reference to the main application

    Sidebar     *SidebarPanel    // Left panel for patterns & settings
    MainContent *MainContentPanel // Right main area for input/output/details
    StatusBar   *StatusBar       // Bottom status bar

    container *fyne.Container // The root container for the layout
}

// NewMainLayout creates and initializes the main layout components.
func NewMainLayout(app *FabricApp) *MainLayout {
    layout := &MainLayout{app: app}

    // Create Panels
    layout.Sidebar = NewSidebarPanel(app)
    layout.MainContent = NewMainContentPanel(app)
    layout.StatusBar = NewStatusBar() // StatusBar is a simple label

    // Assemble layout components
    splitContent := container.NewHSplit(
        layout.Sidebar.Container(),
        layout.MainContent.Container(),
    )
    splitContent.SetOffset(0.25) // Initial split: Sidebar takes 25%

    layout.container = container.NewBorder(
        nil,                  // Top (no global top bar in this Streamlit-like layout)
        layout.StatusBar.Container(), // Bottom status bar
        nil,                  // Left (handled by HSplit)
        nil,                  // Right (handled by HSplit)
        splitContent,         // Center Split containing sidebar and main content
    )

    // Set up global tab change handler for main content panel
    // This ensures input/output are cleared and pattern info updated on tab switch
    layout.MainContent.tabs.OnChanged = func(tab *container.TabItem) {
        app.state.LastActiveTab = tab.Text // Update state with active tab
        if tab.Text == "Execute" {
            // Update pattern info and run button based on current state
            if app.state.CurrentPatternID != "" {
                patternName := app.getPatternNameByID(app.state.CurrentPatternID)
                app.mainLayout.MainContent.patternInfoArea.UpdateInfo(patternName, app.state.CurrentModelName, app.state.CurrentVendorID)
                app.mainLayout.MainContent.UpdateRunButton(patternName)
            } else {
                app.mainLayout.MainContent.patternInfoArea.UpdateInfo("No pattern selected", "", "")
                app.mainLayout.MainContent.UpdateRunButton("")
            }
        } else if tab.Text == "Results" {
            // Update output tab heading with execution info if available
            if app.state.LastOutput != "" && app.state.LastRun != (time.Time{}) {
                app.mainLayout.MainContent.outputArea.outputInfo.SetText(fmt.Sprintf(
                    "Last executed: %s", app.state.LastRun.Format("Jan 2, 2006 15:04:05")))
            }
        } else if tab.Text == "Pattern Details" {
            // When switching to Pattern Details tab, update pattern info
            if app.state.CurrentPatternID != "" {
                patternName := app.getPatternNameByID(app.state.CurrentPatternID)
                app.mainLayout.MainContent.patternInfoArea.UpdateInfo(patternName, app.state.CurrentModelName, app.state.CurrentVendorID)
            } else {
                app.mainLayout.MainContent.patternInfoArea.UpdateInfo("No pattern selected", "", "")
            }
        }
    }

    return layout
}

// Container returns the root Fyne container for the MainLayout.
func (ml *MainLayout) Container() fyne.CanvasObject {
    return ml.container
}

// SidebarPanel manages pattern selection, search, and quick settings.
type SidebarPanel struct {
    app *FabricApp // Reference to the main app

    container *fyne.Container

    // Pattern Management
    patternList   *widget.List
    searchEntry   *widget.Entry
    patternFilter *widget.Select // For tag filtering
    patternSection *CollapsibleSection // Collapsible patterns section

    // Model Provider
    modelSelect  *widget.Select
    vendorSelect *widget.Select
    modelSection *CollapsibleSection // Collapsible model section
    
    // Parameter Settings
    parameterSection *CollapsibleSection // Collapsible parameters section
    // paramSliders map[string]*widget.Slider // E.g., Temperature, TopP etc.
}

// NewSidebarPanel creates a new sidebar panel.
func NewSidebarPanel(app *FabricApp) *SidebarPanel {
    sb := &SidebarPanel{app: app}

    // Pattern Search Entry
    sb.searchEntry = widget.NewEntry()
    sb.searchEntry.SetPlaceHolder("Search patterns...")
    sb.searchEntry.OnChanged = func(text string) {
        app.state.SearchQuery = text // Store search query in state
        filterPatterns(app) // Apply filters
    }

    // Build tag options from loaded patterns
    tagOptions := extractTagOptions(app.state.LoadedPatterns)
    
    // Pattern Filter Select (for tags)
    sb.patternFilter = widget.NewSelect(tagOptions, func(selected string) {
        // Store selected tag in state
        if selected == "All" {
            app.state.SelectedTags = nil // Clear tag filter
        } else {
            app.state.SelectedTags = []string{selected} // Filter by selected tag
        }
        filterPatterns(app) // Apply filters
    })
    sb.patternFilter.SetSelected("All")

    // Pattern List
    sb.patternList = widget.NewList(
        func() int { return len(app.state.FilteredPatterns) }, // Use filtered list
        func() fyne.CanvasObject {
            // Template for each list item
            nameLabel := widget.NewLabel("Pattern Name")
            nameLabel.TextStyle = fyne.TextStyle{Bold: true}
            descLabel := widget.NewLabel("Description")
            descLabel.Importance = widget.LowImportance
            return container.NewVBox(nameLabel, descLabel)
        },
        func(id widget.ListItemID, obj fyne.CanvasObject) {
            // Update item content
            if id < len(app.state.FilteredPatterns) {
                pattern := app.state.FilteredPatterns[id]
                itemContainer := obj.(*fyne.Container)
                nameLabel := itemContainer.Objects[0].(*widget.Label)
                descLabel := itemContainer.Objects[1].(*widget.Label)
                nameLabel.SetText(pattern.Name)
                descLabel.SetText(getShortDescription(pattern))
            }
        },
    )
    sb.patternList.OnSelected = func(id widget.ListItemID) {
        if id < len(app.state.FilteredPatterns) {
            selectedPattern := app.state.FilteredPatterns[id]
            app.state.CurrentPatternID = selectedPattern.ID
            
            // Add to recent patterns list (for history)
            addToRecentPatterns(app, selectedPattern.ID)
            
            // Update Pattern Info tab in main content area
            app.mainLayout.MainContent.patternInfoArea.SetPattern(selectedPattern)
            
            // Update Run button in Execute tab
            app.mainLayout.MainContent.UpdateRunButton(selectedPattern.Name)
            app.ShowMessage(fmt.Sprintf("Selected pattern: %s", selectedPattern.Name))
        }
    }

    // Initialize model select with a placeholder
    sb.modelSelect = widget.NewSelect([]string{"Loading models..."}, nil)
    
    // Populate with models if available
    if len(app.state.LoadedVendors) > 0 && app.state.CurrentVendorID != "" {
        // Get models for the current vendor
        if models, ok := app.state.LoadedModels[app.state.CurrentVendorID]; ok && len(models) > 0 {
            // Sort models alphabetically
            sortedModels := make([]string, len(models))
            copy(sortedModels, models)
            sort.Strings(sortedModels)
            
            sb.modelSelect.Options = sortedModels
        } else {
            // No models available for this vendor
            sb.modelSelect.Options = []string{"No models available"}
        }
    } else {
        // No vendors loaded
        sb.modelSelect.Options = []string{"No models available"}
    }
    
    // Set handler for model selection
    sb.modelSelect.OnChanged = func(selected string) {
        // Skip if it's our placeholder text
        if selected == "No models available" || selected == "Loading models..." {
            return
        }
        
        app.state.CurrentModelID = selected
        app.state.CurrentModelName = selected // Simplify for now, could look up friendly name
        
        // Save to config
        if app.fabricConfig != nil {
            app.fabricConfig.SetConfig("DEFAULT_MODEL", selected)
            app.fabricConfig.SaveEnvConfig()
        }
        
        app.ShowMessage(fmt.Sprintf("Selected model: %s", selected))
    }
    
    // Set initial selection if we have a current model
    if app.state.CurrentModelID != "" && contains(sb.modelSelect.Options, app.state.CurrentModelID) {
        sb.modelSelect.SetSelected(app.state.CurrentModelID)
    } else if len(sb.modelSelect.Options) > 0 && sb.modelSelect.Options[0] != "No models available" && sb.modelSelect.Options[0] != "Loading models..." {
        // Default to first model if current one not found
        sb.modelSelect.SetSelected(sb.modelSelect.Options[0])
        app.state.CurrentModelID = sb.modelSelect.Options[0]
    }
    
    // Create vendor select with better UI
    sb.vendorSelect = widget.NewSelect([]string{"Loading vendors..."}, nil)
    
    // If we have vendors loaded, populate the vendor select
    if len(app.state.LoadedVendors) > 0 {
        // Sort vendors alphabetically for better UX
        vendorList := make([]string, len(app.state.LoadedVendors))
        copy(vendorList, app.state.LoadedVendors)
        sort.Strings(vendorList)
        
        sb.vendorSelect.Options = vendorList
        
        // Set handler for vendor selection with improved UI feedback
        sb.vendorSelect.OnChanged = func(selected string) {
            app.state.CurrentVendorID = selected
            
            // Show loading indicator in model dropdown
            loadingText := "Loading models..."
            sb.modelSelect.Options = []string{loadingText}
            sb.modelSelect.SetSelected(loadingText)
            sb.modelSelect.Refresh()
            
            // Expand the model section if it was collapsed
            if sb.modelSection != nil && !sb.modelSection.IsExpanded {
                sb.modelSection.SetExpanded(true)
            }
            
            // Save vendor to config
            if app.fabricConfig != nil {
                app.fabricConfig.SetConfig("DEFAULT_VENDOR", selected)
                app.fabricConfig.SaveEnvConfig()
            }
            
            app.ShowMessage(fmt.Sprintf("Selected vendor: %s", selected))
            
            // Load models for this vendor asynchronously to avoid UI freezing
            go func() {
                err := app.loadModelsForVendor(selected)
                
                // Update UI on the main thread to avoid concurrency issues
                // Update on the main thread using a simplified approach
                app.window.Canvas().Refresh(sb.modelSelect) // Force refresh the model select
                
                // Process results
                if err != nil {
                    // Show error in model dropdown
                    sb.modelSelect.Options = []string{"Error loading models"}
                    sb.modelSelect.SetSelected("Error loading models")
                    sb.modelSelect.Refresh()
                    app.ShowMessage(fmt.Sprintf("Error loading models for %s", selected))
                } else {
                    // Get the models
                    models, ok := app.state.LoadedModels[selected]
                    if !ok || len(models) == 0 {
                        // No models available
                        sb.modelSelect.Options = []string{"No models available"}
                        sb.modelSelect.SetSelected("No models available")
                        sb.modelSelect.Refresh()
                        app.ShowMessage(fmt.Sprintf("No models available for %s", selected))
                    } else {
                        // Sort models alphabetically
                        sortedModels := make([]string, len(models))
                        copy(sortedModels, models)
                        sort.Strings(sortedModels)
                        
                        // Update dropdown
                        sb.modelSelect.Options = sortedModels
                        if len(sortedModels) > 0 {
                            sb.modelSelect.SetSelected(sortedModels[0])
                            app.state.CurrentModelID = sortedModels[0]
                        }
                        sb.modelSelect.Refresh()
                        app.ShowMessage(fmt.Sprintf("Loaded %d models for %s", len(models), selected))
                    }
                }
            }()
        }
        
        // Set initial selection if we have a current vendor
        if app.state.CurrentVendorID != "" && contains(vendorList, app.state.CurrentVendorID) {
            sb.vendorSelect.SetSelected(app.state.CurrentVendorID)
        } else if len(vendorList) > 0 {
            // Default to first vendor if current one not found
            sb.vendorSelect.SetSelected(vendorList[0])
            app.state.CurrentVendorID = vendorList[0]
        }
    } else {
        // No vendors loaded, show a message
        sb.vendorSelect.Options = []string{"No vendors available"}
        sb.vendorSelect.SetSelected("No vendors available")
    }

        // Initialize collapsible sections - we'll populate them later
    sb.patternSection = nil
    sb.modelSection = nil
    sb.parameterSection = nil
    
        // Create pattern section with search and filter controls
    patternControls := container.NewVBox(
        widget.NewLabel("Search:"),
        sb.searchEntry,
        widget.NewLabel("Filter by tag:"),
        sb.patternFilter,
        widget.NewSeparator(),
        sb.patternList,
    )
    sb.patternSection = NewCollapsibleSection("Patterns", patternControls)
    sb.patternSection.SetExpanded(true) // Start expanded by default
    
    // Create model provider card
    modelProviderContent := container.NewVBox(
        widget.NewLabel("Provider:"),
        sb.vendorSelect,
        widget.NewSeparator(),
        widget.NewLabel("Model:"),
        sb.modelSelect,
    )
    sb.modelSection = NewCollapsibleSection("AI Model", modelProviderContent)
    
    // Create parameter settings section (placeholder for now)
    paramControls := container.NewVBox(
        widget.NewLabel("Temperature, Top-P, etc. will go here"),
    )
    sb.parameterSection = NewCollapsibleSection("Parameters", paramControls)
    
    // Assemble the sidebar with all sections
    sb.container = container.NewVBox(
        widget.NewLabelWithStyle("Fabric Pattern Studio", fyne.TextAlignCenter, fyne.TextStyle{Bold: true, Italic: true}),
        widget.NewSeparator(),
        sb.patternSection,
        widget.NewSeparator(),
        sb.modelSection,
        widget.NewSeparator(),
        sb.parameterSection,
    )
    return sb
}

// Helper function to check if a string is in a slice
func contains(slice []string, str string) bool {
    for _, item := range slice {
        if item == str {
            return true
        }
    }
    return false
}

// Container returns the root Fyne container for the SidebarPanel.
func (sb *SidebarPanel) Container() fyne.CanvasObject {
    return sb.container
}

// Helper function to get short description
func getShortDescription(pattern Pattern) string {
    desc := pattern.Description
    if len(desc) > 60 {
        return desc[:57] + "..."
    }
    return desc
}

// Filter patterns based on search text and selected filter
func filterPatterns(app *FabricApp) {
    searchText := app.state.SearchQuery
    selectedTags := app.state.SelectedTags
    
    // If no filter, show all patterns
    if searchText == "" && (selectedTags == nil || len(selectedTags) == 0) {
        app.state.FilteredPatterns = make([]Pattern, len(app.state.LoadedPatterns))
        copy(app.state.FilteredPatterns, app.state.LoadedPatterns)
        // Only refresh if mainLayout and Sidebar are initialized
        if app.mainLayout != nil && app.mainLayout.Sidebar != nil && app.mainLayout.Sidebar.patternList != nil {
            app.mainLayout.Sidebar.patternList.Refresh()
        }
        return
    }
    
    // Apply filters
    filtered := []Pattern{}
    searchLower := strings.ToLower(searchText)
    
    for _, p := range app.state.LoadedPatterns {
        // Check search text
        nameMatch := searchText == "" || strings.Contains(strings.ToLower(p.Name), searchLower)
        descMatch := searchText == "" || strings.Contains(strings.ToLower(p.Description), searchLower)
        systemMatch := searchText == "" || strings.Contains(strings.ToLower(p.SystemMD), searchLower)
        
        // Check tag filter
        tagMatch := selectedTags == nil || len(selectedTags) == 0
        if !tagMatch && len(p.Tags) > 0 {
            for _, selectedTag := range selectedTags {
                for _, patternTag := range p.Tags {
                    if patternTag == selectedTag {
                        tagMatch = true
                        break
                    }
                }
                if tagMatch {
                    break
                }
            }
        }
        
        // Add pattern if it matches all filters
        if (nameMatch || descMatch || systemMatch) && tagMatch {
            filtered = append(filtered, p)
        }
    }
    
    app.state.FilteredPatterns = filtered
    
    // Only refresh if mainLayout and Sidebar are initialized
    if app.mainLayout != nil && app.mainLayout.Sidebar != nil && app.mainLayout.Sidebar.patternList != nil {
        app.mainLayout.Sidebar.patternList.Refresh()
        
        // Update status
        if len(filtered) == 0 {
            app.ShowMessage("No patterns match the current filters")
        } else {
            app.ShowMessage(fmt.Sprintf("Showing %d/%d patterns", len(filtered), len(app.state.LoadedPatterns)))
        }
    }
}

// extractTagOptions builds a list of tag options from loaded patterns
func extractTagOptions(patterns []Pattern) []string {
    // Use a map to deduplicate tags
    tagMap := make(map[string]bool)
    tagMap["All"] = true // Always include "All" option
    
    for _, pattern := range patterns {
        for _, tag := range pattern.Tags {
            if tag != "" { // Skip empty tags
                tagMap[tag] = true
            }
        }
    }
    
    // Convert map to sorted slice
    tags := make([]string, 0, len(tagMap))
    for tag := range tagMap {
        tags = append(tags, tag)
    }
    
    // Sort tags (with "All" at the front)
    sort.Slice(tags, func(i, j int) bool {
        if tags[i] == "All" {
            return true
        }
        if tags[j] == "All" {
            return false
        }
        return tags[i] < tags[j]
    })
    
    return tags
}

// addToRecentPatterns adds a pattern ID to the recent patterns list
func addToRecentPatterns(app *FabricApp, patternID string) {
    // Check if already in list
    for i, id := range app.state.LastUsedPatterns {
        if id == patternID {
            // If already at front, do nothing
            if i == 0 {
                return
            }
            // Remove from current position
            app.state.LastUsedPatterns = append(app.state.LastUsedPatterns[:i], app.state.LastUsedPatterns[i+1:]...)
            break
        }
    }
    
    // Add to front of list
    app.state.LastUsedPatterns = append([]string{patternID}, app.state.LastUsedPatterns...)
    
    // Limit list size to 10
    if len(app.state.LastUsedPatterns) > 10 {
        app.state.LastUsedPatterns = app.state.LastUsedPatterns[:10]
    }
}

// MainContentPanel mirrors Streamlit's main content area with tabs for Input, Output, and Pattern Info.
type MainContentPanel struct {
    app *FabricApp // Reference to the main app

    container *fyne.Container
    tabs      *container.AppTabs

    // Sub-panels for each tab's content
    inputArea       *InputArea
    outputArea      *OutputArea
    patternInfoArea *PatternInfoArea

    // Directly accessible widgets from the Execute context
    runButton *widget.Button
}

// NewMainContentPanel creates a new main content panel.
func NewMainContentPanel(app *FabricApp) *MainContentPanel {
    mcp := &MainContentPanel{app: app}

    // Create sub-areas
    mcp.inputArea = NewInputArea(app)
    mcp.outputArea = NewOutputArea(app)
    mcp.patternInfoArea = NewPatternInfoArea(app)

    // Create Run button (now placed here for global access)
    mcp.runButton = widget.NewButtonWithIcon("Run Pattern", theme.MediaPlayIcon(), func() {
        mcp.executePattern() // Call the execution logic
    })
    mcp.runButton.Importance = widget.MediumImportance

    // Create tabs for main content with workflow-oriented names
    mcp.tabs = container.NewAppTabs(
        container.NewTabItem("Execute", mcp.inputArea.Container()),
        container.NewTabItem("Results", mcp.outputArea.Container()),
        container.NewTabItem("Pattern Details", mcp.patternInfoArea.Container()),
    )
    mcp.tabs.SetTabLocation(container.TabLocationTop)

    // Main layout assembly: Run button at top, tabs in center
    mcp.container = container.NewBorder(
        container.NewVBox(mcp.runButton), // Top section with the run button
        nil,                          // Bottom
        nil, nil,                     // Left, Right
        mcp.tabs,                     // Center: the tabbed content
    )
    return mcp
}

// Container returns the root Fyne container for the MainContentPanel.
func (mcp *MainContentPanel) Container() fyne.CanvasObject {
    return mcp.container
}

// UpdateRunButton updates the text of the main Run button.
func (mcp *MainContentPanel) UpdateRunButton(patternName string) {
    if patternName == "" {
        mcp.runButton.SetText("Run Pattern")
        mcp.runButton.Importance = widget.MediumImportance
    } else {
        mcp.runButton.SetText(fmt.Sprintf("Run '%s'", patternName))
        mcp.runButton.Importance = widget.HighImportance
    }
    mcp.runButton.Refresh()
}

// executePattern is the core logic for running a pattern.
func (mcp *MainContentPanel) executePattern() {
    // Validate pattern selection
    if mcp.app.state.CurrentPatternID == "" {
        mcp.app.ShowError(fmt.Errorf("no pattern selected"))
        return
    }

    // Get input text based on selected input source
    var input string
    switch mcp.app.state.InputSourceType {
    case "Text":
        input = mcp.inputArea.inputEntry.Text
    case "URL":
        input = mcp.inputArea.urlEntry.Text
    // Other input types will be handled here
    default:
        input = mcp.inputArea.inputEntry.Text
    }
    
    if input == "" {
        mcp.app.ShowError(fmt.Errorf("no input provided"))
        return
    }

    // Save current input to state
    mcp.app.state.CurrentInputText = input
    
    // Update UI to show execution is in progress
    mcp.app.ShowMessage("Executing pattern: " + mcp.app.state.CurrentPatternID)
    mcp.runButton.Disable() // Disable button during execution
    mcp.outputArea.outputEntry.SetText("Thinking...") // Clear previous output and show feedback
    
    // Build execution configuration from app state
    config := ExecutionConfig{
        PatternID:        mcp.app.state.CurrentPatternID,
        Input:            input,
        Model:            mcp.app.state.CurrentModelID,
        Vendor:           mcp.app.state.CurrentVendorID,
        Temperature:      mcp.app.state.Temperature,
        TopP:             mcp.app.state.TopP,
        PresencePenalty:  mcp.app.state.PresencePenalty,
        FrequencyPenalty: mcp.app.state.FrequencyPenalty,
        Seed:             mcp.app.state.Seed,
        ContextLength:    mcp.app.state.ContextLength,
        Strategy:         mcp.app.state.Strategy,
        Stream:           false, // Don't stream by default
        DryRun:           false, // Don't use dry run by default
    }
    
    // Execute in goroutine to avoid blocking UI
    go func() {
        // Ensure the button is re-enabled when execution completes
        defer func() {
            mcp.runButton.Enable()
            mcp.runButton.Refresh()
        }()
        
        // Execute the pattern
        result, err := mcp.app.executePattern(config)
        if err != nil {
            mcp.app.ShowError(fmt.Errorf("execution failed: %v", err))
            mcp.outputArea.outputEntry.SetText(fmt.Sprintf("Error: %v", err))
            return
        }
        
        // Display the result
        mcp.outputArea.outputEntry.SetText(result.Output)
        mcp.app.state.LastOutput = result.Output
        
        // Update status with execution info
        mcp.app.ShowMessage(fmt.Sprintf("Pattern executed in %.2f seconds", 
            result.ExecutionTime.Seconds()))
        
        // Save execution to history
        mcp.app.state.LastRun = result.Timestamp
        
        // Automatically switch to Results tab after execution
        mcp.tabs.SelectIndex(1) // Results tab is index 1
        
        // Update output info with execution timestamp
        mcp.outputArea.outputInfo.SetText(fmt.Sprintf(
            "Executed at: %s - Time: %.2f seconds", 
            result.Timestamp.Format("Jan 2, 2006 15:04:05"),
            result.ExecutionTime.Seconds()))
    }()
}

// StatusBar shows application status.
type StatusBar struct {
    content *widget.Label
}

// NewStatusBar creates a new status bar.
func NewStatusBar() *StatusBar {
    return &StatusBar{
        content: widget.NewLabel("Ready"),
    }
}

// Container returns the status bar content
func (s *StatusBar) Container() fyne.CanvasObject {
    return s.content
}

// ShowError displays an error message.
func (s *StatusBar) ShowError(err error) {
    s.content.SetText("Error: " + err.Error())
    s.content.Refresh() // Ensure update is drawn
}

// ShowMessage displays a status message.
func (s *StatusBar) ShowMessage(msg string) {
    s.content.SetText(msg)
    s.content.Refresh() // Ensure update is drawn
}

// InputArea represents the input section.
type InputArea struct {
    app *FabricApp // Reference to the main app

    container *fyne.Container
    // Main input widgets
    sourceSelect *widget.Select // Text, Clipboard, File, URL
    inputEntry   *widget.Entry
    clipboardBtn *widget.Button
    fileBtn      *widget.Button
    urlEntry     *widget.Entry
    
    // Input statistics and preview
    previewLabel *widget.Label
    charCountLabel *widget.Label
    wordCountLabel *widget.Label
    statsContainer *fyne.Container
}

// NewInputArea creates a new input area.
func NewInputArea(app *FabricApp) *InputArea {
    ia := &InputArea{app: app}

    ia.sourceSelect = widget.NewSelect([]string{"Text", "Clipboard", "File", "URL"}, nil) // OnChanged set later
    ia.inputEntry = widget.NewMultiLineEntry()
    ia.inputEntry.SetPlaceHolder("Enter your input here...")
    ia.inputEntry.SetMinRowsVisible(8)
    
    // Add text change listener to update statistics
    ia.inputEntry.OnChanged = func(text string) {
        ia.updateTextStatistics(text)
    }

    // Setup clipboard paste button with proper action
    ia.clipboardBtn = widget.NewButton("Paste from Clipboard", func() {
        // This would need OS-specific clipboard implementation
        // For now just a placeholder
        ia.app.ShowMessage("Clipboard paste not implemented yet")
    })
    
    ia.fileBtn = widget.NewButton("Choose File...", func() {
        // This would open a file dialog
        // For now just a placeholder
        ia.app.ShowMessage("File selection not implemented yet")
    })
    
    ia.urlEntry = widget.NewEntry()
    ia.urlEntry.SetPlaceHolder("Enter URL here...")

    // Create stats display area
    ia.previewLabel = widget.NewLabelWithStyle("Input Preview", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
    ia.charCountLabel = widget.NewLabel("Characters: 0")
    ia.wordCountLabel = widget.NewLabel("Words: 0")
    
    statsBox := container.NewHBox(
        ia.charCountLabel,
        widget.NewSeparator(),
        ia.wordCountLabel,
    )
    
    ia.statsContainer = container.NewVBox(
        ia.previewLabel,
        statsBox,
    )
    
    // Initial visibility controlled by updateInputSource
    ia.clipboardBtn.Hide()
    ia.fileBtn.Hide()
    ia.urlEntry.Hide()

    ia.container = container.NewVBox(
        widget.NewLabelWithStyle("Input Source:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
        ia.sourceSelect,
        widget.NewSeparator(),
        ia.inputEntry,
        ia.clipboardBtn,
        ia.fileBtn,
        ia.urlEntry,
        widget.NewSeparator(),
        ia.statsContainer)


    // Set default selection and trigger initial UI update
    ia.sourceSelect.SetSelected("Text")
    ia.updateInputSource("Text")

    ia.sourceSelect.OnChanged = func(source string) {
        ia.updateInputSource(source)
    }

    return ia
}

// Container returns the root Fyne container for the InputArea.
func (ia *InputArea) Container() fyne.CanvasObject {
    return ia.container
}

// updateTextStatistics updates the character and word count statistics
func (ia *InputArea) updateTextStatistics(text string) {
    // Update character count
    charCount := len(text)
    ia.charCountLabel.SetText(fmt.Sprintf("Characters: %d", charCount))
    
    // Update word count (simple split by whitespace)
    words := strings.Fields(text)
    wordCount := len(words)
    ia.wordCountLabel.SetText(fmt.Sprintf("Words: %d", wordCount))
    
    // Save to app state
    ia.app.state.CurrentInputText = text
}

// updateInputSource manages the visibility of input widgets.
func (ia *InputArea) updateInputSource(source string) {
    ia.inputEntry.Hide()
    ia.clipboardBtn.Hide()
    ia.fileBtn.Hide()
    ia.urlEntry.Hide()

    switch source {
    case "Text":
        ia.inputEntry.Show()
    case "Clipboard":
        ia.clipboardBtn.Show()
    case "File":
        ia.fileBtn.Show()
    case "URL":
        ia.urlEntry.Show()
    }
    ia.container.Refresh() // Refresh to apply visibility changes
}

// OutputArea represents the output section.
type OutputArea struct {
    app *FabricApp // Reference to the main app

    container *fyne.Container
    // Main output widgets
    outputEntry *widget.Entry
    formatSelect *widget.Select // Text, Markdown, JSON
    copyBtn     *widget.Button
    saveBtn     *widget.Button
    starBtn     *widget.Button // Star/favorite this output
    outputInfo  *widget.Label  // Shows info about pattern run
}

// NewOutputArea creates a new output area.
func NewOutputArea(app *FabricApp) *OutputArea {
    oa := &OutputArea{app: app}

    oa.outputEntry = widget.NewMultiLineEntry()
    oa.outputEntry.Disable()
    oa.outputEntry.SetMinRowsVisible(10)
    oa.outputEntry.SetPlaceHolder("AI output will appear here...")

    oa.formatSelect = widget.NewSelect([]string{"Text", "Markdown", "JSON"}, nil) 
    oa.formatSelect.SetSelected("Text")
    
    // Add format change handler
    oa.formatSelect.OnChanged = func(format string) {
        // Save the format preference in app state
        app.state.OutputFormat = format
        oa.app.ShowMessage(fmt.Sprintf("Output format set to %s", format))
        // Future: implement format conversion
    }
    
    // Create info label for pattern execution metadata
    oa.outputInfo = widget.NewLabel("Ready for execution")
    oa.outputInfo.Importance = widget.MediumImportance

    // Set up action buttons with icons
    oa.copyBtn = widget.NewButtonWithIcon("Copy", theme.ContentCopyIcon(), func() {
        // OS-specific clipboard implementation would go here
        app.ShowMessage("Output copied to clipboard")
    })
    
    oa.saveBtn = widget.NewButtonWithIcon("Save", theme.DocumentSaveIcon(), func() {
        // File save dialog would go here
        app.ShowMessage("Save dialog not implemented yet")
    })
    
    oa.starBtn = widget.NewButtonWithIcon("Favorite", theme.InfoIcon(), func() {
        // Create custom dialog for star name
        nameEntry := widget.NewEntry()
        nameEntry.SetText(fmt.Sprintf("Starred Output #%d", len(app.state.StarredOutputs)+1))
        // Select all text - calling SetText already focuses the field
        // nameEntry.SelectAll() // This method doesn't exist in this version of Fyne
        
        dialog := dialog.NewForm(
            "Save Favorite Output", 
            "Save", "Cancel",
            []*widget.FormItem{
                widget.NewFormItem("Name", nameEntry),
            },
            func(save bool) {
                if save {
                    app.StarOutput(nameEntry.Text)
                }
            },
            app.window)
        dialog.Show()
    })

    // Create button row with proper spacing
    buttonRow := container.NewHBox(
        oa.copyBtn, 
        widget.NewSeparator(), 
        oa.saveBtn,
        widget.NewSeparator(),
        oa.starBtn,
    )

    // Assemble the container with proper sections and separators
    oa.container = container.NewVBox(
        widget.NewLabelWithStyle("Results", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
        oa.outputInfo,
        widget.NewSeparator(),
        widget.NewLabelWithStyle("Output Format:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
        oa.formatSelect,
        widget.NewSeparator(),
        oa.outputEntry,
        widget.NewSeparator(),
        buttonRow,
    )
    return oa
}

// Container returns the root Fyne container for the OutputArea.
func (oa *OutputArea) Container() fyne.CanvasObject {
    return oa.container
}

// PatternInfoArea represents the pattern details display.
type PatternInfoArea struct {
    app *FabricApp // Reference to the main app

    container *fyne.Container
    // Pattern details widgets
    nameLabel    *widget.Label
    modelLabel   *widget.Label
    description  *widget.Entry
    systemPrompt *widget.Entry
    userPrompt   *widget.Entry
    tagsLabel    *widget.Label
}

// NewPatternInfoArea creates a new pattern info area.
func NewPatternInfoArea(app *FabricApp) *PatternInfoArea {
    pia := &PatternInfoArea{app: app}

    // Pattern name (bold)
    pia.nameLabel = widget.NewLabel("No Pattern Selected")
    pia.nameLabel.TextStyle = fyne.TextStyle{Bold: true}
    
    // Model and vendor info
    pia.modelLabel = widget.NewLabel("Model: Not set")
    pia.modelLabel.Importance = widget.HighImportance
    
    // Tags display
    pia.tagsLabel = widget.NewLabel("Tags: none")
    pia.tagsLabel.Importance = widget.MediumImportance

    // Description (read-only)
    pia.description = widget.NewMultiLineEntry()
    pia.description.Disable()
    pia.description.SetMinRowsVisible(3)
    pia.description.SetPlaceHolder("Pattern description")

    // System prompt (read-only)
    pia.systemPrompt = widget.NewMultiLineEntry()
    pia.systemPrompt.Disable()
    pia.systemPrompt.SetMinRowsVisible(10)
    pia.systemPrompt.SetPlaceHolder("System prompt content will be displayed here")

    // User prompt (read-only, optional for many patterns)
    pia.userPrompt = widget.NewMultiLineEntry()
    pia.userPrompt.Disable()
    pia.userPrompt.SetMinRowsVisible(5)
    pia.userPrompt.SetPlaceHolder("User prompt content will be displayed here (if present)")

    // Layout with scrolling for long prompts
    headerSection := container.NewVBox(
        pia.nameLabel,
        pia.modelLabel,
        pia.tagsLabel,
        widget.NewLabel("Description:"),
        pia.description,
    )
    
    promptSection := container.NewVBox(
        widget.NewSeparator(),
        widget.NewLabelWithStyle("System Prompt:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
        pia.systemPrompt,
        widget.NewLabelWithStyle("User Prompt:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
        pia.userPrompt,
    )
    
    // Use a scroll container for the whole view
    scrollContainer := container.NewVScroll(
        container.NewVBox(headerSection, promptSection),
    )
    pia.container = container.NewMax(scrollContainer)
    
    return pia
}

// Container returns the root Fyne container for the PatternInfoArea.
func (pia *PatternInfoArea) Container() fyne.CanvasObject {
    return pia.container
}

// UpdateInfo updates the pattern info display.
func (pia *PatternInfoArea) UpdateInfo(patternName, modelName, vendorName string) {
    if patternName == "" {
        pia.nameLabel.SetText("No Pattern Selected")
        pia.modelLabel.SetText("Model: Not set")
        pia.tagsLabel.SetText("Tags: none")
        pia.description.SetText("")
        pia.systemPrompt.SetText("")
        pia.userPrompt.SetText("")
    } else {
        pia.nameLabel.SetText(patternName)
        
        // Model info
        if modelName != "" {
            if vendorName != "" {
                pia.modelLabel.SetText(fmt.Sprintf("Model: %s (%s)", modelName, vendorName))
            } else {
                pia.modelLabel.SetText(fmt.Sprintf("Model: %s", modelName))
            }
        } else {
            pia.modelLabel.SetText("Model: Not set")
        }
        
        // Find pattern to display details
        for _, p := range pia.app.state.LoadedPatterns {
            if p.Name == patternName || p.ID == pia.app.state.CurrentPatternID {
                pia.SetPattern(p)
                return
            }
        }
        
        // Pattern not found, use placeholder
        pia.description.SetText("Pattern details not available")
        pia.systemPrompt.SetText("")
        pia.userPrompt.SetText("")
        pia.tagsLabel.SetText("Tags: none")
    }
}

// SetPattern sets the content of this panel to display a pattern's details
func (pia *PatternInfoArea) SetPattern(p Pattern) {
    pia.nameLabel.SetText(p.Name)
    pia.description.SetText(p.Description)
    pia.systemPrompt.SetText(p.SystemMD)
    pia.userPrompt.SetText(p.UserMD)
    
    // Format tags
    if len(p.Tags) > 0 {
        pia.tagsLabel.SetText("Tags: " + strings.Join(p.Tags, ", "))
    } else {
        pia.tagsLabel.SetText("Tags: none")
    }
    
    // Model info from app state
    if pia.app.state.CurrentModelName != "" {
        pia.modelLabel.SetText(fmt.Sprintf("Model: %s (%s)", 
            pia.app.state.CurrentModelName, pia.app.state.CurrentVendorID))
    }
}