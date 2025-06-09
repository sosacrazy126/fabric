package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbletea"
)

// Define our model which holds the application state
type model struct {
	choices  []string
	cursor   int
	selected string
	patterns []pattern
}

// Pattern represents a Fabric pattern
type pattern struct {
	name        string
	description string
}

// Initial model
func initialModel() model {
	// Sample patterns
	patterns := []pattern{
		{name: "create_summary", description: "Generate summaries for content"},
		{name: "analyze_paper", description: "Analyze academic papers"},
		{name: "extract_insights", description: "Extract key insights from text"},
		{name: "create_visualization", description: "Create visualizations from data"},
		{name: "translate", description: "Translate text between languages"},
	}

	return model{
		choices:  []string{"List Patterns", "Execute Pattern", "Quit"},
		patterns: patterns,
	}
}

// Init initializes the model
func (m model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}

		case "enter", " ":
			m.selected = m.choices[m.cursor]

			// Handle selection
			switch m.selected {
			case "Quit":
				return m, tea.Quit
			}
		}
	}

	return m, nil
}

// View renders the UI
func (m model) View() string {
	s := "Fabric Terminal UI\n\n"

	// Menu items
	for i, choice := range m.choices {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		s += fmt.Sprintf("%s %s\n", cursor, choice)
	}
	s += "\n"

	// Show selected content
	if m.selected != "" {
		s += fmt.Sprintf("You selected: %s\n\n", m.selected)
		
		switch m.selected {
		case "List Patterns":
			s += "Available Patterns:\n"
			for _, p := range m.patterns {
				s += fmt.Sprintf("- %s: %s\n", p.name, p.description)
			}
		case "Execute Pattern":
			s += "Pattern Execution (demo):\n"
			s += "Pattern: create_summary\n"
			s += "Input: Lorem ipsum dolor sit amet, consectetur adipiscing elit.\n"
			s += "Output: This is a summary of the input text.\n"
		}
	}

	// Help text
	s += "\nPress q to quit, ↑/↓ to navigate, enter to select\n"

	return s
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}