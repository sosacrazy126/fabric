package main

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func main() {
	// Create a new Fyne application
	a := app.New()
	
	// Create a window
	w := a.NewWindow("Fyne Test")
	
	// Set window size
	w.Resize(fyne.NewSize(400, 300))
	
	// Create a label
	hello := widget.NewLabel("Fyne Test Application")
	
	// Create a button
	btn := widget.NewButton("Click Me", func() {
		fmt.Println("Button clicked!")
		hello.SetText("Button was clicked!")
	})
	
	// Create a layout with the label and button
	content := container.NewVBox(
		hello,
		btn,
	)
	
	// Set the window content
	w.SetContent(content)
	
	// Show and run the application
	w.ShowAndRun()
}