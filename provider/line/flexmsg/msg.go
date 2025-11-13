package flexmsg

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
