#Requires -Version 5.1
<#
.SYNOPSIS
    Install reddit-lurker (lurk) on Windows.
.DESCRIPTION
    Downloads the latest release binary from GitHub and configures
    editor MCP integration. Supports Claude Code, Cursor, Windsurf,
    VS Code, Cline, and Zed.
.EXAMPLE
    irm https://raw.githubusercontent.com/ProgenyAlpha/reddit-lurker/master/install.ps1 | iex
#>

$ErrorActionPreference = "Stop"
$Repo = "ProgenyAlpha/reddit-lurker"
$Binary = "lurk.exe"
$InstallDir = Join-Path $env:LOCALAPPDATA "lurk"
$BinaryPath = Join-Path $InstallDir $Binary

# ─── Helpers ──────────────────────────────────────────────────

function Write-Info  { param($msg) Write-Host "→ $msg" -ForegroundColor Cyan }
function Write-Ok    { param($msg) Write-Host "✓ $msg" -ForegroundColor Green }
function Write-Warn  { param($msg) Write-Host "! $msg" -ForegroundColor Yellow }
function Write-Fail  { param($msg) Write-Host "✗ $msg" -ForegroundColor Red; exit 1 }

# ─── Detect version ──────────────────────────────────────────

Write-Info "Fetching latest version..."
try {
    $release = Invoke-RestMethod "https://api.github.com/repos/$Repo/releases/latest"
    $Version = $release.tag_name -replace '^v', ''
} catch {
    Write-Fail "Could not determine latest version. Check https://github.com/$Repo/releases"
}

Write-Host ""
Write-Host "reddit-lurker v$Version" -ForegroundColor White -NoNewline
Write-Host ""
Write-Host "Reddit reader for LLM code editors"
Write-Host ""

# ─── Detect architecture ─────────────────────────────────────

$Arch = if ([Environment]::Is64BitOperatingSystem) {
    if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }
} else {
    Write-Fail "32-bit Windows is not supported"
}
Write-Info "Platform: windows/$Arch"

# ─── Download binary ─────────────────────────────────────────

$Archive = "lurk-windows-$Arch.zip"
$Url = "https://github.com/$Repo/releases/download/v$Version/$Archive"
$ChecksumUrl = "https://github.com/$Repo/releases/download/v$Version/checksums.txt"
$TmpDir = Join-Path ([System.IO.Path]::GetTempPath()) "lurk-install-$(Get-Random)"
New-Item -ItemType Directory -Path $TmpDir -Force | Out-Null

try {
    Write-Info "Downloading lurk v$Version for windows/$Arch..."
    Invoke-WebRequest -Uri $Url -OutFile (Join-Path $TmpDir $Archive) -UseBasicParsing

    # Verify checksum if available
    try {
        $checksums = Invoke-WebRequest -Uri $ChecksumUrl -UseBasicParsing
        $expected = ($checksums.Content -split "`n" | Where-Object { $_ -match $Archive } | ForEach-Object { ($_ -split '\s+')[0] })
        if ($expected) {
            $actual = (Get-FileHash (Join-Path $TmpDir $Archive) -Algorithm SHA256).Hash.ToLower()
            if ($actual -ne $expected.ToLower()) {
                Write-Fail "Checksum verification failed"
            }
            Write-Ok "Checksum verified"
        }
    } catch {
        # Checksum file not available, skip
    }

    Expand-Archive -Path (Join-Path $TmpDir $Archive) -DestinationPath $TmpDir -Force
    Write-Ok "Downloaded lurk"
} catch {
    Write-Fail "Download failed. Is v$Version released? Check https://github.com/$Repo/releases"
}

# ─── Install binary ──────────────────────────────────────────

New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
Copy-Item (Join-Path $TmpDir $Binary) $BinaryPath -Force
Write-Ok "Binary installed to $BinaryPath"

# Add to PATH if not already there
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$UserPath;$InstallDir", "User")
    $env:Path = "$env:Path;$InstallDir"
    Write-Ok "Added $InstallDir to PATH"
    Write-Warn "Restart your terminal for PATH changes to take effect."
} else {
    Write-Ok "$InstallDir already in PATH"
}

# ─── Choose editor ────────────────────────────────────────────

Write-Host ""
Write-Host "Which editor(s) should lurk integrate with?" -ForegroundColor White
Write-Host ""
Write-Host "  1) Claude Code"
Write-Host "  2) Cursor"
Write-Host "  3) Windsurf"
Write-Host "  4) VS Code (Copilot Chat)"
Write-Host "  5) Cline"
Write-Host "  6) Zed"
Write-Host "  7) All of the above"
Write-Host "  8) Just install the binary (I'll configure it myself)"
Write-Host ""
$choice = Read-Host "Choose [1-8, comma-separated] (default: 1)"
if ([string]::IsNullOrWhiteSpace($choice)) { $choice = "1" }

$editors = $choice -split ',' | ForEach-Object { $_.Trim() }
foreach ($e in $editors) {
    if ($e -notmatch '^[1-8]$') { Write-Fail "Invalid choice: $e" }
}

# ─── MCP config helper ───────────────────────────────────────

function Write-McpConfig {
    param(
        [string]$ConfigFile,
        [string]$TopKey,
        [string]$LurkPath,
        [hashtable]$Extra = @{}
    )

    $lurkEntry = @{
        command = $LurkPath
        args = @("serve")
    }
    foreach ($k in $Extra.Keys) { $lurkEntry[$k] = $Extra[$k] }

    $dir = Split-Path $ConfigFile -Parent
    if (!(Test-Path $dir)) { New-Item -ItemType Directory -Path $dir -Force | Out-Null }

    if (Test-Path $ConfigFile) {
        $config = Get-Content $ConfigFile -Raw | ConvertFrom-Json
        # Ensure top-level key exists
        if (-not $config.$TopKey) {
            $config | Add-Member -NotePropertyName $TopKey -NotePropertyValue ([PSCustomObject]@{}) -Force
        }
        $config.$TopKey | Add-Member -NotePropertyName "lurk" -NotePropertyValue ([PSCustomObject]$lurkEntry) -Force
        $config | ConvertTo-Json -Depth 10 | Set-Content $ConfigFile -Encoding UTF8
        Write-Ok "Updated $ConfigFile"
    } else {
        $config = @{ $TopKey = @{ lurk = $lurkEntry } }
        $config | ConvertTo-Json -Depth 10 | Set-Content $ConfigFile -Encoding UTF8
        Write-Ok "Created $ConfigFile"
    }
}

# ─── Editor installers ───────────────────────────────────────

function Install-Claude {
    $claudeConfig = Join-Path $env:USERPROFILE ".claude.json"
    Write-Info "Configuring Claude Code MCP server"
    Write-McpConfig -ConfigFile $claudeConfig -TopKey "mcpServers" -LurkPath $BinaryPath
    Write-Warn "Restart Claude Code to load the new MCP server."
}

function Install-Cursor {
    $config = Join-Path $env:USERPROFILE ".cursor\mcp.json"
    Write-Info "Configuring Cursor MCP server"
    Write-McpConfig -ConfigFile $config -TopKey "mcpServers" -LurkPath $BinaryPath
    Write-Warn "Restart Cursor to load the new MCP server."
}

function Install-Windsurf {
    $config = Join-Path $env:USERPROFILE ".codeium\windsurf\mcp_config.json"
    Write-Info "Configuring Windsurf MCP server"
    Write-McpConfig -ConfigFile $config -TopKey "mcpServers" -LurkPath $BinaryPath
    Write-Warn "Restart Windsurf to load the new MCP server."
}

function Install-VsCode {
    $config = Join-Path $env:APPDATA "Code\User\mcp.json"
    Write-Info "Configuring VS Code (Copilot Chat) MCP server"
    Write-McpConfig -ConfigFile $config -TopKey "servers" -LurkPath $BinaryPath
    Write-Warn "Restart VS Code to load the new MCP server."
    Write-Warn "Requires VS Code 1.99+ and Copilot Chat in Agent mode."
}

function Install-Cline {
    $config = Join-Path $env:APPDATA "Code\User\globalStorage\saoudrizwan.claude-dev\settings\cline_mcp_settings.json"
    Write-Info "Configuring Cline MCP server"
    Write-McpConfig -ConfigFile $config -TopKey "mcpServers" -LurkPath $BinaryPath -Extra @{ disabled = $false; alwaysAllow = @() }
    Write-Warn "Restart VS Code to load the new MCP server."
}

function Install-Zed {
    $config = Join-Path $env:APPDATA "Zed\settings.json"
    Write-Info "Configuring Zed MCP server"
    Write-McpConfig -ConfigFile $config -TopKey "context_servers" -LurkPath $BinaryPath
    Write-Warn "Restart Zed to load the new MCP server."
}

function Install-All {
    Write-Host ""
    Write-Info "Installing for all supported editors..."
    Write-Host ""
    Install-Claude
    Install-Cursor
    Install-Windsurf
    Install-VsCode
    Install-Cline
    Install-Zed
}

function Install-BinaryOnly {
    Write-Info "Binary-only install"
    Write-Host "Configure your editor manually. See README for config examples."
}

# ─── Execute ──────────────────────────────────────────────────

foreach ($e in $editors) {
    switch ($e) {
        "1" { Install-Claude }
        "2" { Install-Cursor }
        "3" { Install-Windsurf }
        "4" { Install-VsCode }
        "5" { Install-Cline }
        "6" { Install-Zed }
        "7" { Install-All }
        "8" { Install-BinaryOnly }
    }
}

# ─── Cleanup ──────────────────────────────────────────────────

Remove-Item $TmpDir -Recurse -Force -ErrorAction SilentlyContinue

Write-Host ""
Write-Ok "Done! Restart your editor and try pasting a Reddit URL."
