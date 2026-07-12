package tui

import (
	"fmt"
	"strings"
)

// SlashCommand represents a parsed slash command from the editor.
type SlashCommand int

const (
	cmdInvalid SlashCommand = iota
	cmdExplain
	cmdSuggest
	cmdOptimize
	cmdHelp
)

// SlashCommandMsg is sent when a slash command is parsed from the editor.
type SlashCommandMsg struct {
	Command        SlashCommand
	Args           string // Everything after the command name
	RawInput       string // The original line that triggered the command
	EditorContent  string // Full editor content when the command was triggered
}

// commandInfo holds metadata about a slash command.
type commandInfo struct {
	name        string
	description string
	usage       string
}

// commandRegistry maps command names to their info.
var commandRegistry = map[string]commandInfo{
	"explain":  {name: "/explain", description: "Explain the current SQL query using AI", usage: "/explain"},
	"suggest":  {name: "/suggest", description: "Suggest a SQL query for a natural language request", usage: "/suggest <description>"},
	"optimize": {name: "/optimize", description: "Optimize the current SQL query using AI", usage: "/optimize"},
	"help":     {name: "/help", description: "Show available slash commands", usage: "/help"},
}

// ParseSlashCommand parses a line of text as a slash command.
// Returns the parsed command and any arguments. Returns cmdInvalid if the
// line does not start with a recognized slash command.
func ParseSlashCommand(line string) SlashCommandMsg {
	line = strings.TrimSpace(line)

	if !strings.HasPrefix(line, "/") {
		return SlashCommandMsg{Command: cmdInvalid}
	}

	// Split into command and arguments
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return SlashCommandMsg{Command: cmdInvalid}
	}

	cmdName := strings.TrimPrefix(parts[0], "/")
	cmdName = strings.ToLower(cmdName)

	_, ok := commandRegistry[cmdName]
	if !ok {
		return SlashCommandMsg{
			Command:  cmdInvalid,
			RawInput: line,
		}
	}

	// Extract arguments (everything after the command name)
	args := ""
	if len(parts) > 1 {
		// Extract the original text after the command
		cmdPrefix := "/" + cmdName
		args = strings.TrimSpace(line[len(cmdPrefix):])
	}

	var cmd SlashCommand
	switch cmdName {
	case "explain":
		cmd = cmdExplain
	case "suggest":
		cmd = cmdSuggest
	case "optimize":
		cmd = cmdOptimize
	case "help":
		cmd = cmdHelp
	}

	return SlashCommandMsg{
		Command:  cmd,
		Args:     args,
		RawInput: line,
	}
}

// IsSlashCommand checks if a line starts with a recognized slash command.
func IsSlashCommand(line string) bool {
	msg := ParseSlashCommand(line)
	return msg.Command != cmdInvalid
}

// CommandHelpText returns the help text listing all available slash commands.
func CommandHelpText() string {
	var b strings.Builder
	b.WriteString("Available slash commands:\n\n")
	for _, info := range commandRegistry {
		b.WriteString(fmt.Sprintf("  %s\n", info.name))
		b.WriteString(fmt.Sprintf("    %s\n", info.description))
		b.WriteString(fmt.Sprintf("    Usage: %s\n\n", info.usage))
	}
	b.WriteString("Type a slash command in the editor and press Ctrl+Enter to execute.")
	return b.String()
}

// CommandName returns the display name for a slash command.
func CommandName(cmd SlashCommand) string {
	switch cmd {
	case cmdExplain:
		return "/explain"
	case cmdSuggest:
		return "/suggest"
	case cmdOptimize:
		return "/optimize"
	case cmdHelp:
		return "/help"
	default:
		return "unknown"
	}
}

// InvalidCommandError returns a formatted error message for an unknown command.
func InvalidCommandError(input string) string {
	return fmt.Sprintf("Unknown command: %s. Type /help for available commands.", input)
}
