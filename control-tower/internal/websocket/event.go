package websocket

type Message struct {
	To          string   `json:"to"`
	Body        string   `json:"body"`
	Attachments []string `json:"attachments"`
}

// type Event struct {
// 	Type         string  `json:"type"`
// 	Msg          Message `json:"message"`
// 	From         string  `json:"from"`
// 	To           string  `json:"to"`
// 	Url          string  `json:"url"`
// 	ConnectionId string  `json:"connectionId"`
// }
