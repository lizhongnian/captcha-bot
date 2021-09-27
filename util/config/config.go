package config

import "github.com/spf13/viper"

// TgConfig 系统配置
type TgConfig struct {
	TgToken                string `json:"tg_token"`
	TgProxy                string `json:"tg_proxy"`
	PromptMsgAfterDelTime  int    `json:"prompt_msg_after_del_time"`
	CaptchaMsgAfterDelTime int    `json:"captcha_msg_after_del_time"`
	CaptchaTimeOut         int    `json:"captcha_time_out"`
	PromptMsgTpl           string `json:"prompt_msg_tpl"`
	CaptchaMsgTpl          string `json:"captcha_msg_tpl"`
	CaptchaSuccessMsgTpl   string `json:"captcha_success_msg_tpl"`
	RuntimeRootPath        string `json:"runtime_root_path"`
	LogSavePath            string `json:"log_save_path"`
	VerifyImgPath          string `json:"verify_img_path"`
	LogMaxSize             int    `json:"log_max_size"`
	LogMaxAge              int    `json:"log_max_age"`
	LogMaxBackups          int    `json:"log_max_backups"`
}

var TgConf TgConfig

// init 配置加载
func init() {
	viper.AddConfigPath("./")
	viper.SetConfigFile(".env")
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}
	err = viper.Unmarshal(&TgConf)
	if err != nil {
		panic(err)
	}
}
