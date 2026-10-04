# Rig

A lightweight, CLI harness for interacting with local LLMs hosted in LM Studio.  
Built for personal use in my homelab workflows, it provides local models with safe, controlled access to your files and terminal tools.

## Features

- Tools:
  - `list_dir`: Explore files and folders.
  - `read_file`: Inspect code safely with pagination.
  - `write_file`: Create or edit files.
    - Prompts for confirmation before writing.
  - `run_command`: Execute shell commands with timeout protection.
    - Prompts for confirmation before running.
- Interactive REPL or One-Shot: Use as an interactive terminal assistant or a direct task runner.
- Automatically detects the model currently loaded in LM Studio.
- Pure Go: Compiles to a single native binary.

## Configuration

Settings can be customized directly in [config/config.json](file:///home/mitchfen/Projects/rig/config/config.json):
```json
{
  "endpoint": "http://127.0.0.1:1234/v1",
  "model": "",
  "instructions_file": "instructions.md",
  "context_files": [
    "~/Projects/homelab/README.md"
  ],
  "max_context_tokens": 16384,
  "max_steps": 10,
  "max_tool_output_chars": 3000
}
```

- `context_files`: Array of markdown files (supporting `~/` expansion or paths relative to `config/`) automatically pre-loaded into the agent's memory on startup.
- `instructions_file`: Base persona, tone, and operational guidelines.

### Custom Instructions (`instructions.md`)
You can define homelab rules, preferred tools, and communication style in [config/instructions.md](file:///home/mitchfen/Projects/rig/config/instructions.md). These are automatically injected into the agent's prompt on startup.

---

## Usage

### 1. Build
```bash
./build.sh
```
This compiles the binary and packages a self-contained distribution folder in `output/`:
- `output/rig`: Executable binary
- `output/config.json`: Configuration file
- `output/instructions.md`: Custom instructions

### 2. Run
```bash
./output/rig
```

### REPL Commands
- `/context`: Displays token usage and percentage of your context limit.
- `/reset` or `/clear`: Resets conversation history back to initial prompt.
- `/exit` or `quit`: Exit the REPL.
