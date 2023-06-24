package record

type Event struct {
	Type string         `json:"type"`
	Body map[string]any `json:"body"`
}
