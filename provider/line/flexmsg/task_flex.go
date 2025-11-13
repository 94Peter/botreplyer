package flexmsg

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/line/line-bot-sdk-go/v7/linebot"
)

type taskBubble struct {
	ActionName string
	ActionUrl  string
	Image      string
	Task       string
	Desc       string
	Details    []*detail
}

func NewTaskBubble() *taskBubble {
	return &taskBubble{}
}

func (t *taskBubble) AddDetail(label, content string) {
	t.Details = append(t.Details, &detail{Label: label, Content: content})
}

type detail struct {
	Label   string
	Content string
}

type TaskCarousel struct {
	contestns []*taskBubble
}

func (t *TaskCarousel) AddTask(task *taskBubble) {
	t.contestns = append(t.contestns, task)
}

func (t *TaskCarousel) Build() (linebot.SendingMessage, error) {
	msgTmpl, err := getFlexTemplate("taskBubble")
	if err != nil {
		return nil, fmt.Errorf("failed to get task bubble template: %w", err)
	}

	var container linebot.CarouselContainer
	container.Type = linebot.FlexContainerTypeCarousel
	for _, c := range t.contestns {
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
