package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

var (
	// Brand / Color Tokens
	PrimaryColor   = lipgloss.Color("#7D56F4") // Purple / Violet
	SecondaryColor = lipgloss.Color("#04B575") // Vibrant Emerald Green
	AccentColor    = lipgloss.Color("#FF79C6") // Pink / Magenta
	WarningColor   = lipgloss.Color("#FFB86C") // Amber / Orange
	ErrorColor     = lipgloss.Color("#FF5555") // Red
	MutedColor     = lipgloss.Color("#6272A4") // Slate / Muted gray
	BorderColor    = lipgloss.Color("#44475A") // Subtle border gray

	// Styles
	Badge = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(PrimaryColor).
		Padding(0, 1)

	ModelBadge = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(SecondaryColor).
		Padding(0, 1)

	BannerBox = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(PrimaryColor).
		Padding(0, 1).
		MarginBottom(1)

	ToolCard = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(WarningColor).
		Padding(0, 1).
		MarginTop(1).
		MarginBottom(1)

	ToolTitle = lipgloss.NewStyle().
		Bold(true).
		Foreground(WarningColor)

	ToolArgs = lipgloss.NewStyle().
		Foreground(MutedColor)

	ResultCard = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(SecondaryColor).
		Padding(0, 1).
		MarginBottom(1)

	ResultTitle = lipgloss.NewStyle().
		Bold(true).
		Foreground(SecondaryColor)

	PromptPrefix = lipgloss.NewStyle().
		Bold(true).
		Foreground(PrimaryColor).
		Render("rig") + lipgloss.NewStyle().Foreground(SecondaryColor).Render(" ❯ ")

	ThinkingStyle = lipgloss.NewStyle().
		Italic(true).
		Foreground(MutedColor)

	PromptWarning = lipgloss.NewStyle().
		Bold(true).
		Foreground(WarningColor)

	ErrorStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(ErrorColor)
)

// RenderMarkdown renders markdown text using Glamour dark theme.
func RenderMarkdown(content string) string {
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(100),
	)
	if err != nil {
		return content
	}

	out, err := renderer.Render(content)
	if err != nil {
		return content
	}
	return strings.TrimSpace(out)
}

// RenderToolCall renders a styled card for tool invocations.
func RenderToolCall(name, args string) string {
	content := fmt.Sprintf("%s %s\n%s",
		ToolTitle.Render("⚙ Tool:"),
		lipgloss.NewStyle().Bold(true).Foreground(AccentColor).Render(name),
		ToolArgs.Render("args: "+args),
	)
	return ToolCard.Render(content)
}

// RenderToolResult renders a preview card for tool results.
func RenderToolResult(output string) string {
	preview := output
	if len(preview) > 500 {
		preview = preview[:500] + "\n... (truncated)"
	}
	content := fmt.Sprintf("%s\n%s",
		ResultTitle.Render("✓ Output:"),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#F8F8F2")).Render(preview),
	)
	return ResultCard.Render(content)
}
