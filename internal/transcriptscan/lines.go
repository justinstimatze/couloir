package transcriptscan

import "encoding/json"

// transcriptLine is the common envelope every transcript JSONL line
// shares, confirmed by direct inspection of a real session transcript.
// Fields not relevant to this Lens (uuid, parentUuid, cwd, gitBranch,
// version, and every plugin-injected type this host's hook stack adds)
// are deliberately not modeled — an unrecognized type is ignored, not
// an error.
type transcriptLine struct {
	Type            string           `json:"type"`
	Subtype         string           `json:"subtype"`
	PromptSource    string           `json:"promptSource"`
	Message         json.RawMessage  `json:"message"`
	CompactMetadata *compactMetadata `json:"compactMetadata"`
}

type compactMetadata struct {
	Trigger string `json:"trigger"`
}

// assistantMessage is the subset of an assistant line's message object
// this Lens reads.
type assistantMessage struct {
	Model string `json:"model"`
	Usage struct {
		InputTokens              int `json:"input_tokens"`
		CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
		CacheReadInputTokens     int `json:"cache_read_input_tokens"`
		OutputTokens             int `json:"output_tokens"`
	} `json:"usage"`
}

// userMessage is the subset of a user line's message object this Lens
// reads. Content is raw because it's a string for a real typed prompt
// but an array of content blocks (tool_result, etc.) otherwise — the
// caller distinguishes by attempting a string unmarshal.
type userMessage struct {
	Content json.RawMessage `json:"content"`
}
