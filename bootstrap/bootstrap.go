package bootstrap

import (
	"github.com/assimon/captcha-bot/telegram"
	_ "github.com/assimon/captcha-bot/util/config"
	_ "github.com/assimon/captcha-bot/util/log"
	"os"
	"os/signal"
	"syscall"
)

// Start 服务启动
func Start() {
	// 机器人启动
	go func() {
		telegram.BotStart()
	}()
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan
}
