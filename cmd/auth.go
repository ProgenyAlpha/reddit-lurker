package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ProgenyAlpha/reddit-lurker/reddit"
)

const redditAppsURL = "https://www.reddit.com/prefs/apps"

// Auth handles the "lurk auth" command — interactive Reddit app setup.
func Auth(args []string) {
	if len(args) > 0 && args[0] == "--status" {
		authStatus()
		return
	}
	if len(args) > 0 && args[0] == "--clear" {
		authClear()
		return
	}

	fmt.Print(`
┌─────────────────────────────────────────────────────────────┐
│                    Lurk — Reddit Auth Setup                  │
│                                                             │
│  This gives you 6x faster rate limits (60 req/min vs 10).  │
│  Takes about 2 minutes. No personal data needed.            │
└─────────────────────────────────────────────────────────────┘

Opening Reddit's app page in your browser...

`)

	openBrowser(redditAppsURL)

	fmt.Print(`If the browser didn't open, go to:
  https://www.reddit.com/prefs/apps

Log in to Reddit, then scroll down and click "create another app..."

Fill in the form EXACTLY like this:
┌─────────────────────────────────────────────────────────────┐
│  name:          Reddit Lurker MCP                           │
│  type:          (*) script                                  │
│  description:   Reddit reader for Claude Code               │
│  about url:     (leave blank)                               │
│  redirect uri:  http://localhost                            │
└─────────────────────────────────────────────────────────────┘

Click "create app". You'll see two values:

  1. Client ID — the string under "personal use script" (looks like: Ab1Cd2Ef3Gh4Ij)
  2. Client Secret — labeled "secret" (looks like: Kl5Mn6Op7Qr8St9Uv0Wx1Yz)

`)

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Paste your Client ID: ")
	clientID, _ := reader.ReadString('\n')
	clientID = strings.TrimSpace(clientID)

	fmt.Print("Paste your Client Secret: ")
	clientSecret, _ := reader.ReadString('\n')
	clientSecret = strings.TrimSpace(clientSecret)

	if clientID == "" || clientSecret == "" {
		fmt.Fprintln(os.Stderr, "Both Client ID and Client Secret are required.")
		os.Exit(1)
	}

	// Test the credentials
	fmt.Print("\nTesting credentials... ")
	if err := reddit.TestCredentials(clientID, clientSecret); err != nil {
		fmt.Fprintf(os.Stderr, "failed.\n\nError: %s\n\nDouble-check your Client ID and Secret and try again.\n", err)
		os.Exit(1)
	}
	fmt.Println("authenticated.")

	// Save credentials
	creds := reddit.Credentials{
		ClientID:     clientID,
		ClientSecret: clientSecret,
	}

	if err := saveCredentials(creds); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to save credentials: %s\n", err)
		os.Exit(1)
	}

	fmt.Print(`
Done. Lurk will now use OAuth for all requests.

  Rate limit: 60 req/min (was 10)
  Token auto-refreshes every 24h
  Credentials stored in: ~/.config/lurk/credentials.json

If lurk is running as an MCP server, restart it to pick up the new credentials.

Run "lurk auth --status" to check, or "lurk auth --clear" to remove.
`)
}

func authStatus() {
	creds, err := reddit.LoadCredentials()
	if err != nil || creds.ClientID == "" {
		fmt.Println("Not authenticated. Run \"lurk auth\" to set up.")
		return
	}
	// Mask the secret
	masked := creds.ClientSecret
	if len(masked) > 4 {
		masked = masked[:4] + strings.Repeat("*", len(masked)-4)
	}
	fmt.Printf("Authenticated.\n  Client ID: %s\n  Secret:    %s\n", creds.ClientID, masked)

	// Test if credentials still work
	fmt.Print("  Status:    ")
	if err := reddit.TestCredentials(creds.ClientID, creds.ClientSecret); err != nil {
		fmt.Printf("invalid (%s)\n", err)
	} else {
		fmt.Println("valid")
	}
}

func authClear() {
	path := credentialsPath()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Failed to remove credentials: %s\n", err)
		os.Exit(1)
	}
	fmt.Println("Credentials removed. Lurk will use anonymous access (10 req/min).")
}

func credentialsPath() string {
	configDir, _ := os.UserConfigDir()
	return filepath.Join(configDir, "lurk", "credentials.json")
}

func saveCredentials(creds reddit.Credentials) error {
	path := credentialsPath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	}
	if cmd != nil {
		cmd.Start()
	}
}
