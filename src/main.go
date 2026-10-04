package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"rig/src/agent"
	"rig/src/client"
	"rig/src/config"
	"rig/src/tools"
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
		fmt.Fprintf(os.Stderr, "%sWarning loading config file:%s %v (using defaults)\n", agent.ColorYellow, agent.ColorReset, err)
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
	fmt.Printf("%s[rig]%s Connecting to LM Studio at %s...\n", agent.ColorBold+agent.ColorCyan, agent.ColorReset, cfg.Endpoint)
	availableModels, err := llmClient.ListModels(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%sError connecting to LM Studio:%s %v\n", agent.ColorRed, agent.ColorReset, err)
		fmt.Fprintf(os.Stderr, "Make sure LM Studio is running, local server is started, and listening at %s\n", cfg.Endpoint)
		os.Exit(1)
	}

	if len(availableModels) == 0 {
		fmt.Fprintf(os.Stderr, "%sNo models currently loaded in LM Studio! Please load a model in LM Studio first.%s\n", agent.ColorRed, agent.ColorReset)
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

	fmt.Printf("%s[rig]%s Using model: %s%s%s (Context limit: %d tokens)\n",
		agent.ColorBold+agent.ColorCyan,
		agent.ColorReset,
		agent.ColorGreen+agent.ColorBold,
		selectedModel,
		agent.ColorReset,
		cfg.MaxContextTokens,
	)

	registry := tools.NewRegistry(cfg.MaxToolOutputChars)
	bot := agent.New(llmClient, selectedModel, registry, cfg)

	// Interactive REPL Mode
	fmt.Printf("\n%s=== rig Interactive Coding Harness ===%s\n", agent.ColorBold+agent.ColorGreen, agent.ColorReset)
	fmt.Printf("Commands: %s/reset%s (clear context), %s/context%s (token usage), %s/exit%s\n\n",
		agent.ColorYellow, agent.ColorReset, agent.ColorYellow, agent.ColorReset, agent.ColorYellow, agent.ColorReset)

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Printf("%srig > %s", agent.ColorBold+agent.ColorCyan, agent.ColorReset)
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		// REPL control commands
		switch input {
		case "/exit", "quit", "exit":
			fmt.Println("Goodbye!")
			return
		case "/reset", "/clear":
			bot.Reset()
			fmt.Printf("%s[Conversation reset. Context cleared back to system prompt]%s\n\n", agent.ColorGreen, agent.ColorReset)
			continue
		case "/context":
			usedTokens := bot.EstimateTokens()
			pct := float64(usedTokens) / float64(cfg.MaxContextTokens) * 100
			fmt.Printf("%s[Context Usage: ~%d / %d tokens (%.1f%%) across %d messages]%s\n\n",
				agent.ColorCyan, usedTokens, cfg.MaxContextTokens, pct, len(bot.Messages), agent.ColorReset)
			continue
		}

		if err := bot.RunTurn(ctx, input); err != nil {
			fmt.Printf("%sTurn error:%s %v\n", agent.ColorRed, agent.ColorReset, err)
		}
		fmt.Println()
	}
}
