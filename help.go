package main

import (
	"fmt"
	"os"
)

func printHelp(cmd string, args []string) {
	// `oura help [command]` routes here with cmd=="".
	if cmd == "" {
		if len(args) == 0 {
			printUsage()
			return
		}
		cmd = args[0]
		args = args[1:]
	}

	switch cmd {
	case "", "--help", "-h":
		printUsage()
	case "auth":
		fmt.Println("Usage: oura auth")
		fmt.Println("Authenticate with Oura (OAuth2).")
	case "completion", "completions":
		printCompletionUsage()
	case "personal-info", "personal_info", "personal":
		printPersonalInfoUsage()
	case "tag":
		printTagUsage()
	case "enhanced-tag", "enhanced_tag":
		printEnhancedTagUsage()
	case "session":
		printSessionUsage()
	case "webhook":
		printWebhookUsage()
	default:
		// For legacy date-based commands, keep help short.
		fmt.Fprintf(os.Stderr, "Unknown command for help: %s\n\n", cmd)
		printUsage()
	}
}
