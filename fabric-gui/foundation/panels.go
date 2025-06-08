package foundation

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// PatternsPanel manages pattern selection and display
type PatternsPanel struct {
	app         *FabricApp
	patternList *widget.List
	searchEntry *widget.Entry
	systemText  *widget.TextArea
	userText    *widget.TextArea
	content     *fyne.Container
}

// NewPatternsPanel creates a new patterns panel
func NewPatternsPanel(app *FabricApp) *PatternsPanel {
	p := &PatternsPanel{app: app}
	
	// Create search entry
	p.searchEntry = widget.NewEntry()
	p.searchEntry.SetPlaceHolder("Search patterns...")
	
	// Create pattern list
	p.patternList = widget.NewList(
		func() int { return len(app.state.LoadedPatterns) },
		func() fyne.CanvasObject { 
			return container.NewVBox(
				widget.NewLabel("Pattern Name"),
				widget.NewLabel("Description"),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id >= len(app.state.LoadedPatterns) {
				return
			}
			container := obj.(*fyne.Container)
			nameLabel := container.Objects[0].(*widget.Label)
			descLabel := container.Objects[1].(*widget.Label)
			
			pattern := app.state.LoadedPatterns[id]
			nameLabel.SetText(pattern.Name)
			
			// Truncate description if needed
			desc := pattern.Description
			if len(desc) > 60 {
				desc = desc[:57] + "..."
			}
			descLabel.SetText(desc)
		},
	)

	// Create details view
	p.systemText = widget.NewTextArea()
	p.systemText.Disable()
	
	p.userText = widget.NewTextArea()
	p.userText.Disable()
	
	// Tabs for system.md and user.md
	detailsTabs := container.NewAppTabs(
		container.NewTabItem("System Prompt", p.systemText),
		container.NewTabItem("User Prompt", p.userText),
	)

	// Handle selection
	p.patternList.OnSelected = func(id widget.ListItemID) {
		if id >= len(app.state.LoadedPatterns) {
			return
		}
		pattern := app.state.LoadedPatterns[id]
		app.state.CurrentPatternID = pattern.ID
		
		// Update preview
		p.systemText.SetText(pattern.SystemMD)
		p.userText.SetText(pattern.UserMD)
		
		// Update status
		app.status.ShowMessage(fmt.Sprintf("Selected pattern: %s", pattern.Name))
	}

	// Create main layout
	p.content = container.NewBorder(
		p.searchEntry, nil, nil, nil,
		container.NewHSplit(
			p.patternList,
			detailsTabs,
		),
	)

	return p
}

// Container returns the main container for this panel
func (p *PatternsPanel) Container() fyne.CanvasObject {
	return p.content
}

// ExecutePanel manages pattern execution
type ExecutePanel struct {
	app         *FabricApp
	patternInfo *widget.Label
	input       *widget.Entry
	output      *widget.TextArea
	runBtn      *widget.Button
	content     *fyne.Container
}

// NewExecutePanel creates a new execution panel
func NewExecutePanel(app *FabricApp) *ExecutePanel {
	e := &ExecutePanel{app: app}

	// Create pattern info label
	e.patternInfo = widget.NewLabel("No pattern selected")
	
	// Create input/output areas
	e.input = widget.NewMultiLineEntry()
	e.input.SetPlaceHolder("Enter your input here...")
	
	e.output = widget.NewTextArea()
	e.output.Disable()

	// Create run button
	e.runBtn = widget.NewButton("Execute Pattern", func() {
		e.executePattern()
	})

	// Create layout
	e.content = container.NewBorder(
		container.NewVBox(
			e.patternInfo,
			e.runBtn,
		), 
		nil, nil, nil,
		container.NewVSplit(e.input, e.output),
	)

	return e
}

// Container returns the main container for this panel
func (e *ExecutePanel) Container() fyne.CanvasObject {
	return e.content
}

// executePattern executes the selected pattern with the current input
func (e *ExecutePanel) executePattern() {
	if e.app.state.CurrentPatternID == "" {
		e.app.status.ShowMessage("Error: No pattern selected")
		return
	}

	input := e.input.Text
	if input == "" {
		e.app.status.ShowMessage("Error: No input provided")
		return
	}

	// Placeholder for actual execution
	// In a real implementation, this would use Fabric's core components
	go func() {
		e.app.status.ShowMessage("Executing pattern: " + e.app.state.CurrentPatternID)
		
		// Simulated output
		output := fmt.Sprintf("Pattern: %s\nInput: %s\n\nSimulated execution result.", 
			e.app.state.CurrentPatternID, input)
		
		e.output.SetText(output)
		e.app.state.LastOutput = output
		e.app.status.ShowMessage("Pattern execution complete")
	}()
}

// StatusBar shows application status
type StatusBar struct {
	content *widget.Label
}

// NewStatusBar creates a new status bar
func NewStatusBar() *StatusBar {
	return &StatusBar{
		content: widget.NewLabel("Ready"),
	}
}

// ShowError displays an error message
func (s *StatusBar) ShowError(err error) {
	s.content.SetText("Error: " + err.Error())
}

// ShowMessage displays a status message
func (s *StatusBar) ShowMessage(msg string) {
	s.content.SetText(msg)
}