package flexmsg

import (
	"github.com/line/line-bot-sdk-go/v7/linebot"
)

type EventMemory struct {
	Photos       [3]string
	UserName     string
	TeamName     string
	EventName    string
	EventSession string
	MemoryURL    string
	Logo         string
}

func (data *EventMemory) Build() (linebot.SendingMessage, error) {
	return getBubbleSendingMessage("eventMemory", data)
}
