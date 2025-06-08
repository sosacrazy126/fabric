// +build linux,!android

package main

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func main() {
	log.Println("Starting Fabric GUI")
	
	myApp := app.New()
	myWindow := myApp.NewWindow("Simple Fabric GUI")

	label := widget.NewLabel("Welcome to Fabric GUI")
	button := widget.NewButton("Click Me", func() {
		label.SetText("Button clicked!")
	})

	content := container.NewVBox(label, button)
	myWindow.SetContent(content)

	myWindow.Resize(fyne.NewSize(300, 200))
	log.Println("Showing window...")
	myWindow.Show()
	
	log.Println("Running main event loop...")
	myApp.Run()
	log.Println("Application closed")
}