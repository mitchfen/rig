package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"rig/src/client"
)

// Tool defines an executable agent capability.
type Tool interface {
	Name() string
	Definition() client.ToolDefinition
	Execute(ctx context.Context, argsJSON string) (string, error)
	IsDestructive() bool
}

// Registry manages all available tools.
type Registry struct {
	tools          map[string]Tool
	maxOutputChars int
}

func NewRegistry(maxOutputChars int) *Registry {
	if maxOutputChars <= 0 {
		maxOutputChars = 3000
	}
	r := &Registry{
		tools:          make(map[string]Tool),
		maxOutputChars: maxOutputChars,
	}
	r.Register(&ReadFileTool{})
	r.Register(&WriteFileTool{})
	r.Register(&ListDirTool{})
	r.Register(&RunCommandTool{})
	return r
}

func (r *Registry) MaxOutputChars() int {
	return r.maxOutputChars
}

func (r *Registry) Register(t Tool) {
	r.tools[t.Name()] = t
}

func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

func (r *Registry) Definitions() []client.ToolDefinition {
	var defs []client.ToolDefinition
	for _, t := range r.tools {
		defs = append(defs, t.Definition())
	}
	return defs
}

// -------------------------------------------------------------
// read_file Tool
// -------------------------------------------------------------
type ReadFileTool struct{}

func (t *ReadFileTool) Name() string         { return "read_file" }
func (t *ReadFileTool) IsDestructive() bool  { return false }

func (t *ReadFileTool) Definition() client.ToolDefinition {
	return client.ToolDefinition{
		Type: "function",
		Function: client.FunctionSchema{
			Name:        "read_file",
			Description: "Read the contents of a file at the given relative or absolute path.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Path to the file to read",
					},
					"max_lines": map[string]interface{}{
						"type":        "integer",
						"description": "Optional max number of lines to return (defaults to 500)",
					},
				},
				"required": []string{"path"},
			},
		},
	}
}

func (t *ReadFileTool) Execute(ctx context.Context, argsJSON string) (string, error) {
	var args struct {
		Path     string `json:"path"`
		MaxLines int    `json:"max_lines"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	content, err := os.ReadFile(args.Path)
	if err != nil {
		return "", fmt.Errorf("failed to read file '%s': %w", args.Path, err)
	}

	lines := strings.Split(string(content), "\n")
	max := args.MaxLines
	if max <= 0 || max > 1000 {
		max = 500
	}

	if len(lines) > max {
		lines = lines[:max]
		return fmt.Sprintf("%s\n\n... (truncated at %d lines)", strings.Join(lines, "\n"), max), nil
	}

	return string(content), nil
}

// -------------------------------------------------------------
// write_file Tool
// -------------------------------------------------------------
type WriteFileTool struct{}

func (t *WriteFileTool) Name() string         { return "write_file" }
func (t *WriteFileTool) IsDestructive() bool  { return true }

func (t *WriteFileTool) Definition() client.ToolDefinition {
	return client.ToolDefinition{
		Type: "function",
		Function: client.FunctionSchema{
			Name:        "write_file",
			Description: "Create a new file or completely overwrite an existing file with the provided content.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Path to the file to write",
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "Full file contents to write",
					},
				},
				"required": []string{"path", "content"},
			},
		},
	}
}

func (t *WriteFileTool) Execute(ctx context.Context, argsJSON string) (string, error) {
	var args struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	dir := filepath.Dir(args.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory '%s': %w", dir, err)
	}

	if err := os.WriteFile(args.Path, []byte(args.Content), 0644); err != nil {
		return "", fmt.Errorf("failed to write file '%s': %w", args.Path, err)
	}

	return fmt.Sprintf("Successfully wrote %d bytes to %s", len(args.Content), args.Path), nil
}

// -------------------------------------------------------------
// list_dir Tool
// -------------------------------------------------------------
type ListDirTool struct{}

func (t *ListDirTool) Name() string         { return "list_dir" }
func (t *ListDirTool) IsDestructive() bool  { return false }

func (t *ListDirTool) Definition() client.ToolDefinition {
	return client.ToolDefinition{
		Type: "function",
		Function: client.FunctionSchema{
			Name:        "list_dir",
			Description: "List files and subdirectories inside a directory path.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Directory path to list (defaults to current directory '.')",
					},
				},
			},
		},
	}
}

func (t *ListDirTool) Execute(ctx context.Context, argsJSON string) (string, error) {
	var args struct {
		Path string `json:"path"`
	}
	_ = json.Unmarshal([]byte(argsJSON), &args)
	if args.Path == "" {
		args.Path = "."
	}

	entries, err := os.ReadDir(args.Path)
	if err != nil {
		return "", fmt.Errorf("failed to read dir '%s': %w", args.Path, err)
	}

	var out []string
	for _, entry := range entries {
		info, err := entry.Info()
		size := int64(0)
		if err == nil {
			size = info.Size()
		}
		if entry.IsDir() {
			out = append(out, fmt.Sprintf("[DIR]  %s/", entry.Name()))
		} else {
			out = append(out, fmt.Sprintf("[FILE] %s (%d bytes)", entry.Name(), size))
		}
	}

	if len(out) == 0 {
		return "(directory is empty)", nil
	}

	return strings.Join(out, "\n"), nil
}

// -------------------------------------------------------------
// run_command Tool
// -------------------------------------------------------------
type RunCommandTool struct{}

func (t *RunCommandTool) Name() string         { return "run_command" }
func (t *RunCommandTool) IsDestructive() bool  { return true }

func (t *RunCommandTool) Definition() client.ToolDefinition {
	return client.ToolDefinition{
		Type: "function",
		Function: client.FunctionSchema{
			Name:        "run_command",
			Description: "Run a shell / terminal command on the host system and return combined stdout and stderr.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"command": map[string]interface{}{
						"type":        "string",
						"description": "The exact shell command line string to execute",
					},
				},
				"required": []string{"command"},
			},
		},
	}
}

func (t *RunCommandTool) Execute(ctx context.Context, argsJSON string) (string, error) {
	var args struct {
		Command string `json:"command"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if strings.TrimSpace(args.Command) == "" {
		return "Error: empty command", nil
	}

	// 60-second execution timeout per command
	cmdCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "bash", "-c", args.Command)
	var outBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &outBuf

	err := cmd.Run()
	output := outBuf.String()

	if cmdCtx.Err() == context.DeadlineExceeded {
		return output + "\n[Execution timed out after 60 seconds]", nil
	}

	if err != nil {
		if output == "" {
			output = fmt.Sprintf("[Process exited with error: %v]", err)
		} else {
			output += fmt.Sprintf("\n[Process exited with error: %v]", err)
		}
	}

	if strings.TrimSpace(output) == "" {
		output = "(command completed with no output)"
	}

	return output, nil
}
