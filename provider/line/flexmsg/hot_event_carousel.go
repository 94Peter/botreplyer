package flexmsg

import (
	"github.com/line/line-bot-sdk-go/v7/linebot"
)

type hotEventBubble struct {
	Name        string
	Description string
	ImgUrl      string
	Url         string
}

type HotEventCarousel struct {
	contents []any
}

func (h *HotEventCarousel) AddContent(name, desc, imgUrl, url string) {
	h.contents = append(h.contents, &hotEventBubble{
		Name:        name,
		Description: desc,
		ImgUrl:      imgUrl,
		Url:         url,
	})
}

func (h *HotEventCarousel) IsEmpty() bool {
	return len(h.contents) == 0
}

func (data *HotEventCarousel) Build() (linebot.SendingMessage, error) {
	return getCarouselSendingMessage("hotEventBubble", data.contents)
}
