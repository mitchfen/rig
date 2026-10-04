package agent

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"rig/client"
	"rig/tools"
)

// ANSI color escape codes for clean terminal output
const (
	ColorReset  = "\033[0m"
	ColorBold   = "\033[1m"
	ColorDim    = "\033[2m"
	ColorCyan   = "\033[36m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorRed    = "\033[31m"
	ColorMagenta= "\033[35m"
)

type Config struct {
	AutoApprove bool // If true, don't ask before running destructive tools
	MaxSteps    int  // Max iterative tool turns to prevent infinite loops
}

type Agent struct {
	Client   *client.Client
	Model    string
	Registry *tools.Registry
	Config   Config
	Messages []client.Message
	scanner  *bufio.Scanner
}

func New(c *client.Client, model string, reg *tools.Registry, cfg Config) *Agent {
	if cfg.MaxSteps <= 0 {
		cfg.MaxSteps = 15
	}

	systemPrompt := `You are rig, an expert AI programming assistant running locally.
You have access to tools to inspect and modify code, read and write files, and run terminal commands.

Guidelines:
1. Always explore the workspace first using list_dir or read_file if you need context.
2. Formulate clear, concise explanations and use tools directly to solve the user's tasks.
3. When creating or updating files, ensure content is complete and syntactically correct.
4. If a command or tool fails, analyze the error output and attempt a fix.`

	return &Agent{
		Client:   c,
		Model:    model,
		Registry: reg,
		Config:   cfg,
		Messages: []client.Message{
			{Role: "system", Content: systemPrompt},
		},
		scanner: bufio.NewScanner(os.Stdin),
	}
}

// RunTurn processes a user message through the tool-calling loop until the model finishes.
func (a *Agent) RunTurn(ctx context.Context, userInput string) error {
	a.Messages = append(a.Messages, client.Message{
		Role:    "user",
		Content: userInput,
	})

	for step := 0; step < a.Config.MaxSteps; step++ {
		req := client.ChatRequest{
			Model:       a.Model,
			Messages:    a.Messages,
			Tools:       a.Registry.Definitions(),
			Temperature: 0.2,
		}

		fmt.Printf("%sThinking...%s\r", ColorDim, ColorReset)
		respMsg, err := a.Client.Chat(ctx, req)
		// Clear the thinking indicator
		fmt.Print("\r\033[K")

		if err != nil {
			return fmt.Errorf("model error: %w", err)
		}

		a.Messages = append(a.Messages, *respMsg)

		// Print any text response from the model
		if strings.TrimSpace(respMsg.Content) != "" {
			fmt.Printf("\n%s%s%s\n", ColorCyan, respMsg.Content, ColorReset)
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
				fmt.Printf("%s[Tool Error: %s]%s\n", ColorRed, errMsg, ColorReset)
				a.Messages = append(a.Messages, client.Message{
					Role:       "tool",
					ToolCallID: tc.ID,
					Name:       toolName,
					Content:    errMsg,
				})
				continue
			}

			// Show tool action banner
			fmt.Printf("\n%s⚙ Tool Call:%s %s%s%s\n", ColorYellow+ColorBold, ColorReset, ColorMagenta, toolName, ColorReset)
			fmt.Printf("%sArgs:%s %s\n", ColorDim, ColorReset, toolArgs)

			// Permission confirmation if destructive and not auto-approved
			if tool.IsDestructive() && !a.Config.AutoApprove {
				if !a.confirmExecution(toolName, toolArgs) {
					userDeniedMsg := "Tool execution cancelled by user."
					fmt.Printf("%s[Cancelled]%s\n", ColorRed, ColorReset)
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

			// Print preview of output
			preview := output
			if len(preview) > 300 {
				preview = preview[:300] + "... (truncated)"
			}
			fmt.Printf("%sResult:%s %s\n", ColorGreen, ColorReset, preview)

			a.Messages = append(a.Messages, client.Message{
				Role:       "tool",
				ToolCallID: tc.ID,
				Name:       toolName,
				Content:    output,
			})
		}
	}

	fmt.Printf("\n%s[Warning: reached maximum tool turn limit (%d)]%s\n", ColorYellow, a.Config.MaxSteps, ColorReset)
	return nil
}

func (a *Agent) confirmExecution(name, args string) bool {
	fmt.Printf("%sAllow execution of %s%s? [Y/n]: ", ColorYellow+ColorBold, name, ColorReset)
	if !a.scanner.Scan() {
		return false
	}
	input := strings.TrimSpace(strings.ToLower(a.scanner.Text()))
	return input == "" || input == "y" || input == "yes"
}
