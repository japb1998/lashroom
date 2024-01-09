package websocket

type Message struct {
	To          string   `json:"to"`
	Body        string   `json:"body"`
	Attachments []string `json:"attachments"`
}
