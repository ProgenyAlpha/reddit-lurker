#!/usr/bin/env bash
set -euo pipefail

REPO="ProgenyAlpha/reddit-lurker"
BINARY="lurk"
INSTALL_DIR="$HOME/.local/bin"
SKILL_DIR="$HOME/.claude/skills/reddit"
CLAUDE_CONFIG="$HOME/.claude.json"
VERSION="1.0.0"

# Colors (if terminal supports them)
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

info()  { echo -e "${CYAN}→${NC} $*"; }
ok()    { echo -e "${GREEN}✓${NC} $*"; }
warn()  { echo -e "${YELLOW}!${NC} $*"; }
fail()  { echo -e "${RED}✗${NC} $*"; exit 1; }

echo -e "${BOLD}reddit-lurker${NC} v${VERSION}"
echo "Reddit reader for LLM code editors"
echo

# ─── Detect platform ───────────────────────────────────────────

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
    x86_64)        ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) fail "Unsupported architecture: $ARCH" ;;
esac
info "Platform: ${OS}/${ARCH}"

# ─── Get the binary ──────────────────────────────────────────

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

if [ -f "./main.go" ] && [ -f "./go.mod" ]; then
    # Running from cloned repo — build from source
    info "Source detected, building from source..."
    command -v go &>/dev/null || fail "Go is required. Install from https://go.dev/dl/"
    go build -ldflags "-s -w" -o "$TMPDIR/$BINARY" .
    ok "Built $BINARY ($(du -h "$TMPDIR/$BINARY" | cut -f1))"
else
    # Running via curl | bash — download prebuilt binary
    ARCHIVE="lurk-${OS}-${ARCH}.tar.gz"
    URL="https://github.com/${REPO}/releases/download/v${VERSION}/${ARCHIVE}"

    info "Downloading lurk v${VERSION} for ${OS}/${ARCH}..."

    if command -v curl &>/dev/null; then
        curl -fsSL "$URL" -o "$TMPDIR/$ARCHIVE" || fail "Download failed. Is v${VERSION} released? Check https://github.com/${REPO}/releases"
    elif command -v wget &>/dev/null; then
        wget -q "$URL" -O "$TMPDIR/$ARCHIVE" || fail "Download failed. Is v${VERSION} released? Check https://github.com/${REPO}/releases"
    else
        fail "Need curl or wget to download the binary"
    fi

    tar xzf "$TMPDIR/$ARCHIVE" -C "$TMPDIR"
    chmod +x "$TMPDIR/$BINARY"
    ok "Downloaded $BINARY"
fi

# ─── Install binary to PATH ──────────────────────────────────

install_binary_to_path() {
    mkdir -p "$INSTALL_DIR"
    if ! cp "$TMPDIR/$BINARY" "$INSTALL_DIR/$BINARY" 2>/dev/null; then
        warn "Binary is locked (MCP server running?). Copying with new name..."
        cp "$TMPDIR/$BINARY" "$INSTALL_DIR/${BINARY}.new"
        mv "$INSTALL_DIR/${BINARY}.new" "$INSTALL_DIR/$BINARY"
    fi
    chmod +x "$INSTALL_DIR/$BINARY"
    ok "Binary installed to $INSTALL_DIR/$BINARY"

    # Check if ~/.local/bin is in PATH
    if ! echo "$PATH" | tr ':' '\n' | grep -qx "$INSTALL_DIR"; then
        warn "$INSTALL_DIR is not in your PATH."
        warn "Add it: export PATH=\"\$HOME/.local/bin:\$PATH\""
    fi
}

install_binary_to_skill() {
    mkdir -p "$SKILL_DIR"
    if ! cp "$TMPDIR/$BINARY" "$SKILL_DIR/$BINARY" 2>/dev/null; then
        warn "Binary is locked (MCP server running?). Copying with new name..."
        cp "$TMPDIR/$BINARY" "$SKILL_DIR/${BINARY}.new"
        mv "$SKILL_DIR/${BINARY}.new" "$SKILL_DIR/$BINARY"
    fi
    chmod +x "$SKILL_DIR/$BINARY"
}

# ─── Choose editor ────────────────────────────────────────────

echo -e "${BOLD}Which editor(s) should lurk integrate with?${NC}"
echo
echo "  1) Claude Code"
echo "  2) Cursor"
echo "  3) Windsurf"
echo "  4) VS Code (Copilot Chat)"
echo "  5) Cline"
echo "  6) Zed"
echo "  7) All of the above"
echo "  8) Just install the binary (I'll configure it myself)"
echo
read -rp "Choose [1-8] (default: 1): " editor
editor="${editor:-1}"

# ─── Choose mode (for editors that support both) ─────────────

choose_mode() {
    local editor_name="$1"
    echo
    echo -e "${BOLD}How should ${editor_name} use lurk?${NC}"
    echo
    echo "  1) Skill (recommended) — Claude Code only"
    echo "     Claude runs lurk via Bash when it sees Reddit URLs."
    echo "     ~20 tokens of context overhead."
    echo
    echo "  2) MCP server"
    echo "     Native tool integration — no Bash needed."
    echo "     ~438 tokens of context overhead. Pick this for heavy Reddit use."
    echo
    read -rp "Choose [1/2] (default: 1): " mode
    mode="${mode:-1}"
    echo "$mode"
}

# ─── Skill install (Claude Code only) ────────────────────────

install_skill() {
    info "Installing skill to $SKILL_DIR"

    [ -d "$HOME/.claude/skills/lurk" ] && rm -rf "$HOME/.claude/skills/lurk"

    install_binary_to_skill

    cat > "$SKILL_DIR/SKILL.md" << 'SKILLEOF'
# Lurk — Reddit Reader

Read Reddit threads, browse subreddits, search posts, and check user profiles. Full comment trees with automatic expansion of collapsed threads. No auth needed.

## When to Use

- User pastes a Reddit URL
- User asks about a Reddit thread, subreddit, or user
- User wants to search Reddit for information
- User asks "what's on r/..." or "check Reddit for..."

## Commands

```bash
# Read a full thread with all comments
~/.claude/skills/reddit/lurk thread "<reddit_url>" --compact

# Browse a subreddit
~/.claude/skills/reddit/lurk subreddit <name> --sort hot --limit 25 --compact

# Search Reddit (optionally within a subreddit)
~/.claude/skills/reddit/lurk search "<query>" --sub <subreddit> --sort relevance --limit 25 --compact

# View user activity
~/.claude/skills/reddit/lurk user <username> --limit 10 --compact

# Subreddit info
~/.claude/skills/reddit/lurk subreddit <name> --info --compact
```

## Sort Options

- Subreddit: hot, new, top, rising, controversial
- Search: relevance, hot, top, new, comments
- Time filter (for top/controversial): hour, day, week, month, year, all

## Pagination

Results include a `--after` token when more pages are available.

## Output

Always use `--compact` for efficient token usage. Add `--json` for raw JSON if needed.
SKILLEOF

    ok "Skill installed"
}

# ─── MCP config helpers ──────────────────────────────────────

# Write or merge JSON MCP config into a file
# $1 = config file path
# $2 = top-level key (mcpServers, servers, context_servers)
# $3 = lurk binary path
# $4 = extra fields (optional, for Cline)
write_mcp_config() {
    local config_file="$1"
    local top_key="$2"
    local lurk_path="$3"
    local extra="${4:-}"

    if [ ! -f "$config_file" ]; then
        mkdir -p "$(dirname "$config_file")"
        if [ -n "$extra" ]; then
            cat > "$config_file" << MCPEOF
{
  "$top_key": {
    "lurk": {
      "command": "$lurk_path",
      "args": ["serve"]${extra}
    }
  }
}
MCPEOF
        else
            cat > "$config_file" << MCPEOF
{
  "$top_key": {
    "lurk": {
      "command": "$lurk_path",
      "args": ["serve"]
    }
  }
}
MCPEOF
        fi
        ok "Created $config_file"
        return
    fi

    # File exists — merge using jq or python3
    if command -v jq &>/dev/null; then
        local tmp
        tmp=$(mktemp)
        if [ "$top_key" = "context_servers" ]; then
            jq --arg path "$lurk_path" --arg key "$top_key" \
                '.[$key].lurk = {"command": $path, "args": ["serve"]}' \
                "$config_file" > "$tmp"
        else
            jq --arg path "$lurk_path" --arg key "$top_key" \
                '.[$key].lurk = {"command": $path, "args": ["serve"]}' \
                "$config_file" > "$tmp"
        fi
        mv "$tmp" "$config_file"
        ok "Updated $config_file"
    elif command -v python3 &>/dev/null; then
        python3 << PYEOF
import json
with open("$config_file") as f:
    config = json.load(f)
config.setdefault("$top_key", {})["lurk"] = {
    "command": "$lurk_path",
    "args": ["serve"]
}
with open("$config_file", "w") as f:
    json.dump(config, f, indent=2)
PYEOF
        ok "Updated $config_file"
    else
        warn "Neither jq nor python3 found. Add lurk manually to $config_file:"
        echo "  Under \"$top_key\": {\"lurk\": {\"command\": \"$lurk_path\", \"args\": [\"serve\"]}}"
    fi
}

# ─── Editor-specific installers ──────────────────────────────

install_claude_mcp() {
    local lurk_path="$SKILL_DIR/$BINARY"
    info "Configuring Claude Code MCP server"
    install_binary_to_skill
    write_mcp_config "$CLAUDE_CONFIG" "mcpServers" "$lurk_path"
    warn "Restart Claude Code to load the new MCP server."
}

install_claude() {
    local mode
    mode=$(choose_mode "Claude Code")
    case "$mode" in
        1) install_skill ;;
        2) install_claude_mcp ;;
        *) fail "Invalid choice: $mode" ;;
    esac
}

install_cursor() {
    local config_dir="$HOME/.cursor"
    local config_file="$config_dir/mcp.json"
    info "Configuring Cursor MCP server"
    install_binary_to_path
    write_mcp_config "$config_file" "mcpServers" "$INSTALL_DIR/$BINARY"
    ok "Cursor configured (global)"
    warn "Restart Cursor to load the new MCP server."
    warn "MCP tools are available in Agent mode and Composer, not regular chat."
}

install_windsurf() {
    local config_dir="$HOME/.codeium/windsurf"
    local config_file="$config_dir/mcp_config.json"
    info "Configuring Windsurf MCP server"
    install_binary_to_path
    write_mcp_config "$config_file" "mcpServers" "$INSTALL_DIR/$BINARY"
    ok "Windsurf configured"
    warn "Restart Windsurf to load the new MCP server."
}

install_vscode() {
    local config_file="$HOME/.vscode/mcp.json"

    # VS Code user-level MCP config location varies by OS
    if [ "$OS" = "darwin" ]; then
        config_file="$HOME/Library/Application Support/Code/User/mcp.json"
    elif [ "$OS" = "linux" ]; then
        config_file="$HOME/.config/Code/User/mcp.json"
    fi

    info "Configuring VS Code (Copilot Chat) MCP server"
    install_binary_to_path

    # VS Code uses "servers" not "mcpServers"
    write_mcp_config "$config_file" "servers" "$INSTALL_DIR/$BINARY"
    ok "VS Code configured"
    warn "Restart VS Code to load the new MCP server."
    warn "Requires VS Code 1.99+ and Copilot Chat in Agent mode."
}

install_cline() {
    local config_file

    if [ "$OS" = "darwin" ]; then
        config_file="$HOME/Library/Application Support/Code/User/globalStorage/saoudrizwan.claude-dev/settings/cline_mcp_settings.json"
    elif [ "$OS" = "linux" ]; then
        config_file="$HOME/.config/Code/User/globalStorage/saoudrizwan.claude-dev/settings/cline_mcp_settings.json"
    else
        fail "Cline config path unknown for $OS. Configure manually via the Cline UI."
    fi

    info "Configuring Cline MCP server"
    install_binary_to_path
    write_mcp_config "$config_file" "mcpServers" "$INSTALL_DIR/$BINARY" ',
      "disabled": false,
      "alwaysAllow": []'
    ok "Cline configured"
    warn "Restart VS Code to load the new MCP server."
}

install_zed() {
    local config_file="$HOME/.config/zed/settings.json"
    info "Configuring Zed MCP server"
    install_binary_to_path

    if [ ! -f "$config_file" ]; then
        mkdir -p "$(dirname "$config_file")"
        cat > "$config_file" << MCPEOF
{
  "context_servers": {
    "lurk": {
      "command": "$INSTALL_DIR/$BINARY",
      "args": ["serve"]
    }
  }
}
MCPEOF
        ok "Created $config_file"
    elif command -v jq &>/dev/null; then
        local tmp
        tmp=$(mktemp)
        jq --arg path "$INSTALL_DIR/$BINARY" \
            '.context_servers.lurk = {"command": $path, "args": ["serve"]}' \
            "$config_file" > "$tmp"
        mv "$tmp" "$config_file"
        ok "Updated $config_file"
    elif command -v python3 &>/dev/null; then
        python3 << PYEOF
import json
with open("$config_file") as f:
    config = json.load(f)
config.setdefault("context_servers", {})["lurk"] = {
    "command": "$INSTALL_DIR/$BINARY",
    "args": ["serve"]
}
with open("$config_file", "w") as f:
    json.dump(config, f, indent=2)
PYEOF
        ok "Updated $config_file"
    else
        warn "Neither jq nor python3 found. Add lurk manually to $config_file under context_servers."
    fi
    warn "Restart Zed to load the new MCP server."
}

install_all() {
    echo
    info "Installing for all supported editors..."
    echo

    # Claude Code gets the skill/MCP choice
    install_claude

    # Everything else gets MCP via binary in PATH
    install_cursor
    install_windsurf
    install_vscode
    install_cline
    install_zed
}

install_binary_only() {
    info "Installing binary only"
    install_binary_to_path
    echo
    ok "Binary installed to $INSTALL_DIR/$BINARY"
    echo "Configure your editor manually. See README for config examples."
}

# ─── Execute ───────────────────────────────────────────────────

case "$editor" in
    1) install_claude ;;
    2) install_cursor ;;
    3) install_windsurf ;;
    4) install_vscode ;;
    5) install_cline ;;
    6) install_zed ;;
    7) install_all ;;
    8) install_binary_only ;;
    *) fail "Invalid choice: $editor" ;;
esac

echo
ok "Done! Restart your editor and try pasting a Reddit URL."
