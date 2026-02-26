#!/usr/bin/env node
"use strict";

const fs = require("fs");
const path = require("path");
const readline = require("readline");

const SKILL_DIR = path.join(process.env.HOME, ".claude", "skills", "reddit");
const CLAUDE_CONFIG = path.join(process.env.HOME, ".claude.json");
const BIN_PATH = path.join(__dirname, "bin", "lurk");

function ask(question) {
  const rl = readline.createInterface({ input: process.stdin, output: process.stdout });
  return new Promise((resolve) => rl.question(question, (a) => { rl.close(); resolve(a.trim()); }));
}

function installSkill() {
  fs.mkdirSync(SKILL_DIR, { recursive: true });

  // Copy binary
  fs.copyFileSync(BIN_PATH, path.join(SKILL_DIR, "lurk"));
  fs.chmodSync(path.join(SKILL_DIR, "lurk"), 0o755);

  // Write SKILL.md
  fs.writeFileSync(path.join(SKILL_DIR, "SKILL.md"), `# Lurk — Reddit Reader

Read Reddit threads, browse subreddits, search posts, and check user profiles. Full comment trees with automatic expansion of collapsed threads. No auth needed.

## When to Use

- User pastes a Reddit URL
- User asks about a Reddit thread, subreddit, or user
- User wants to search Reddit for information
- User asks "what's on r/..." or "check Reddit for..."

## Commands

\`\`\`bash
~/.claude/skills/reddit/lurk thread "<reddit_url>"
~/.claude/skills/reddit/lurk subreddit <name> --sort hot --limit 25
~/.claude/skills/reddit/lurk search "<query>" --sub <subreddit> --limit 25
~/.claude/skills/reddit/lurk user <username> --limit 10
~/.claude/skills/reddit/lurk subreddit <name> --info
\`\`\`

## Sort Options

- Subreddit: hot, new, top, rising, controversial
- Search: relevance, hot, top, new, comments
- Time filter: hour, day, week, month, year, all

## Pagination

Results include a \`--after\` token when more pages are available.

## Output

Default is markdown. Add \`--json\` for raw JSON or \`--compact\` for AI-optimized compact notation.
`);

  console.log(`✓ Skill installed to ${SKILL_DIR}`);
}

function installMCP() {
  const lurkPath = path.join(SKILL_DIR, "lurk");

  // Ensure binary is in skill dir
  if (!fs.existsSync(lurkPath)) {
    fs.mkdirSync(SKILL_DIR, { recursive: true });
    fs.copyFileSync(BIN_PATH, lurkPath);
    fs.chmodSync(lurkPath, 0o755);
  }

  let config = {};
  if (fs.existsSync(CLAUDE_CONFIG)) {
    config = JSON.parse(fs.readFileSync(CLAUDE_CONFIG, "utf8"));
  }

  if (!config.mcpServers) config.mcpServers = {};
  config.mcpServers.lurk = {
    type: "stdio",
    command: lurkPath,
    args: ["serve"],
  };

  fs.writeFileSync(CLAUDE_CONFIG, JSON.stringify(config, null, 2));
  console.log(`✓ MCP server configured in ${CLAUDE_CONFIG}`);
  console.log(`  Restart Claude Code to load the new MCP server.`);
}

async function main() {
  console.log("reddit-lurker — Claude Code setup\n");
  console.log("  1) Skill (recommended) — ~20 tokens overhead");
  console.log("  2) MCP server — ~438 tokens overhead, native tool calls\n");

  const choice = await ask("Choose [1/2] (default: 1): ") || "1";

  switch (choice) {
    case "1": installSkill(); break;
    case "2": installMCP(); break;
    default: console.error("Invalid choice"); process.exit(1);
  }

  console.log("\n✓ Done! Start a new Claude Code session and paste a Reddit URL.");
}

main();
