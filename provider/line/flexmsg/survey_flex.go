package flexmsg

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/line/line-bot-sdk-go/v7/linebot"
)

type SurveyFlex struct {
	SurveyURL string
}

func (data *SurveyFlex) Build() (linebot.SendingMessage, error) {
	msgTmpl, err := getFlexTemplate("survey")
	if err != nil {
		return nil, fmt.Errorf("failed to get survey template: %w", err)
	}

	var buf bytes.Buffer
	if err := msgTmpl.Template.Execute(&buf, data); err != nil {
		return nil, err
	}

	var bc linebot.BubbleContainer
	if err := json.Unmarshal(buf.Bytes(), &bc); err != nil {
		return nil, err
	}

	return linebot.NewFlexMessage(msgTmpl.AltText, &bc), nil
}
