# Rig

A lightweight, zero-dependency CLI harness for interacting with local LLMs hosted in LM Studio.  
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
- Pure Go: Compiles to a single zero-dependency native binary.

---

## Usage

### 1. Build
```bash
go build -o rig main.go
```

### 2. Run
```bash
./rig
```
*Options:*
- `./rig -y`: Run in autonomous mode (auto-approve all tool executions).
- `./rig -model qwen/qwen3.8-27b`: Override model selection.
- `./rig -url http://127.0.0.1:1234/v1`: Custom LM Studio address.
