package flexmsg

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/line/line-bot-sdk-go/v7/linebot"
)

type hotEventBubble struct {
	Name        string
	Description string
	ImgUrl      string
	Url         string
}

type HotEventCarousel struct {
	contents []*hotEventBubble
}

func (h *HotEventCarousel) AddContent(name, desc, ImgUrl, url string) {
	h.contents = append(h.contents, &hotEventBubble{
		Name:        name,
		Description: desc,
		ImgUrl:      ImgUrl,
		Url:         url,
	})
}

func (h *HotEventCarousel) IsEmpty() bool {
	return len(h.contents) == 0
}

func (data *HotEventCarousel) Build() (linebot.SendingMessage, error) {
	msgTpl, err := getFlexTemplate("hotEventBubble")
	if err != nil {
		return nil, fmt.Errorf("failed to get hot event bubble template: %w", err)
	}

	var container linebot.CarouselContainer
	container.Type = linebot.FlexContainerTypeCarousel
	for _, c := range data.contents {
		var buf bytes.Buffer
		if err := msgTpl.Template.Execute(&buf, c); err != nil {
			return nil, fmt.Errorf("failed to execute template: %w", err)
		}

		var bc linebot.BubbleContainer
		if err := json.Unmarshal(buf.Bytes(), &bc); err != nil {
			return nil, fmt.Errorf("failed to unmarshal json: %w", err)
		}
		container.Contents = append(container.Contents, &bc)
	}
	return linebot.NewFlexMessage(msgTpl.AltText, &container), nil
}
