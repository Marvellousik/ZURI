package github

import (
	"strings"
)

// ReplyAction represents the parsed intent from a user's reply.
type ReplyAction string

const (
	ActionConfirm ReplyAction = "confirm"
	ActionReject  ReplyAction = "reject"
	ActionEdit    ReplyAction = "edit"
	ActionNone    ReplyAction = "none"
)

// ParsedReply holds the result of parsing a user's comment.
type ParsedReply struct {
	Action  ReplyAction
	Content string // The remaining content (e.g., the edited decision)
}

// ParseReply analyzes a comment body to extract Zuri resolution commands.
func ParseReply(body string) ParsedReply {
	body = strings.TrimSpace(body)
	lowerBody := strings.ToLower(body)

	if strings.HasPrefix(lowerBody, "confirm") {
		return ParsedReply{Action: ActionConfirm}
	}

	if strings.HasPrefix(lowerBody, "reject") {
		return ParsedReply{Action: ActionReject}
	}

	if strings.HasPrefix(lowerBody, "edit ") {
		// Extract the rest of the string after "edit " keeping original casing
		content := strings.TrimSpace(body[5:])
		return ParsedReply{Action: ActionEdit, Content: content}
	}

	return ParsedReply{Action: ActionNone}
}
