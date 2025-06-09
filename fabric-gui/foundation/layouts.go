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
    modelProvider *ModelProviderPanel
    
    // Parameter Settings
    parameterSection *CollapsibleSection // Collapsible parameters section
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
                
                // Get the labels from the container
                vbox := obj.(*fyne.Container)
                nameLabel := vbox.Objects[0].(*widget.Label)
                descLabel := vbox.Objects[1].(*widget.Label)
                
                // Update labels
                nameLabel.SetText(pattern.Name)
                
                // Truncate description if too long
                desc := pattern.Description
                if len(desc) > 80 {
                    desc = desc[:77] + "..."
                }
                descLabel.SetText(desc)
            }
        },
    )
    
    // Set up pattern selection handler
    sb.patternList.OnSelected = func(id widget.ListItemID) {
        if id < len(app.state.FilteredPatterns) {
            pattern := app.state.FilteredPatterns[id]
            
            // Update app state with selected pattern
            app.state.CurrentPatternID = pattern.ID
            
            // Update UI to show pattern is selected
            app.mainLayout.MainContent.patternInfoArea.UpdateInfo(
                pattern.Name,
                app.state.CurrentModelName,
                app.state.CurrentVendorID,
            )
            
            // Update run button with pattern name
            app.mainLayout.MainContent.UpdateRunButton(pattern.Name)
            
            // Switch to Execute tab if not already there
            if app.state.LastActiveTab != "Execute" {
                app.mainLayout.MainContent.tabs.Select(0) // Execute tab
            }
            
            // Show message
            app.ShowMessage(fmt.Sprintf("Selected pattern: %s", pattern.Name))
            
            // Deselect after a moment (visual feedback but don't stay highlighted)
            go func() {
                time.Sleep(100 * time.Millisecond)
                app.mainWindow.Canvas().Focus(nil) // Remove focus
            }()
        }
    }

    // Create model provider panel (handles all model/vendor selection)
    sb.modelProvider = NewModelProviderPanel(app)

    // Initialize collapsible sections
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
        sb.modelProvider.Container(), // Use the ModelProviderPanel container
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

// extractTagOptions builds a list of unique tags from patterns
func extractTagOptions(patterns []Pattern) []string {
    // Start with "All" option
    options := []string{"All"}
    
    // Create a map to track unique tags
    tagMap := make(map[string]bool)
    
    // Extract tags from all patterns
    for _, pattern := range patterns {
        for _, tag := range pattern.Tags {
            tagMap[tag] = true
        }
    }
    
    // Convert map keys to slice
    for tag := range tagMap {
        options = append(options, tag)
    }
    
    // Sort options for better UX (keep "All" at the beginning)
    sort.Slice(options[1:], func(i, j int) bool {
        return strings.ToLower(options[i+1]) < strings.ToLower(options[j+1])
    })
    
    return options
}

// filterPatterns applies current search query and tag filters
func filterPatterns(app *FabricApp) {
    // Start with all patterns
    filteredPatterns := make([]Pattern, 0)
    
    // Filter by search query
    searchQuery := strings.ToLower(app.state.SearchQuery)
    
    for _, pattern := range app.state.LoadedPatterns {
        // Skip if doesn't match search query
        if searchQuery != "" {
            nameMatch := strings.Contains(strings.ToLower(pattern.Name), searchQuery)
            descMatch := strings.Contains(strings.ToLower(pattern.Description), searchQuery)
            
            if !nameMatch && !descMatch {
                continue
            }
        }
        
        // Skip if doesn't match tag filter
        if len(app.state.SelectedTags) > 0 {
            tagMatch := false
            for _, selectedTag := range app.state.SelectedTags {
                for _, patternTag := range pattern.Tags {
                    if patternTag == selectedTag {
                        tagMatch = true
                        break
                    }
                }
                if tagMatch {
                    break
                }
            }
            
            if !tagMatch {
                continue
            }
        }
        
        // Pattern passed all filters
        filteredPatterns = append(filteredPatterns, pattern)
    }
    
    // Update app state
    app.state.FilteredPatterns = filteredPatterns
    
    // Refresh pattern list UI
    if app.mainLayout != nil && app.mainLayout.Sidebar != nil {
        app.mainLayout.Sidebar.patternList.Refresh()
    }
}

// MainContentPanel manages the main content area with tabs.
type MainContentPanel struct {
    app *FabricApp // Reference to the main app

    container *fyne.Container
    tabs      *container.AppTabs
    
    // Tab content panels
    inputArea       *InputArea
    outputArea      *OutputArea
    patternInfoArea *PatternInfoArea
    
    // Action buttons
    runButton *widget.Button
}

// NewMainContentPanel creates a new main content panel with tabs.
func NewMainContentPanel(app *FabricApp) *MainContentPanel {
    mc := &MainContentPanel{app: app}
    
    // Create input area (for Execute tab)
    mc.inputArea = NewInputArea(app)
    
    // Create output area (for Results tab)
    mc.outputArea = NewOutputArea(app)
    
    // Create pattern info area (for Pattern Details tab)
    mc.patternInfoArea = NewPatternInfoArea(app)
    
    // Create run button
    mc.runButton = widget.NewButton("Run Pattern", func() {
        mc.executePattern()
    })
    mc.runButton.Importance = widget.HighImportance
    mc.runButton.Disable() // Disabled until pattern is selected
    
    // Create Execute tab with input area and run button
    executeContent := container.NewBorder(
        nil,                  // Top
        container.NewVBox(    // Bottom
            widget.NewSeparator(),
            container.NewHBox(
                mc.runButton,
                widget.NewLabel(""), // Spacer
            ),
        ),
        nil, nil,             // Left, Right
        mc.inputArea.Container(), // Center
    )
    
    // Create tabs
    mc.tabs = container.NewAppTabs(
        container.NewTabItem("Execute", executeContent),
        container.NewTabItem("Results", mc.outputArea.Container()),
        container.NewTabItem("Pattern Details", mc.patternInfoArea.Container()),
    )
    
    // Set initial tab
    mc.tabs.SetTabLocation(container.TabLocationTop)
    
    // Create main container
    mc.container = container.NewMax(mc.tabs)
    
    return mc
}

// Container returns the root Fyne container for the MainContentPanel.
func (mc *MainContentPanel) Container() fyne.CanvasObject {
    return mc.container
}

// UpdateRunButton updates the run button text and state based on pattern selection.
func (mc *MainContentPanel) UpdateRunButton(patternName string) {
    if patternName == "" {
        mc.runButton.SetText("Run Pattern")
        mc.runButton.Disable()
    } else {
        mc.runButton.SetText(fmt.Sprintf("Run '%s'", patternName))
        mc.runButton.Enable()
    }
}

// executePattern runs the currently selected pattern.
func (mc *MainContentPanel) executePattern() {
    // Get current pattern and input
    patternID := mc.app.state.CurrentPatternID
    if patternID == "" {
        mc.app.ShowError("No pattern selected")
        return
    }
    
    // Get input text
    input := mc.inputArea.GetInput()
    if input == "" {
        mc.app.ShowError("Input is empty")
        return
    }
    
    // Get current model and vendor
    modelID := mc.app.state.CurrentModelID
    vendorID := mc.app.state.CurrentVendorID
    
    if modelID == "" || vendorID == "" {
        mc.app.ShowError("No model or vendor selected")
        return
    }
    
    // Show execution in progress
    mc.runButton.Disable()
    mc.runButton.SetText("Executing...")
    mc.app.StatusBar.ShowMessage("Executing pattern...")
    
    // Execute pattern asynchronously
    go func() {
        // Get pattern by ID
        var pattern Pattern
        found := false
        for _, p := range mc.app.state.LoadedPatterns {
            if p.ID == patternID {
                pattern = p
                found = true
                break
            }
        }
        
        if !found {
            mc.app.ShowError("Pattern not found")
            mc.runButton.Enable()
            mc.runButton.SetText(fmt.Sprintf("Run '%s'", mc.app.getPatternNameByID(patternID)))
            return
        }
        
        // Create execution manager
        execManager := NewExecutionManager(mc.app.fabricConfig)
        
        // Execute pattern
        result, err := execManager.ExecutePattern(pattern, input, modelID, vendorID)
        
        // Update UI on main thread
        fyne.CurrentApp().Driver().RunOnMain(func() {
            if err != nil {
                mc.app.ShowError(fmt.Sprintf("Execution failed: %v", err))
                mc.outputArea.SetOutput("Execution failed: " + err.Error())
            } else {
                // Update output area
                mc.outputArea.SetOutput(result)
                
                // Update state
                mc.app.state.LastOutput = result
                mc.app.state.LastRun = time.Now()
                
                // Show success message
                mc.app.StatusBar.ShowMessage("Execution completed successfully")
                
                // Switch to Results tab
                mc.tabs.Select(1) // Results tab
            }
            
            // Re-enable run button
            mc.runButton.Enable()
            mc.runButton.SetText(fmt.Sprintf("Run '%s'", pattern.Name))
        })
    }()
}

// InputArea manages the input area for pattern execution.
type InputArea struct {
    app *FabricApp // Reference to the main app

    container *fyne.Container
    
    // Input components
    inputSource *widget.Select
    textInput   *widget.TextArea
    fileInput   *widget.Button
    urlInput    *widget.Entry
    
    // Preview components
    previewLabel *widget.Label
    previewStats *widget.Label
}

// NewInputArea creates a new input area.
func NewInputArea(app *FabricApp) *InputArea {
    ia := &InputArea{app: app}
    
    // Create input source selector
    ia.inputSource = widget.NewSelect([]string{"Text", "File", "URL"}, func(selected string) {
        ia.updateInputSource(selected)
    })
    ia.inputSource.SetSelected("Text")
    
    // Create text input
    ia.textInput = widget.NewMultiLineEntry()
    ia.textInput.SetPlaceHolder("Enter text here...")
    ia.textInput.OnChanged = func(text string) {
        ia.updatePreview()
    }
    
    // Create file input button
    ia.fileInput = widget.NewButton("Select File", func() {
        dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
            if err != nil {
                app.ShowError(fmt.Sprintf("Error opening file: %v", err))
                return
            }
            if reader == nil {
                return // User cancelled
            }
            
            // TODO: Read file content
            // For now, just show filename
            ia.textInput.SetText(fmt.Sprintf("File: %s", reader.URI().Name()))
            ia.updatePreview()
        }, app.mainWindow)
    })
    ia.fileInput.Hide() // Hidden initially
    
    // Create URL input
    ia.urlInput = widget.NewEntry()
    ia.urlInput.SetPlaceHolder("Enter URL here...")
    ia.urlInput.OnChanged = func(url string) {
        ia.updatePreview()
    }
    ia.urlInput.Hide() // Hidden initially
    
    // Create preview components
    ia.previewLabel = widget.NewLabel("Input Preview")
    ia.previewStats = widget.NewLabel("Characters: 0  Words: 0")
    
    // Create input source section
    inputSourceSection := container.NewVBox(
        widget.NewLabel("Input Source:"),
        ia.inputSource,
    )
    
    // Create input content section
    inputContentSection := container.NewVBox(
        ia.textInput,
        ia.fileInput,
        ia.urlInput,
    )
    
    // Create preview section
    previewSection := container.NewVBox(
        widget.NewSeparator(),
        widget.NewLabelWithStyle("Input Preview", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
        ia.previewStats,
    )
    
    // Assemble the input area
    ia.container = container.NewVBox(
        inputSourceSection,
        inputContentSection,
        previewSection,
    )
    
    return ia
}

// Container returns the root Fyne container for the InputArea.
func (ia *InputArea) Container() fyne.CanvasObject {
    return ia.container
}

// GetInput returns the current input text.
func (ia *InputArea) GetInput() string {
    switch ia.inputSource.Selected {
    case "Text":
        return ia.textInput.Text
    case "File":
        // TODO: Implement file reading
        return ia.textInput.Text // For now, return placeholder
    case "URL":
        // TODO: Implement URL fetching
        return ia.urlInput.Text // For now, return URL
    default:
        return ""
    }
}

// updateInputSource updates the UI based on the selected input source.
func (ia *InputArea) updateInputSource(source string) {
    // Hide all input components first
    ia.textInput.Hide()
    ia.fileInput.Hide()
    ia.urlInput.Hide()
    
    // Show the selected input component
    switch source {
    case "Text":
        ia.textInput.Show()
    case "File":
        ia.fileInput.Show()
    case "URL":
        ia.urlInput.Show()
    }
    
    // Update preview
    ia.updatePreview()
}

// updatePreview updates the preview stats.
func (ia *InputArea) updatePreview() {
    input := ia.GetInput()
    
    // Count characters
    charCount := len(input)
    
    // Count words
    wordCount := 0
    if input != "" {
        wordCount = len(strings.Fields(input))
    }
    
    // Update stats label
    ia.previewStats.SetText(fmt.Sprintf("Characters: %d  Words: %d", charCount, wordCount))
}

// OutputArea manages the output display for pattern execution results.
type OutputArea struct {
    app *FabricApp // Reference to the main app

    container *fyne.Container
    
    // Output components
    outputInfo *widget.Label
    outputText *widget.TextArea
    
    // Action buttons
    copyButton *widget.Button
    saveButton *widget.Button
    clearButton *widget.Button
}

// NewOutputArea creates a new output area.
func NewOutputArea(app *FabricApp) *OutputArea {
    oa := &OutputArea{app: app}
    
    // Create output info label
    oa.outputInfo = widget.NewLabel("No output yet")
    
    // Create output text area
    oa.outputText = widget.NewMultiLineEntry()
    oa.outputText.SetPlaceHolder("Output will appear here...")
    oa.outputText.Disable() // Read-only
    
    // Create action buttons
    oa.copyButton = widget.NewButtonWithIcon("Copy", theme.ContentCopyIcon(), func() {
        app.mainWindow.Clipboard().SetContent(oa.outputText.Text)
        app.ShowMessage("Output copied to clipboard")
    })
    
    oa.saveButton = widget.NewButtonWithIcon("Save", theme.DocumentSaveIcon(), func() {
        dialog.ShowFileSave(func(writer fyne.URIWriteCloser, err error) {
            if err != nil {
                app.ShowError(fmt.Sprintf("Error saving file: %v", err))
                return
            }
            if writer == nil {
                return // User cancelled
            }
            
            // Write output to file
            _, err = writer.Write([]byte(oa.outputText.Text))
            writer.Close()
            
            if err != nil {
                app.ShowError(fmt.Sprintf("Error writing to file: %v", err))
                return
            }
            
            app.ShowMessage(fmt.Sprintf("Output saved to %s", writer.URI().Name()))
        }, app.mainWindow)
    })
    
    oa.clearButton = widget.NewButtonWithIcon("Clear", theme.DeleteIcon(), func() {
        oa.outputText.SetText("")
        oa.outputInfo.SetText("Output cleared")
        app.state.LastOutput = ""
    })
    
    // Create action button container
    actionButtons := container.NewHBox(
        oa.copyButton,
        oa.saveButton,
        oa.clearButton,
    )
    
    // Assemble the output area
    oa.container = container.NewBorder(
        oa.outputInfo,        // Top
        actionButtons,        // Bottom
        nil, nil,             // Left, Right
        oa.outputText,        // Center
    )
    
    return oa
}

// Container returns the root Fyne container for the OutputArea.
func (oa *OutputArea) Container() fyne.CanvasObject {
    return oa.container
}

// SetOutput sets the output text and updates the UI.
func (oa *OutputArea) SetOutput(output string) {
    oa.outputText.SetText(output)
    oa.outputInfo.SetText(fmt.Sprintf("Last executed: %s", time.Now().Format("Jan 2, 2006 15:04:05")))
    
    // Enable buttons if output is not empty
    if output == "" {
        oa.copyButton.Disable()
        oa.saveButton.Disable()
        oa.clearButton.Disable()
    } else {
        oa.copyButton.Enable()
        oa.saveButton.Enable()
        oa.clearButton.Enable()
    }
}

// PatternInfoArea displays details about the selected pattern.
type PatternInfoArea struct {
    app *FabricApp // Reference to the main app

    container *fyne.Container
    
    // Pattern info components
    nameLabel       *widget.Label
    descriptionText *widget.TextArea
    tagsLabel       *widget.Label
    
    // System and user prompts
    systemPromptText *widget.TextArea
    userPromptText   *widget.TextArea
    
    // Model info
    modelInfoLabel *widget.Label
}

// NewPatternInfoArea creates a new pattern info area.
func NewPatternInfoArea(app *FabricApp) *PatternInfoArea {
    pia := &PatternInfoArea{app: app}
    
    // Create pattern info components
    pia.nameLabel = widget.NewLabelWithStyle("No pattern selected", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
    
    pia.descriptionText = widget.NewMultiLineEntry()
    pia.descriptionText.SetPlaceHolder("Pattern description will appear here...")
    pia.descriptionText.Disable() // Read-only
    
    pia.tagsLabel = widget.NewLabel("Tags: none")
    
    // Create system prompt text area
    pia.systemPromptText = widget.NewMultiLineEntry()
    pia.systemPromptText.SetPlaceHolder("System prompt will appear here...")
    pia.systemPromptText.Disable() // Read-only
    
    // Create user prompt text area
    pia.userPromptText = widget.NewMultiLineEntry()
    pia.userPromptText.SetPlaceHolder("User prompt will appear here...")
    pia.userPromptText.Disable() // Read-only
    
    // Create model info label
    pia.modelInfoLabel = widget.NewLabel("Model: none  Vendor: none")
    
    // Create prompt tabs
    promptTabs := container.NewAppTabs(
        container.NewTabItem("System Prompt", pia.systemPromptText),
        container.NewTabItem("User Prompt", pia.userPromptText),
    )
    
    // Create info section
    infoSection := container.NewVBox(
        pia.nameLabel,
        widget.NewSeparator(),
        widget.NewLabel("Description:"),
        pia.descriptionText,
        pia.tagsLabel,
        widget.NewSeparator(),
        pia.modelInfoLabel,
    )
    
    // Assemble the pattern info area
    pia.container = container.NewBorder(
        infoSection,          // Top
        nil,                  // Bottom
        nil, nil,             // Left, Right
        promptTabs,           // Center
    )
    
    return pia
}

// Container returns the root Fyne container for the PatternInfoArea.
func (pia *PatternInfoArea) Container() fyne.CanvasObject {
    return pia.container
}

// UpdateInfo updates the pattern info display.
func (pia *PatternInfoArea) UpdateInfo(patternName, modelName, vendorName string) {
    if patternName == "" {
        pia.nameLabel.SetText("No pattern selected")
        pia.descriptionText.SetText("")
        pia.tagsLabel.SetText("Tags: none")
        pia.systemPromptText.SetText("")
        pia.userPromptText.SetText("")
        return
    }
    
    // Update pattern name
    pia.nameLabel.SetText(patternName)
    
    // Find pattern by name
    var pattern Pattern
    found := false
    for _, p := range pia.app.state.LoadedPatterns {
        if p.Name == patternName {
            pattern = p
            found = true
            break
        }
    }
    
    if !found {
        pia.descriptionText.SetText("Pattern details not found")
        pia.tagsLabel.SetText("Tags: none")
        pia.systemPromptText.SetText("")
        pia.userPromptText.SetText("")
        return
    }
    
    // Update description
    pia.descriptionText.SetText(pattern.Description)
    
    // Update tags
    if len(pattern.Tags) == 0 {
        pia.tagsLabel.SetText("Tags: none")
    } else {
        pia.tagsLabel.SetText("Tags: " + strings.Join(pattern.Tags, ", "))
    }
    
    // Update prompts
    pia.systemPromptText.SetText(pattern.SystemPrompt)
    pia.userPromptText.SetText(pattern.UserPrompt)
    
    // Update model info
    pia.modelInfoLabel.SetText(fmt.Sprintf("Model: %s  Vendor: %s", modelName, vendorName))
}

// StatusBar displays status messages at the bottom of the window.
type StatusBar struct {
    label     *widget.Label
    container *fyne.Container
}

// NewStatusBar creates a new status bar.
func NewStatusBar() *StatusBar {
    sb := &StatusBar{}
    
    // Create status label
    sb.label = widget.NewLabel("Ready")
    
    // Create container
    sb.container = container.NewHBox(
        sb.label,
    )
    
    return sb
}

// Container returns the root Fyne container for the StatusBar.
func (sb *StatusBar) Container() fyne.CanvasObject {
    return sb.container
}

// ShowMessage displays a message in the status bar.
func (sb *StatusBar) ShowMessage(message string) {
    sb.label.SetText(message)
}
