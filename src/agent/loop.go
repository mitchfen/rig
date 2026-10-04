package agent

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"path/filepath"

	"rig/src/client"
	"rig/src/config"
	"rig/src/tools"
	"rig/src/ui"
)

type Agent struct {
	Client       *client.Client
	Model        string
	Registry     *tools.Registry
	Config       config.Config
	SystemPrompt string
	Messages     []client.Message
	scanner      *bufio.Scanner
}

func New(c *client.Client, model string, reg *tools.Registry, cfg config.Config) *Agent {
	if cfg.MaxSteps <= 0 {
		cfg.MaxSteps = 10
	}
	if cfg.MaxContextTokens <= 0 {
		cfg.MaxContextTokens = 16384
	}

	basePrompt := `You are rig, an expert AI programming assistant running locally.
You have access to tools to inspect and modify code, read and write files, and run terminal commands.

Key Directives:
1. Be concise, direct, and pragmatic.
2. Execute only the minimal required commands to answer the user's question.
3. For status queries (e.g. Kubernetes, Docker, system), provide summary commands first; do not exhaustively troubleshoot or inspect individual components unless explicitly asked.`

	// Check for custom instructions file (e.g. instructions.md or config/instructions.md)
	if cfg.InstructionsFile != "" {
		content, err := os.ReadFile(cfg.InstructionsFile)
		if err != nil && os.IsNotExist(err) {
			content, err = os.ReadFile("config/" + cfg.InstructionsFile)
		}
		if err == nil && len(strings.TrimSpace(string(content))) > 0 {
			basePrompt += "\n\nAdditional Instructions:\n" + string(content)
		}
	}

	// Load additional context files specified in config (e.g. ~/Projects/homelab/README.md)
	for _, rawPath := range cfg.ContextFiles {
		resolvedPath := resolvePath(rawPath)
		content, err := os.ReadFile(resolvedPath)
		if err != nil && os.IsNotExist(err) {
			// Also check config/ subdirectory
			content, err = os.ReadFile(filepath.Join("config", resolvedPath))
		}
		if err == nil && len(strings.TrimSpace(string(content))) > 0 {
			basePrompt += fmt.Sprintf("\n\n=== PRE-LOADED BACKGROUND CONTEXT: %s ===\n%s\n=== END OF %s ===", rawPath, strings.TrimSpace(string(content)), rawPath)
		}
	}

	a := &Agent{
		Client:       c,
		Model:        model,
		Registry:     reg,
		Config:       cfg,
		SystemPrompt: basePrompt,
		scanner:      bufio.NewScanner(os.Stdin),
	}
	a.Reset()
	return a
}

// Reset clears the conversation back to only the system prompt.
func (a *Agent) Reset() {
	a.Messages = []client.Message{
		{Role: "system", Content: a.SystemPrompt},
	}
}

// EstimateTokens calculates approximate token count (~4 characters per token).
func (a *Agent) EstimateTokens() int {
	totalChars := 0
	for _, m := range a.Messages {
		totalChars += len(m.Content)
		for _, tc := range m.ToolCalls {
			totalChars += len(tc.Function.Name) + len(tc.Function.Arguments)
		}
	}
	return totalChars / 4
}

// PruneContext ensures conversation does not exceed 75% of MaxContextTokens.
func (a *Agent) PruneContext() {
	limit := int(float64(a.Config.MaxContextTokens) * 0.75)
	if a.EstimateTokens() <= limit || len(a.Messages) <= 4 {
		return
	}

	// Prune older tool outputs from the middle of the conversation
	pruned := 0
	for i := 1; i < len(a.Messages)-2; i++ {
		if a.Messages[i].Role == "tool" && len(a.Messages[i].Content) > 200 {
			a.Messages[i].Content = "[Output pruned to preserve context window]"
			pruned++
			if a.EstimateTokens() <= limit {
				break
			}
		}
	}

	// If still too large, drop oldest turns (preserve system prompt at index 0)
	for a.EstimateTokens() > limit && len(a.Messages) > 4 {
		a.Messages = append(a.Messages[:1], a.Messages[2:]...)
		pruned++
	}

	if pruned > 0 {
		fmt.Println(ui.ThinkingStyle.Render(fmt.Sprintf("[Notice: Pruned older context to fit %d token limit]", a.Config.MaxContextTokens)))
	}
}

// RunTurn processes a user message through the tool-calling loop until the model finishes.
func (a *Agent) RunTurn(ctx context.Context, userInput string) error {
	a.Messages = append(a.Messages, client.Message{
		Role:    "user",
		Content: userInput,
	})

	for step := 0; step < a.Config.MaxSteps; step++ {
		a.PruneContext()

		req := client.ChatRequest{
			Model:       a.Model,
			Messages:    a.Messages,
			Tools:       a.Registry.Definitions(),
			Temperature: 0.2,
		}

		fmt.Print(ui.ThinkingStyle.Render("Thinking...") + "\r")
		respMsg, err := a.Client.Chat(ctx, req)
		fmt.Print("\r\033[K") // clear thinking text

		if err != nil {
			return fmt.Errorf("model error: %w", err)
		}

		a.Messages = append(a.Messages, *respMsg)

		// Render model text as formatted markdown using Glamour
		if strings.TrimSpace(respMsg.Content) != "" {
			renderedMarkdown := ui.RenderMarkdown(respMsg.Content)
			fmt.Println(renderedMarkdown)
		}

		// If no tools were called, the turn is finished
		if len(respMsg.ToolCalls) == 0 {
			return nil
		}

		// Execute each tool call requested by the model
		for _, tc := range respMsg.ToolCalls {
			toolName := tc.Function.Name
			toolArgs := tc.Function.Arguments

			tool, exists := a.Registry.Get(toolName)
			if !exists {
				errMsg := fmt.Sprintf("Error: Unknown tool '%s'", toolName)
				fmt.Println(ui.ErrorStyle.Render(errMsg))
				a.Messages = append(a.Messages, client.Message{
					Role:       "tool",
					ToolCallID: tc.ID,
					Name:       toolName,
					Content:    errMsg,
				})
				continue
			}

			// Render tool call card with Lip Gloss
			fmt.Println(ui.RenderToolCall(toolName, toolArgs))

			// Permission confirmation for destructive actions (always required)
			if tool.IsDestructive() {
				if !a.confirmExecution(toolName, toolArgs) {
					userDeniedMsg := "Tool execution cancelled by user."
					fmt.Println(ui.ErrorStyle.Render("[Execution Cancelled by User]"))
					a.Messages = append(a.Messages, client.Message{
						Role:       "tool",
						ToolCallID: tc.ID,
						Name:       toolName,
						Content:    userDeniedMsg,
					})
					continue
				}
			}

			// Execute tool
			output, execErr := tool.Execute(ctx, toolArgs)
			if execErr != nil {
				output = fmt.Sprintf("Tool error: %v", execErr)
			}

			// Enforce max output characters limit
			maxChars := a.Registry.MaxOutputChars()
			if maxChars > 0 && len(output) > maxChars {
				output = fmt.Sprintf("%s\n\n... [Output truncated: %d chars exceeded max limit of %d. Use specific filters/flags if more details needed.]",
					output[:maxChars], len(output), maxChars)
			}

			// Render tool result preview with Lip Gloss
			fmt.Println(ui.RenderToolResult(output))

			a.Messages = append(a.Messages, client.Message{
				Role:       "tool",
				ToolCallID: tc.ID,
				Name:       toolName,
				Content:    output,
			})
		}
	}

	fmt.Println(ui.PromptWarning.Render(fmt.Sprintf("[Warning: reached maximum tool turn limit (%d)]", a.Config.MaxSteps)))
	return nil
}

func (a *Agent) confirmExecution(name, args string) bool {
	prompt := fmt.Sprintf("⚡ Confirm execution of %s? [Y/n]: ", name)
	fmt.Print(ui.PromptWarning.Render(prompt))
	if !a.scanner.Scan() {
		return false
	}
	input := strings.TrimSpace(strings.ToLower(a.scanner.Text()))
	return input == "" || input == "y" || input == "yes"
}

// resolvePath expands ~ to the user's home directory.
func resolvePath(p string) string {
	if strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, p[2:])
		}
	}
	return p
}
