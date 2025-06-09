package foundation

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// CollapsibleSection is a custom widget that can be expanded or collapsed
type CollapsibleSection struct {
	widget.BaseWidget
	Title        string
	Content      fyne.CanvasObject
	IsExpanded   bool
	TitleStyle   fyne.TextStyle
	ToggleButton *widget.Button
	container    *fyne.Container
	onToggle     func(bool)
}

// NewCollapsibleSection creates a new collapsible section with the given title and content
func NewCollapsibleSection(title string, content fyne.CanvasObject) *CollapsibleSection {
	section := &CollapsibleSection{
		Title:      title,
		Content:    content,
		IsExpanded: false,
		TitleStyle: fyne.TextStyle{Bold: true},
	}
	section.ExtendBaseWidget(section)
	
	// Create toggle button with appropriate icon
	section.ToggleButton = widget.NewButtonWithIcon("", theme.NavigateNextIcon(), func() {
		section.Toggle()
	})
	section.ToggleButton.Importance = widget.LowImportance
	
	// Create container that will be updated when toggled
	section.updateContainer()
	
	return section
}

// SetOnToggle sets a function to be called when the section is toggled
func (s *CollapsibleSection) SetOnToggle(f func(bool)) {
	s.onToggle = f
}

// Toggle toggles the expanded state of the section
func (s *CollapsibleSection) Toggle() {
	s.IsExpanded = !s.IsExpanded
	s.updateContainer()
	s.Refresh()
	if s.onToggle != nil {
		s.onToggle(s.IsExpanded)
	}
}

// SetExpanded sets the expanded state of the section
func (s *CollapsibleSection) SetExpanded(expanded bool) {
	if s.IsExpanded != expanded {
		s.IsExpanded = expanded
		s.updateContainer()
		s.Refresh()
		if s.onToggle != nil {
			s.onToggle(s.IsExpanded)
		}
	}
}

// updateContainer creates or updates the section's layout
func (s *CollapsibleSection) updateContainer() {
	// Update button icon based on expanded state
	if s.IsExpanded {
		// Use MenuDropDown as a substitute for NavigateDown
		s.ToggleButton.SetIcon(theme.MenuDropDownIcon())
	} else {
		s.ToggleButton.SetIcon(theme.NavigateNextIcon())
	}
	
	// Create title row with label and toggle button
	titleLabel := widget.NewLabelWithStyle(s.Title, fyne.TextAlignLeading, s.TitleStyle)
	titleRow := container.NewBorder(nil, nil, nil, s.ToggleButton, titleLabel)
	
	// Build container based on expanded state
	if s.IsExpanded {
		s.container = container.NewVBox(
			titleRow,
			s.Content,
		)
	} else {
		s.container = container.NewVBox(
			titleRow,
		)
	}
}

// CreateRenderer is a private method to Fyne which links this widget to its renderer
func (s *CollapsibleSection) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(s.container)
}

// MinSize returns the minimum size of the collapsible section
func (s *CollapsibleSection) MinSize() fyne.Size {
	return s.container.MinSize()
}

// CardSection creates a card-like container with a title and content
func CardSection(title string, content fyne.CanvasObject) fyne.CanvasObject {
	titleLabel := widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	
	// Create a background rectangle with rounded corners
	background := canvas.NewRectangle(theme.BackgroundColor())
	background.CornerRadius = 8
	background.StrokeWidth = 1
	background.StrokeColor = theme.ShadowColor()
	
	// Create a padding container for the content
	paddedContent := container.NewPadded(content)
	
	// Create a border container with the title at the top
	bordered := container.NewBorder(
		container.NewPadded(titleLabel),
		nil, nil, nil,
		paddedContent,
	)
	
	// Put everything in a container with the background
	return container.NewMax(background, bordered)
}

// ToggleableSection creates a section that can be shown/hidden with a button
func ToggleableSection(title string, initialVisible bool, content fyne.CanvasObject) (fyne.CanvasObject, func(bool)) {
	// Create a container that will hold the content
	contentContainer := container.NewVBox()
	
	// Create button with title that toggles visibility
	var toggleButton *widget.Button
	var icon fyne.Resource
	if initialVisible {
		icon = theme.VisibilityIcon()
	} else {
		icon = theme.VisibilityOffIcon()
	}
	
	toggleButton = widget.NewButtonWithIcon(
		title,
		icon,
		func() {
			// Toggle visibility
			visible := contentContainer.Visible()
			if !visible {
				contentContainer.Show()
			} else {
				contentContainer.Hide()
			}
			if !visible {
				toggleButton.SetIcon(theme.VisibilityIcon())
			} else {
				toggleButton.SetIcon(theme.VisibilityOffIcon())
			}
			contentContainer.Refresh()
		},
	)
	toggleButton.Alignment = widget.ButtonAlignLeading
	toggleButton.Importance = widget.MediumImportance
	
	// Initialize visibility
	if initialVisible {
		contentContainer.Add(content)
	} else {
		contentContainer.Hide()
	}
	
	// The section contains the button and content container
	section := container.NewVBox(
		toggleButton,
		contentContainer,
	)
	
	// Return the section and a function to control visibility programmatically
	setVisible := func(visible bool) {
		if visible != contentContainer.Visible() {
			if visible {
				contentContainer.Show()
				toggleButton.SetIcon(theme.VisibilityIcon())
			} else {
				contentContainer.Hide()
				toggleButton.SetIcon(theme.VisibilityOffIcon())
			}
			contentContainer.Refresh()
		}
		return
	}
	
	return section, setVisible
}

// ModelProviderCard creates a specialized card for the model provider section
func ModelProviderCard(vendorSelect, modelSelect *widget.Select) fyne.CanvasObject {
	// Create main vendor selector with a bold heading
	vendorSection := container.NewVBox(
		widget.NewLabelWithStyle("AI Provider", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		vendorSelect,
	)
	
	// Create model selector (initially empty/hidden)
	modelSection := container.NewVBox(
		widget.NewLabelWithStyle("Model", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		modelSelect,
	)
	
	// Put all content in a padded box with a border
	content := container.NewVBox(
		vendorSection,
		widget.NewSeparator(),
		modelSection,
	)
	
	// Use our CardSection helper to create a nice looking card
	return CardSection("Model Configuration", content)
}