#!/usr/bin/env bash
set -euo pipefail

REPO="ProgenyAlpha/reddit-lurker"
BINARY="lurk"
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
echo "Reddit reader for Claude Code"
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

# ─── Choose install mode ──────────────────────────────────────

echo
echo -e "${BOLD}How should Claude Code use lurk?${NC}"
echo
echo "  1) Skill (recommended)"
echo "     Claude runs lurk via Bash when it sees Reddit URLs."
echo "     ~20 tokens of context overhead."
echo
echo "  2) MCP server"
echo "     Claude gets lurk as a native tool — no Bash needed."
echo "     ~438 tokens of context overhead. Pick this for heavy Reddit use."
echo
read -rp "Choose [1/2] (default: 1): " mode
mode="${mode:-1}"

# ─── Install functions ─────────────────────────────────────────

install_binary() {
    mkdir -p "$SKILL_DIR"

    # Handle locked binary (MCP server may be holding it)
    if ! cp "$TMPDIR/$BINARY" "$SKILL_DIR/$BINARY" 2>/dev/null; then
        warn "Binary is locked (MCP server running?). Copying with new name..."
        cp "$TMPDIR/$BINARY" "$SKILL_DIR/${BINARY}.new"
        mv "$SKILL_DIR/${BINARY}.new" "$SKILL_DIR/$BINARY"
    fi
    chmod +x "$SKILL_DIR/$BINARY"
}

install_skill() {
    info "Installing skill to $SKILL_DIR"

    # Remove old lurk skill if it exists under different name
    [ -d "$HOME/.claude/skills/lurk" ] && rm -rf "$HOME/.claude/skills/lurk"

    install_binary

    # Write SKILL.md
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

install_mcp() {
    local lurk_path="$SKILL_DIR/$BINARY"
    info "Configuring MCP server"

    install_binary

    # Edit claude config
    if [ ! -f "$CLAUDE_CONFIG" ]; then
        cat > "$CLAUDE_CONFIG" << MCPEOF
{
  "mcpServers": {
    "lurk": {
      "type": "stdio",
      "command": "$lurk_path",
      "args": ["serve"]
    }
  }
}
MCPEOF
        ok "Created $CLAUDE_CONFIG with lurk MCP server"
        return
    fi

    # Check if already configured
    if grep -q '"lurk"' "$CLAUDE_CONFIG" 2>/dev/null; then
        info "Updating existing lurk entry in $CLAUDE_CONFIG"
    fi

    # Use jq if available, fall back to python3
    if command -v jq &>/dev/null; then
        local tmp
        tmp=$(mktemp)
        jq --arg path "$lurk_path" '.mcpServers.lurk = {"type": "stdio", "command": $path, "args": ["serve"]}' "$CLAUDE_CONFIG" > "$tmp"
        mv "$tmp" "$CLAUDE_CONFIG"
    elif command -v python3 &>/dev/null; then
        python3 << PYEOF
import json
with open("$CLAUDE_CONFIG") as f:
    config = json.load(f)
config.setdefault("mcpServers", {})["lurk"] = {
    "type": "stdio",
    "command": "$lurk_path",
    "args": ["serve"]
}
with open("$CLAUDE_CONFIG", "w") as f:
    json.dump(config, f, indent=2)
PYEOF
    else
        warn "Neither jq nor python3 found. Add lurk MCP server manually:"
        echo
        echo "  Add to $CLAUDE_CONFIG under mcpServers:"
        echo "    \"lurk\": {\"type\": \"stdio\", \"command\": \"$lurk_path\", \"args\": [\"serve\"]}"
        return
    fi

    ok "MCP server configured in $CLAUDE_CONFIG"
    warn "Restart Claude Code to load the new MCP server."
}

# ─── Execute ───────────────────────────────────────────────────

case "$mode" in
    1) install_skill ;;
    2) install_mcp ;;
    *) fail "Invalid choice: $mode" ;;
esac

echo
ok "Done! Start a new Claude Code session and try pasting a Reddit URL."
