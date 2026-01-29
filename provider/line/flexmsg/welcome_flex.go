package flexmsg

import (
	"github.com/line/line-bot-sdk-go/v7/linebot"
)

type WelcomeFlex struct{}

func (data *WelcomeFlex) Build() (linebot.SendingMessage, error) {
	return getBubbleSendingMessage("welcome", data)
}
