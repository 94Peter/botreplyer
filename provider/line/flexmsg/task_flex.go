package flexmsg

import (
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
	contestns []any
}

func (t *TaskCarousel) AddTask(task *taskBubble) {
	t.contestns = append(t.contestns, task)
}

func (t *TaskCarousel) Build() (linebot.SendingMessage, error) {
	return getCarouselSendingMessage("taskBubble", t.contestns)
}
