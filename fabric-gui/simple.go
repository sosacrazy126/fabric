package main

import (
	"fmt"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func main() {
	// Print environment info
	fmt.Println("DISPLAY:", os.Getenv("DISPLAY"))
	fmt.Println("XDG_SESSION_TYPE:", os.Getenv("XDG_SESSION_TYPE"))
	
	// Create Fyne app
	a := app.New()
	w := a.NewWindow("Hello")
	
	// Create content
	label := widget.NewLabel("Hello Fyne!")
	button := widget.NewButton("Click Me", func() {
		label.SetText("Button clicked!")
	})
	
	// Set content
	w.SetContent(container.NewVBox(
		label,
		button,
	))
	
	// Show and run
	w.Resize(fyne.NewSize(200, 100))
	fmt.Println("Showing window...")
	w.ShowAndRun()
	fmt.Println("Application closed")
}