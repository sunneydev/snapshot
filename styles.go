package main

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle    = lipgloss.NewStyle().Bold(true)
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	accentStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
	successStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)
