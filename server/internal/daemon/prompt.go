package daemon

import (
	"fmt"
	"strings"
)

// BuildPrompt constructs the task prompt for an agent CLI.
// Keep this minimal — detailed instructions live in CLAUDE.md / AGENTS.md
// injected by execenv.InjectRuntimeConfig.
func BuildPrompt(task Task) string {
	if task.ChatSessionID != "" {
		return buildChatPrompt(task)
	}
	var b strings.Builder
	b.WriteString("You are running as a local coding agent for a Multica workspace.\n\n")
	fmt.Fprintf(&b, "Your assigned issue ID is: %s\n\n", task.IssueID)
	if task.IssueTitle != "" {
		fmt.Fprintf(&b, "Issue title: %s\n\n", task.IssueTitle)
	}
	if strings.TrimSpace(task.IssueDescription) != "" {
		b.WriteString("Issue description:\n")
		b.WriteString(task.IssueDescription)
		b.WriteString("\n\n")
	}
	b.WriteString("Use the embedded issue context first. Only fall back to `multica issue get --output json` if you need extra platform metadata.\n")
	return b.String()
}

// buildChatPrompt constructs a prompt for interactive chat tasks.
func buildChatPrompt(task Task) string {
	var b strings.Builder
	b.WriteString("You are running as a chat assistant for a Multica workspace.\n")
	b.WriteString("A user is chatting with you directly. Respond to their message.\n\n")
	fmt.Fprintf(&b, "User message:\n%s\n", task.ChatMessage)
	return b.String()
}
