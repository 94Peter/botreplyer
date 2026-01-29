package flexmsg

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/line/line-bot-sdk-go/v7/linebot"
)

type question struct {
	Question string
	Options  []*QuestOption
	Index    int
}

type QuestOption struct {
	Label string
	Text  string
}

type QuestionConfirm struct {
	question *question
}

func (data *QuestionConfirm) SetQuestion(index int, q string, options ...QuestOption) *QuestionConfirm {
	data.question = &question{
		Question: q,
		Index:    index,
	}
	for _, o := range options {
		data.question.Options = append(data.question.Options, &QuestOption{
			Label: o.Label,
			Text:  o.Text,
		})
	}
	return data
}

func (data *QuestionConfirm) Build() (linebot.SendingMessage, error) {
	msgTmpl, err := getFlexTemplate("question")
	if err != nil {
		return nil, fmt.Errorf("failed to get question template: %w", err)
	}

	var buf bytes.Buffer
	if err := msgTmpl.Template.Execute(&buf, data.question); err != nil {
		return nil, err
	}

	var bc linebot.BubbleContainer
	if err := json.Unmarshal(buf.Bytes(), &bc); err != nil {
		return nil, err
	}

	return linebot.NewFlexMessage(msgTmpl.AltText, &bc), nil
}
