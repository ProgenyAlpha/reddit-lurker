#!/usr/bin/env bash
set -euo pipefail

BINARY="lurk"
SKILL_DIR="$HOME/.claude/skills/lurk"
CLAUDE_CONFIG="$HOME/.claude.json"

echo "Lurk — Reddit Reader for Claude Code"
echo "======================================"
echo

# Detect OS and arch
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

echo "Detected: ${OS}/${ARCH}"
echo

# Check if binary exists in current dir
if [ ! -f "./$BINARY" ]; then
    echo "Building from source..."
    if ! command -v go &>/dev/null; then
        echo "Error: Go is required to build from source."
        echo "Install Go from https://go.dev/dl/"
        exit 1
    fi
    go build -ldflags "-s -w" -o "$BINARY" .
    echo "Built successfully."
fi

install_skill() {
    mkdir -p "$SKILL_DIR"
    cp "$BINARY" "$SKILL_DIR/$BINARY"
    chmod +x "$SKILL_DIR/$BINARY"
    cp skill/SKILL.md "$SKILL_DIR/SKILL.md"
    echo "Installed lurk skill to $SKILL_DIR"
    echo "Claude will automatically use it when you paste Reddit URLs."
}

install_mcp() {
    local lurk_path

    # Install binary to ~/.local/bin
    mkdir -p "$HOME/.local/bin"
    cp "$BINARY" "$HOME/.local/bin/$BINARY"
    chmod +x "$HOME/.local/bin/$BINARY"
    lurk_path="$HOME/.local/bin/$BINARY"

    # Add MCP config
    if [ -f "$CLAUDE_CONFIG" ]; then
        # Check if lurk is already configured
        if python3 -c "import json; c=json.load(open('$CLAUDE_CONFIG')); exit(0 if 'lurk' in c.get('mcpServers',{}) else 1)" 2>/dev/null; then
            echo "Lurk MCP server already configured in $CLAUDE_CONFIG"
            return
        fi
        # Add to existing config
        python3 -c "
import json
with open('$CLAUDE_CONFIG') as f:
    config = json.load(f)
config.setdefault('mcpServers', {})['lurk'] = {
    'command': '$lurk_path',
    'args': ['serve']
}
with open('$CLAUDE_CONFIG', 'w') as f:
    json.dump(config, f, indent=2)
"
    else
        # Create new config
        python3 -c "
import json
config = {'mcpServers': {'lurk': {'command': '$lurk_path', 'args': ['serve']}}}
with open('$CLAUDE_CONFIG', 'w') as f:
    json.dump(config, f, indent=2)
"
    fi
    echo "Added lurk MCP server to $CLAUDE_CONFIG"
    echo "Restart Claude Code to pick up the new MCP server."
}

echo
echo "Install modes:"
echo "  1) Skill only (recommended) — Claude runs lurk via Bash when it sees Reddit URLs"
echo "  2) MCP server only — Claude gets lurk as a native tool (adds ~800 tokens to context)"
echo "  3) Both — MCP for tool calls, skill as fallback"
echo
read -rp "Choose [1/2/3]: " mode

case "$mode" in
    1|"")
        install_skill
        ;;
    2)
        install_mcp
        ;;
    3)
        install_skill
        install_mcp
        echo
        echo "Installed both skill and MCP server."
        ;;
    *)
        echo "Invalid choice."
        exit 1
        ;;
esac

echo
echo "Done. Try: lurk thread \"https://www.reddit.com/r/ClaudeAI/comments/...\""
