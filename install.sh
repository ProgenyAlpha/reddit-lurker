#!/usr/bin/env bash
set -euo pipefail

REPO="ProgenyAlpha/reddit-lurker"
BINARY="lurk"
INSTALL_DIR="$HOME/.local/bin"
SKILL_DIR="$HOME/.claude/skills/reddit"
CLAUDE_CONFIG="$HOME/.claude.json"
BINARY_INSTALLED=false

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

# ─── Detect version ───────────────────────────────────────────

if [ -f "./main.go" ] && [ -f "./go.mod" ]; then
    # In cloned repo — use git tag if available
    VERSION=$(git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//' || echo "dev")
else
    # Fetch latest release tag from GitHub
    if command -v curl &>/dev/null; then
        VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null | grep '"tag_name"' | sed 's/.*"v\(.*\)".*/\1/' || echo "")
    elif command -v wget &>/dev/null; then
        VERSION=$(wget -qO- "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null | grep '"tag_name"' | sed 's/.*"v\(.*\)".*/\1/' || echo "")
    fi
    [ -z "$VERSION" ] && fail "Could not determine latest version. Check https://github.com/${REPO}/releases"
fi

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
    go build -ldflags "-s -w -X main.version=${VERSION}" -o "$TMPDIR/$BINARY" .
    ok "Built $BINARY ($(du -h "$TMPDIR/$BINARY" | cut -f1))"
else
    # Running via curl | bash — download prebuilt binary
    ARCHIVE="lurk-${OS}-${ARCH}.tar.gz"
    CHECKSUM_FILE="checksums.txt"
    URL="https://github.com/${REPO}/releases/download/v${VERSION}/${ARCHIVE}"
    CHECKSUM_URL="https://github.com/${REPO}/releases/download/v${VERSION}/${CHECKSUM_FILE}"

    info "Downloading lurk v${VERSION} for ${OS}/${ARCH}..."

    if command -v curl &>/dev/null; then
        curl -fsSL "$URL" -o "$TMPDIR/$ARCHIVE" || fail "Download failed. Is v${VERSION} released? Check https://github.com/${REPO}/releases"
        curl -fsSL "$CHECKSUM_URL" -o "$TMPDIR/$CHECKSUM_FILE" 2>/dev/null || true
    elif command -v wget &>/dev/null; then
        wget -q "$URL" -O "$TMPDIR/$ARCHIVE" || fail "Download failed. Is v${VERSION} released? Check https://github.com/${REPO}/releases"
        wget -q "$CHECKSUM_URL" -O "$TMPDIR/$CHECKSUM_FILE" 2>/dev/null || true
    else
        fail "Need curl or wget to download the binary"
    fi

    # Verify checksum if available
    if [ -f "$TMPDIR/$CHECKSUM_FILE" ]; then
        expected=$(grep "$ARCHIVE" "$TMPDIR/$CHECKSUM_FILE" | awk '{print $1}')
        if [ -n "$expected" ]; then
            if command -v sha256sum &>/dev/null; then
                actual=$(sha256sum "$TMPDIR/$ARCHIVE" | awk '{print $1}')
            elif command -v shasum &>/dev/null; then
                actual=$(shasum -a 256 "$TMPDIR/$ARCHIVE" | awk '{print $1}')
            else
                actual=""
                warn "No sha256sum or shasum found, skipping checksum verification"
            fi
            if [ -n "$actual" ] && [ "$expected" != "$actual" ]; then
                fail "Checksum verification failed (expected $expected, got $actual)"
            elif [ -n "$actual" ]; then
                ok "Checksum verified"
            fi
        fi
    fi

    tar xzf "$TMPDIR/$ARCHIVE" -C "$TMPDIR"
    chmod +x "$TMPDIR/$BINARY"
    ok "Downloaded $BINARY"
fi

# ─── Install binary helpers ──────────────────────────────────

ensure_binary_in_path() {
    if [ "$BINARY_INSTALLED" = true ]; then
        return
    fi
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
    BINARY_INSTALLED=true
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
read -rp "Choose [1-8, comma-separated] (default: 1): " editor_input </dev/tty
editor_input="${editor_input:-1}"

# Parse comma-separated choices into array
IFS=',' read -ra editor_choices <<< "$editor_input"
# Trim whitespace from each choice
for i in "${!editor_choices[@]}"; do
    editor_choices[$i]=$(echo "${editor_choices[$i]}" | tr -d ' ')
done
# Validate all choices
for choice in "${editor_choices[@]}"; do
    case "$choice" in
        [1-8]) ;;
        *) fail "Invalid choice: $choice" ;;
    esac
done

# ─── Choose mode (for editors that support both) ─────────────

choose_mode() {
    local editor_name="$1"
    {
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
    } >&2
    read -rp "Choose [1/2] (default: 1): " mode </dev/tty
    mode="${mode:-1}"
    printf '%s' "$mode"
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
# $4 = extra JSON fields (optional, e.g. '"disabled": false')
write_mcp_config() {
    local config_file="$1"
    local top_key="$2"
    local lurk_path="$3"
    local extra="${4:-}"

    local extra_jq=""
    local extra_py=""
    if [ -n "$extra" ]; then
        extra_jq=", ${extra}"
        extra_py=", ${extra}"
    fi

    if [ ! -f "$config_file" ]; then
        mkdir -p "$(dirname "$config_file")"
        if [ -n "$extra" ]; then
            cat > "$config_file" << MCPEOF
{
  "$top_key": {
    "lurk": {
      "command": "$lurk_path",
      "args": ["serve"],
      ${extra}
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
        if [ -n "$extra" ]; then
            # Build the full object with extra fields using jq
            jq --arg path "$lurk_path" --arg key "$top_key" \
                '.[$key].lurk = (.[$key].lurk // {}) * {"command": $path, "args": ["serve"]} * {'"$extra"'}' \
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
entry = {"command": "$lurk_path", "args": ["serve"]}
$([ -n "$extra" ] && echo "entry.update({$extra_py})")
config.setdefault("$top_key", {})["lurk"] = entry
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

configure_cursor() {
    local config_file="$HOME/.cursor/mcp.json"
    info "Configuring Cursor MCP server"
    write_mcp_config "$config_file" "mcpServers" "$INSTALL_DIR/$BINARY"
    ok "Cursor configured (global)"
    warn "Restart Cursor to load the new MCP server."
    warn "MCP tools are available in Agent mode and Composer, not regular chat."
}

configure_windsurf() {
    local config_file="$HOME/.codeium/windsurf/mcp_config.json"
    info "Configuring Windsurf MCP server"
    write_mcp_config "$config_file" "mcpServers" "$INSTALL_DIR/$BINARY"
    ok "Windsurf configured"
    warn "Restart Windsurf to load the new MCP server."
}

configure_vscode() {
    local config_file

    # VS Code user-level MCP config location varies by OS
    if [ "$OS" = "darwin" ]; then
        config_file="$HOME/Library/Application Support/Code/User/mcp.json"
    else
        config_file="$HOME/.config/Code/User/mcp.json"
    fi

    info "Configuring VS Code (Copilot Chat) MCP server"

    # VS Code uses "servers" not "mcpServers"
    write_mcp_config "$config_file" "servers" "$INSTALL_DIR/$BINARY"
    ok "VS Code configured"
    warn "Restart VS Code to load the new MCP server."
    warn "Requires VS Code 1.99+ and Copilot Chat in Agent mode."
}

configure_cline() {
    local config_file

    if [ "$OS" = "darwin" ]; then
        config_file="$HOME/Library/Application Support/Code/User/globalStorage/saoudrizwan.claude-dev/settings/cline_mcp_settings.json"
    elif [ "$OS" = "linux" ]; then
        config_file="$HOME/.config/Code/User/globalStorage/saoudrizwan.claude-dev/settings/cline_mcp_settings.json"
    else
        fail "Cline config path unknown for $OS. Configure manually via the Cline UI."
    fi

    info "Configuring Cline MCP server"
    write_mcp_config "$config_file" "mcpServers" "$INSTALL_DIR/$BINARY" '"disabled": false, "alwaysAllow": []'
    ok "Cline configured"
    warn "Restart VS Code to load the new MCP server."
}

configure_zed() {
    local config_file="$HOME/.config/zed/settings.json"
    info "Configuring Zed MCP server"
    write_mcp_config "$config_file" "context_servers" "$INSTALL_DIR/$BINARY"
    warn "Restart Zed to load the new MCP server."
}

# Standalone editor installers (binary + config)
install_cursor()   { ensure_binary_in_path; configure_cursor; }
install_windsurf() { ensure_binary_in_path; configure_windsurf; }
install_vscode()   { ensure_binary_in_path; configure_vscode; }
install_cline()    { ensure_binary_in_path; configure_cline; }
install_zed()      { ensure_binary_in_path; configure_zed; }

install_all() {
    echo
    info "Installing for all supported editors..."
    echo

    # Claude Code gets the skill/MCP choice
    install_claude

    # Install binary once for all other editors
    ensure_binary_in_path

    # Configure each editor
    configure_cursor
    configure_windsurf
    configure_vscode
    configure_cline
    configure_zed
}

install_binary_only() {
    info "Installing binary only"
    ensure_binary_in_path
    echo
    echo "Configure your editor manually. See README for config examples."
}

# ─── Execute ───────────────────────────────────────────────────

for editor in "${editor_choices[@]}"; do
    case "$editor" in
        1) install_claude ;;
        2) install_cursor ;;
        3) install_windsurf ;;
        4) install_vscode ;;
        5) install_cline ;;
        6) install_zed ;;
        7) install_all ;;
        8) install_binary_only ;;
    esac
done

echo
ok "Done! Restart your editor and try pasting a Reddit URL."
