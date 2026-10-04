package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"rig/src/agent"
	"rig/src/client"
	"rig/src/config"
	"rig/src/tools"
	"rig/src/ui"
)

func main() {
	configPath := flag.String("config", "config.json", "Path to config file")
	endpointFlag := flag.String("url", "", "LM Studio OpenAI endpoint (overrides config)")
	modelFlag := flag.String("model", "", "Model ID to use (overrides config)")
	maxSteps := flag.Int("steps", 0, "Maximum iterative tool steps per turn (overrides config)")
	maxContext := flag.Int("context", 0, "Maximum context tokens (overrides config)")
	flag.Parse()

	// Load configuration file
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Println(ui.PromptWarning.Render(fmt.Sprintf("Warning loading config file: %v (using defaults)", err)))
		cfg = config.Default()
	}

	// Apply CLI overrides if specified
	if *endpointFlag != "" {
		cfg.Endpoint = *endpointFlag
	}
	if *modelFlag != "" {
		cfg.Model = *modelFlag
	}
	if *maxSteps > 0 {
		cfg.MaxSteps = *maxSteps
	}
	if *maxContext > 0 {
		cfg.MaxContextTokens = *maxContext
	}

	ctx := context.Background()
	llmClient := client.New(cfg.Endpoint)

	// Fetch available models from LM Studio
	fmt.Print(ui.ThinkingStyle.Render(fmt.Sprintf("Connecting to LM Studio at %s...", cfg.Endpoint)) + "\r")
	availableModels, err := llmClient.ListModels(ctx)
	fmt.Print("\r\033[K")

	if err != nil {
		fmt.Println(ui.ErrorStyle.Render(fmt.Sprintf("Error connecting to LM Studio: %v", err)))
		fmt.Printf("Make sure LM Studio is running, local server is started, and listening at %s\n", cfg.Endpoint)
		os.Exit(1)
	}

	if len(availableModels) == 0 {
		fmt.Println(ui.ErrorStyle.Render("No models currently loaded in LM Studio! Please load a model in LM Studio first."))
		os.Exit(1)
	}

	// Select model: prefer configured model if active, else auto-detect
	selectedModel := cfg.Model
	modelFound := false
	for _, m := range availableModels {
		if m == selectedModel {
			modelFound = true
			break
		}
	}

	if !modelFound {
		for _, m := range availableModels {
			if strings.Contains(strings.ToLower(m), "muse") || strings.Contains(strings.ToLower(m), "qwen") {
				selectedModel = m
				break
			}
		}
		if selectedModel == "" {
			selectedModel = availableModels[0]
		}
	}

	// Styled Startup Banner
	bannerText := fmt.Sprintf("%s  %s\n\n%s %s\n%s %s\n%s %s",
		ui.Badge.Render("RIG"),
		lipgloss.NewStyle().Foreground(ui.MutedColor).Render("Local Homelab Coding Harness"),
		lipgloss.NewStyle().Bold(true).Render("Model:"), ui.ModelBadge.Render(selectedModel),
		lipgloss.NewStyle().Bold(true).Render("Context:"), lipgloss.NewStyle().Foreground(ui.PrimaryColor).Render(fmt.Sprintf("%d tokens", cfg.MaxContextTokens)),
		lipgloss.NewStyle().Bold(true).Render("Endpoint:"), lipgloss.NewStyle().Foreground(ui.MutedColor).Render(cfg.Endpoint),
	)
	if len(cfg.ContextFiles) > 0 {
		bannerText += fmt.Sprintf("\n%s %s",
			lipgloss.NewStyle().Bold(true).Render("Context:"),
			lipgloss.NewStyle().Foreground(ui.MutedColor).Render(strings.Join(cfg.ContextFiles, ", ")),
		)
	}
	fmt.Println(ui.BannerBox.Render(bannerText))
	fmt.Printf("%s /context (usage), /reset (clear history), /exit\n\n", ui.ThinkingStyle.Render("Commands:"))

	registry := tools.NewRegistry(cfg.MaxToolOutputChars)
	bot := agent.New(llmClient, selectedModel, registry, cfg)

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print(ui.PromptPrefix)
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		switch input {
		case "/exit", "quit", "exit":
			fmt.Println(lipgloss.NewStyle().Foreground(ui.MutedColor).Render("Goodbye!"))
			return
		case "/reset", "/clear":
			bot.Reset()
			fmt.Println(lipgloss.NewStyle().Foreground(ui.SecondaryColor).Render("✓ Conversation reset. Context cleared back to system prompt.\n"))
			continue
		case "/context":
			usedTokens := bot.EstimateTokens()
			pct := float64(usedTokens) / float64(cfg.MaxContextTokens) * 100
			contextCard := fmt.Sprintf("Context Usage: ~%d / %d tokens (%.1f%%) across %d messages",
				usedTokens, cfg.MaxContextTokens, pct, len(bot.Messages))
			fmt.Println(lipgloss.NewStyle().Foreground(ui.PrimaryColor).Bold(true).Render(contextCard) + "\n")
			continue
		}

		if err := bot.RunTurn(ctx, input); err != nil {
			fmt.Println(ui.ErrorStyle.Render(fmt.Sprintf("Turn error: %v", err)))
		}
		fmt.Println()
	}
}
