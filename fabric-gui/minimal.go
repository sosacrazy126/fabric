package main

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func main() {
	log.Println("Starting minimal Fabric GUI")
	
	// Create the Fyne application
	a := app.New()
	w := a.NewWindow("Fabric GUI - Minimal")
	
	// Create a simple UI
	label := widget.NewLabel("Welcome to Fabric GUI")
	button := widget.NewButton("Click Me", func() {
		label.SetText("Button clicked!")
	})
	
	// Set the window content
	w.SetContent(container.NewVBox(
		label,
		button,
	))
	
	// Set window size and show
	w.Resize(fyne.NewSize(400, 300))
	log.Println("Running application...")
	w.ShowAndRun()
}