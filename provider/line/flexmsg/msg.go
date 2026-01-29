package flexmsg

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/line/line-bot-sdk-go/v7/linebot"
)

type LineFlexMessageBuilder interface {
	Build() string
}

type LinebotFlexMsgConfig struct {
	Logo      string
	SurveyURL string
}

var linebotFlexMsgDefaultConfig = &LinebotFlexMsgConfig{
	Logo:      "https://developers-resource.landpress.line.me/fx/clip/clip13.jpg",
	SurveyURL: "https://tw.yahoo.com/",
}

type LinebotFlexMsgOptions func(*LinebotFlexMsgConfig)

func WithLinebotFlexMsgLogo(logo string) LinebotFlexMsgOptions {
	return func(cfg *LinebotFlexMsgConfig) {
		cfg.Logo = logo
	}
}

func Default() *LinebotFlexMsgConfig {
	return linebotFlexMsgDefaultConfig
}

func getBubbleSendingMessage(templateName string, data any) (linebot.SendingMessage, error) {
	msgTmpl, err := getFlexTemplate(templateName)
	if err != nil {
		return nil, fmt.Errorf("failed to get template %s: %w", templateName, err)
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

func getCarouselSendingMessage(templateName string, contents []any) (linebot.SendingMessage, error) {
	msgTmpl, err := getFlexTemplate(templateName)
	if err != nil {
		return nil, fmt.Errorf("failed to get template %s: %w", templateName, err)
	}
	var container linebot.CarouselContainer
	container.Type = linebot.FlexContainerTypeCarousel
	for _, c := range contents {
		var buf bytes.Buffer
		if err := msgTmpl.Template.Execute(&buf, c); err != nil {
			return nil, fmt.Errorf("failed to execute template: %w", err)
		}

		var bc linebot.BubbleContainer
		if err := json.Unmarshal(buf.Bytes(), &bc); err != nil {
			return nil, fmt.Errorf("failed to unmarshal json: %w", err)
		}
		container.Contents = append(container.Contents, &bc)
	}
	return linebot.NewFlexMessage(msgTmpl.AltText, &container), nil
}
