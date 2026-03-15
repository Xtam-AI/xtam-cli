# xtam

Private artifact registry CLI for XTAM. Install and manage skills, MCP servers, CLI tools, configs, and templates — restricted to `@xtam.ai` employees.

## Install

```bash
brew tap xtam-ai/tap
brew install xtam
```

Or download directly from [releases](https://github.com/Xtam-AI/xtam-cli/releases):

```bash
curl -sL https://github.com/Xtam-AI/xtam-cli/releases/latest/download/xtam-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m | sed 's/x86_64/amd64/' | sed 's/aarch64/arm64/').tar.gz | tar xz
sudo mv xtam /usr/local/bin/
```

## Authenticate

```bash
xtam login
```

Opens your browser to sign in with your `@xtam.ai` Google account. Only `@xtam.ai` emails are accepted.

## Usage

```bash
xtam catalog              # List available artifacts
xtam catalog --type skill  # Filter by type
xtam info <name>           # Show artifact details
xtam install <name>        # Install an artifact
xtam list                  # Show installed artifacts
xtam update --all          # Update all installed artifacts
xtam uninstall <name>      # Remove an artifact
```

## Publishing (admins only)

```bash
xtam publish-setup                          # Store GitHub PAT (one-time)
xtam publish ./my-skill --type skill --version 1.0.0 --description "My skill"
```

## Artifact Types

| Type | Installs to |
|------|------------|
| `skill` | `~/.claude/skills/<name>/` |
| `mcp-server` | `~/.xtam/mcp-servers/<name>/` + `.mcp.json` config |
| `cli-tool` | `~/.local/bin/<name>` |
| `config` | Target paths specified in manifest |
| `template` | Current directory |
