package telegram

import (
	"github.com/assimon/captcha-bot/util/config"
	"github.com/assimon/captcha-bot/util/log"
	tb "gopkg.in/tucnak/telebot.v2"
	"net/http"
	"net/url"
	"time"
)

var bots *tb.Bot

// BotStart 机器人启动
func BotStart() {
	var err error
	botSetting := tb.Settings{
		Token:  config.TgConf.TgToken,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	}
	if config.TgConf.TgProxy != "" {
		uri := url.URL{}
		urlProxy, _ := uri.Parse(config.TgConf.TgProxy)
		botSetting.Client = &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(urlProxy),
			},
		}
	}
	bots, err = tb.NewBot(botSetting)
	if err != nil {
		log.Sugar.Error(err.Error())
		return
	}
	err = bots.SetCommands(Cmds)
	if err != nil {
		log.Sugar.Error(err.Error())
		return
	}
	RegisterHandle()
	bots.Start()
}

// RegisterHandle 注册处理器
func RegisterHandle() {
	bots.Handle(PING_CMD, ping)
	bots.Handle(START_CMD, StartChat)
	bots.Handle(tb.OnText, onText)
	bots.Handle(tb.OnUserJoined, userJoinGroup)
	bots.Handle(tb.OnUserLeft, func(m *tb.Message) {
		err := bots.Delete(m)
		if err != nil {
			log.Sugar.Error(err)
		}
	})
}
