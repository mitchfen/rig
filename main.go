package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"rig/agent"
	"rig/client"
	"rig/tools"
)

func main() {
	endpoint := flag.String("url", "http://127.0.0.1:1234/v1", "LM Studio OpenAI endpoint")
	modelFlag := flag.String("model", "", "Model ID to use (auto-detected if blank)")
	autoYes := flag.Bool("y", false, "Auto-approve all tool actions without prompting")
	flag.BoolVar(autoYes, "yes", false, "Auto-approve all tool actions without prompting")
	maxSteps := flag.Int("steps", 15, "Maximum iterative tool steps per turn")
	flag.Parse()

	ctx := context.Background()
	llmClient := client.New(*endpoint)

	// Fetch available models
	fmt.Printf("%s[rig]%s Connecting to LM Studio at %s...\n", agent.ColorBold+agent.ColorCyan, agent.ColorReset, *endpoint)
	availableModels, err := llmClient.ListModels(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%sError connecting to LM Studio:%s %v\n", agent.ColorRed, agent.ColorReset, err)
		fmt.Fprintf(os.Stderr, "Make sure LM Studio is open, local server is started, and listening at %s\n", *endpoint)
		os.Exit(1)
	}

	if len(availableModels) == 0 {
		fmt.Fprintf(os.Stderr, "%sNo models currently loaded in LM Studio! Please load a model in LM Studio first.%s\n", agent.ColorRed, agent.ColorReset)
		os.Exit(1)
	}

	selectedModel := *modelFlag
	if selectedModel == "" {
		// Prefer muse-glimmer or qwen if present, otherwise default to first
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

	fmt.Printf("%s[rig]%s Using model: %s%s%s (Available: %s)\n",
		agent.ColorBold+agent.ColorCyan,
		agent.ColorReset,
		agent.ColorGreen+agent.ColorBold,
		selectedModel,
		agent.ColorReset,
		strings.Join(availableModels, ", "),
	)

	registry := tools.NewRegistry()
	bot := agent.New(llmClient, selectedModel, registry, agent.Config{
		AutoApprove: *autoYes,
		MaxSteps:    *maxSteps,
	})

	// Interactive REPL Mode
	fmt.Printf("\n%s=== rig Interactive Coding Harness ===%s\n", agent.ColorBold+agent.ColorGreen, agent.ColorReset)
	fmt.Printf("Type your request below. Type %s/exit%s or %squit%s to stop.\n\n", agent.ColorYellow, agent.ColorReset, agent.ColorYellow, agent.ColorReset)

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

		if input == "/exit" || input == "quit" || input == "exit" {
			fmt.Println("Goodbye!")
			break
		}

		if err := bot.RunTurn(ctx, input); err != nil {
			fmt.Printf("%sTurn error:%s %v\n", agent.ColorRed, agent.ColorReset, err)
		}
		fmt.Println()
	}
}
