package foundation

import (
	"fmt"
	"log"
	"sort"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// ModelProviderPanel manages the UI for provider and model selection
type ModelProviderPanel struct {
	app *FabricApp // Reference to the main app

	// UI components
	container     *fyne.Container
	section       *CollapsibleSection
	vendorSelect  *widget.Select
	modelSelect   *widget.Select
	statusLabel   *widget.Label
	infoContainer *fyne.Container

	// State
	isLoading      bool
	lastVendorLoad time.Time
	loadingModels  bool
}

// NewModelProviderPanel creates a new panel for model and provider selection
func NewModelProviderPanel(app *FabricApp) *ModelProviderPanel {
	panel := &ModelProviderPanel{
		app:           app,
		isLoading:     false,
		lastVendorLoad: time.Time{},
	}

	panel.initializeComponents()
	panel.createLayout()

	// Initialize with data after UI is ready
	go panel.initializeData()

	return panel
}

// initializeComponents creates and configures all UI components
func (mp *ModelProviderPanel) initializeComponents() {
	// Status label for showing loading/error states
	mp.statusLabel = widget.NewLabel("")
	mp.statusLabel.Hide()

	// Create vendor select with loading placeholder
	mp.vendorSelect = widget.NewSelect([]string{"Loading providers..."}, mp.onVendorChanged)
	mp.vendorSelect.PlaceHolder = "Select AI Provider"
	
	// Create model select with placeholder
	mp.modelSelect = widget.NewSelect([]string{"Select a provider first"}, mp.onModelChanged)
	mp.modelSelect.PlaceHolder = "Select Model"
	mp.modelSelect.Disable() // Disabled until a provider is selected

	// Info container for additional provider/model info
	mp.infoContainer = container.NewVBox()
}

// createLayout assembles the UI components into a layout
func (mp *ModelProviderPanel) createLayout() {
	content := container.NewVBox(
		widget.NewLabel("Provider:"),
		mp.vendorSelect,
		widget.NewSeparator(),
		widget.NewLabel("Model:"),
		mp.modelSelect,
		mp.statusLabel,
		mp.infoContainer,
	)

	// Create collapsible section
	mp.section = NewCollapsibleSection("AI Model", content)
	
	// Main container
	mp.container = container.NewVBox(mp.section)
}

// Container returns the root container for this panel
func (mp *ModelProviderPanel) Container() fyne.CanvasObject {
	return mp.container
}

// initializeData loads initial data for the panel
func (mp *ModelProviderPanel) initializeData() {
	// Set loading state
	mp.setLoading(true, "Initializing providers...")
	
	// Load vendors first
	err := mp.loadVendors()
	if err != nil {
		mp.setError(fmt.Sprintf("Failed to load providers: %v", err))
		return
	}
	
	// If we have a current vendor in state, select it and load its models
	if mp.app.state.CurrentVendorID != "" {
		// Check if the vendor exists in our options
		vendorExists := false
		for _, v := range mp.vendorSelect.Options {
			if v == mp.app.state.CurrentVendorID {
				vendorExists = true
				break
			}
		}
		
		if vendorExists {
			// This will trigger onVendorChanged which loads models
			mp.vendorSelect.SetSelected(mp.app.state.CurrentVendorID)
		} else if len(mp.vendorSelect.Options) > 0 {
			// Select first available vendor if current one not found
			mp.vendorSelect.SetSelected(mp.vendorSelect.Options[0])
		}
	} else if len(mp.vendorSelect.Options) > 0 {
		// No vendor in state, select first available
		mp.vendorSelect.SetSelected(mp.vendorSelect.Options[0])
	}
	
	mp.setLoading(false, "")
}

// loadVendors loads the list of available vendors
func (mp *ModelProviderPanel) loadVendors() error {
	// Skip if we loaded recently (debounce)
	if !mp.lastVendorLoad.IsZero() && time.Since(mp.lastVendorLoad) < 5*time.Second {
		return nil
	}
	
	// Set loading state
	mp.setLoading(true, "Loading providers...")
	
	// Load vendors from Fabric config
	vendors, err := mp.app.fabricConfig.LoadVendors()
	if err != nil {
		return fmt.Errorf("failed to load providers: %w", err)
	}
	
	// Update last load time
	mp.lastVendorLoad = time.Now()
	
	// Sort vendors alphabetically
	sort.Strings(vendors)
	
	// Store in app state
	mp.app.state.LoadedVendors = vendors
	
	// Update UI on main thread
	fyne.CurrentApp().Driver().RunOnMain(func() {
		// Update vendor select options
		if len(vendors) == 0 {
			mp.vendorSelect.Options = []string{"No providers available"}
			mp.vendorSelect.Disable()
		} else {
			mp.vendorSelect.Options = vendors
			mp.vendorSelect.Enable()
		}
		
		mp.vendorSelect.Refresh()
		mp.setLoading(false, "")
	})
	
	return nil
}

// loadModelsForVendor loads models for the specified vendor
func (mp *ModelProviderPanel) loadModelsForVendor(vendorName string) {
	// Set loading state
	mp.loadingModels = true
	mp.setLoading(true, fmt.Sprintf("Loading models for %s...", vendorName))
	
	// Disable model select during loading
	mp.modelSelect.Disable()
	mp.modelSelect.Options = []string{"Loading models..."}
	mp.modelSelect.SetSelected("Loading models...")
	mp.modelSelect.Refresh()
	
	// Load models asynchronously
	go func() {
		// Check if models are already cached in state
		if models, ok := mp.app.state.LoadedModels[vendorName]; ok && len(models) > 0 {
			log.Printf("Using cached models for %s (%d models)", vendorName, len(models))
			mp.updateModelSelectWithModels(models)
			return
		}
		
		// Load models from Fabric config
		models, err := mp.app.fabricConfig.LoadModelsForVendor(vendorName)
		if err != nil {
			log.Printf("Error loading models for %s: %v", vendorName, err)
			fyne.CurrentApp().Driver().RunOnMain(func() {
				mp.modelSelect.Options = []string{"Error loading models"}
				mp.modelSelect.SetSelected("Error loading models")
				mp.modelSelect.Disable()
				mp.modelSelect.Refresh()
				mp.setError(fmt.Sprintf("Failed to load models: %v", err))
			})
			return
		}
		
		// Cache models in state
		mp.app.state.LoadedModels[vendorName] = models
		mp.app.state.VendorModelCounts[vendorName] = len(models)
		
		// Update UI with models
		mp.updateModelSelectWithModels(models)
	}()
}

// updateModelSelectWithModels updates the model select with the provided models
func (mp *ModelProviderPanel) updateModelSelectWithModels(models []string) {
	// Sort models alphabetically
	sortedModels := make([]string, len(models))
	copy(sortedModels, models)
	sort.Strings(sortedModels)
	
	// Update UI on main thread
	fyne.CurrentApp().Driver().RunOnMain(func() {
		mp.loadingModels = false
		
		if len(sortedModels) == 0 {
			mp.modelSelect.Options = []string{"No models available"}
			mp.modelSelect.SetSelected("No models available")
			mp.modelSelect.Disable()
		} else {
			mp.modelSelect.Options = sortedModels
			mp.modelSelect.Enable()
			
			// Select current model from state if available
			if mp.app.state.CurrentModelID != "" {
				modelExists := false
				for _, m := range sortedModels {
					if m == mp.app.state.CurrentModelID {
						modelExists = true
						break
					}
				}
				
				if modelExists {
					mp.modelSelect.SetSelected(mp.app.state.CurrentModelID)
				} else {
					// Select first model if current one not found
					mp.modelSelect.SetSelected(sortedModels[0])
					mp.app.state.CurrentModelID = sortedModels[0]
					
					// Save to config
					if mp.app.fabricConfig != nil {
						mp.app.fabricConfig.SetConfig("DEFAULT_MODEL", sortedModels[0])
						mp.app.fabricConfig.SaveEnvConfig()
					}
				}
			} else if len(sortedModels) > 0 {
				// No model in state, select first available
				mp.modelSelect.SetSelected(sortedModels[0])
				mp.app.state.CurrentModelID = sortedModels[0]
				
				// Save to config
				if mp.app.fabricConfig != nil {
					mp.app.fabricConfig.SetConfig("DEFAULT_MODEL", sortedModels[0])
					mp.app.fabricConfig.SaveEnvConfig()
				}
			}
		}
		
		mp.modelSelect.Refresh()
		mp.setLoading(false, "")
		
		// Show model count in status
		if len(sortedModels) > 0 {
			mp.showStatus(fmt.Sprintf("%d models available", len(sortedModels)))
		}
	})
}

// onVendorChanged handles vendor selection changes
func (mp *ModelProviderPanel) onVendorChanged(selected string) {
	// Skip if nothing selected or no change
	if selected == "" || selected == "Loading providers..." || selected == "No providers available" {
		return
	}
	
	// Update app state
	mp.app.state.CurrentVendorID = selected
	
	// Save to config
	if mp.app.fabricConfig != nil {
		mp.app.fabricConfig.SetConfig("DEFAULT_VENDOR", selected)
		mp.app.fabricConfig.SaveEnvConfig()
	}
	
	// Show feedback
	mp.app.ShowMessage(fmt.Sprintf("Selected provider: %s", selected))
	
	// Load models for this vendor
	mp.loadModelsForVendor(selected)
	
	// Expand section if collapsed
	if !mp.section.IsExpanded {
		mp.section.SetExpanded(true)
	}
}

// onModelChanged handles model selection changes
func (mp *ModelProviderPanel) onModelChanged(selected string) {
	// Skip if nothing selected or loading/error state
	if selected == "" || selected == "Loading models..." || 
	   selected == "No models available" || selected == "Error loading models" ||
	   selected == "Select a provider first" {
		return
	}
	
	// Update app state
	mp.app.state.CurrentModelID = selected
	mp.app.state.CurrentModelName = selected // Use ID as name for now
	
	// Save to config
	if mp.app.fabricConfig != nil {
		mp.app.fabricConfig.SetConfig("DEFAULT_MODEL", selected)
		mp.app.fabricConfig.SaveEnvConfig()
	}
	
	// Show feedback
	mp.app.ShowMessage(fmt.Sprintf("Selected model: %s", selected))
	
	// Update any dependent UI components
	if mp.app.state.CurrentPatternID != "" {
		patternName := mp.app.getPatternNameByID(mp.app.state.CurrentPatternID)
		mp.app.mainLayout.MainContent.patternInfoArea.UpdateInfo(
			patternName, 
			mp.app.state.CurrentModelName, 
			mp.app.state.CurrentVendorID,
		)
	}
}

// setLoading updates the loading state of the panel
func (mp *ModelProviderPanel) setLoading(loading bool, message string) {
	mp.isLoading = loading
	
	fyne.CurrentApp().Driver().RunOnMain(func() {
		if loading {
			mp.showStatus(message)
		} else if !mp.loadingModels {
			mp.statusLabel.Hide()
		}
	})
}

// setError displays an error message
func (mp *ModelProviderPanel) setError(message string) {
	fyne.CurrentApp().Driver().RunOnMain(func() {
		mp.statusLabel.SetText(message)
		mp.statusLabel.Show()
		mp.isLoading = false
	})
}

// showStatus displays a status message
func (mp *ModelProviderPanel) showStatus(message string) {
	fyne.CurrentApp().Driver().RunOnMain(func() {
		mp.statusLabel.SetText(message)
		mp.statusLabel.Show()
	})
}

// Refresh updates the panel with the latest data
func (mp *ModelProviderPanel) Refresh() {
	// Reload vendors if needed
	if time.Since(mp.lastVendorLoad) > 30*time.Second {
		go mp.loadVendors()
	}
	
	// Refresh UI components
	mp.vendorSelect.Refresh()
	mp.modelSelect.Refresh()
}

// SetExpanded sets the expanded state of the panel
func (mp *ModelProviderPanel) SetExpanded(expanded bool) {
	mp.section.SetExpanded(expanded)
}
