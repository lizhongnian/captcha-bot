package telegram

import (
	"fmt"
	"github.com/assimon/captcha-bot/util/captcha"
	"github.com/assimon/captcha-bot/util/config"
	"github.com/assimon/captcha-bot/util/log"
	tb "gopkg.in/tucnak/telebot.v2"
	"os"
	"strconv"
	"time"
)

// JoinCaptcha 验证结构体
type JoinCaptcha struct {
	UserID           int         // tg用户唯一标识
	CaptchaImgID     string      // 验证码ID
	UserInfo         *tb.User    // tg用户信息
	GroupChat        *tb.Chat    // 群组会话
	PrivateChat      *tb.Chat    // 私聊会话
	BotPromptMessage *tb.Message // 机器人提示消息
	CaptchaMessage   *tb.Message // 验证码消息
}

var (
	// PendingCaptchaList 等待验证的集合
	PendingCaptchaList = make(map[int]*JoinCaptcha)
	// adminRole 管理组权限
	adminRole = map[tb.MemberStatus]int{
		tb.Administrator: 1,
		tb.Creator:       1,
	}
)

// ping 存活检测
func ping(m *tb.Message) {
	_, err := bots.Send(m.Chat, "Hi👋")
	if err != nil {
		log.Sugar.Error(err)
	}
}

// onText 文本消息
func onText(m *tb.Message) {
	// 如果是私聊消息 开始判断验证
	if m.Private() {
		if user, notVerify := PendingCaptchaList[m.Sender.ID]; notVerify {
			if captcha.VerifyCaptcha(user.CaptchaImgID, m.Text) {
				memberPass(m.Sender.ID)
				_, err := bots.Send(m.Sender, config.TgConf.CaptchaSuccessMsgTpl)
				if err != nil {
					log.Sugar.Error(err)
				}
			} else {
				_, err := bots.Send(m.Sender, "验证码错误，请重新尝试")
				if err != nil {
					log.Sugar.Error(err)
				}
			}
		}
	}
}

// StartChat 开始会话
func StartChat(m *tb.Message) {
	if m.Payload != "" {
		sendCaptcha(m)
	}
}

// userJoinGroup 用户加群事件
func userJoinGroup(m *tb.Message) {
	userID := m.UserJoined.ID
	if userID != m.Sender.ID {
		return
	}
	log.Sugar.Infof("新用户加入群会话:%s, UserID:%d, UserName:%s, UserInfo:%v",
		m.Chat.Title,
		userID,
		m.UserJoined.Username,
		m.UserJoined,
	)
	// 获得这个用户群群内所属权限
	member, err := bots.ChatMemberOf(m.Chat, m.UserJoined)
	if err != nil {
		log.Sugar.Error(err)
		return
	}
	// 已经禁言 且 不在待验证群组内
	if _, verify := PendingCaptchaList[userID]; !verify && member.Role == tb.Restricted {
		log.Sugar.Infof("用户:%v，已被封禁，无需处理",
			m.UserJoined,
		)
		return
	}
	// 先封禁用户，使其不能发言，需要私聊机器人后解除禁用
	banUserMsg := tb.ChatMember{
		User:            m.UserJoined,
		RestrictedUntil: tb.Forever(),
		Rights:          tb.Rights{CanSendMessages: false},
	}
	err = bots.Restrict(m.Chat, &banUserMsg)
	if err != nil {
		log.Sugar.Errorf("禁言用户失败，err:%v", err)
		//return
	}
	// 向群发送需要用户解禁的消息，并且at用户
	captchaBtn := tb.InlineButton{
		Unique: fmt.Sprintf("captcha-%d", userID),
		Text:   "🤖自助解禁",
		URL:    fmt.Sprintf("https://t.me/%s?start=%d", bots.Me.Username, userID),
	}
	checkOKBtn := tb.InlineButton{
		Unique: fmt.Sprintf("checkOKBtn-%d", m.Sender.ID),
		Text:   "手动通过[管理员]",
		Data:   fmt.Sprintf("%d", userID),
	}
	checkNotBtn := tb.InlineButton{
		Unique: fmt.Sprintf("checkNotBtn-%d", m.Sender.ID),
		Text:   "手动拒绝[管理员]",
		Data:   fmt.Sprintf("%d", userID),
	}
	// 按钮布局
	inlineKeys := [][]tb.InlineButton{
		{
			captchaBtn,
		},
		{
			checkOKBtn,
			checkNotBtn,
		},
	}
	bots.Handle(&checkOKBtn, manuallyPass)
	bots.Handle(&checkNotBtn, manualRejection)
	afterPromptTime := config.TgConf.PromptMsgAfterDelTime
	promptMsg := fmt.Sprintf(config.TgConf.PromptMsgTpl,
		m.UserJoined.Username,
		m.Chat.Title,
		afterPromptTime,
	)
	respMsg, err := bots.Send(
		m.Chat,
		promptMsg,
		&tb.ReplyMarkup{InlineKeyboard: inlineKeys},
		tb.ModeMarkdown,
	)
	if err != nil {
		log.Sugar.Error(err)
		return
	}
	pending := &JoinCaptcha{
		UserID:           userID,
		UserInfo:         m.UserJoined,
		GroupChat:        m.Chat,
		BotPromptMessage: respMsg,
	}
	PendingCaptchaList[userID] = pending
	// 删除加群消息
	err = bots.Delete(m)
	if err != nil {
		log.Sugar.Error(err)
	}
	// 友好提示，如果还未通过验证就删除这条消息，不能让验证消息刷群
	afterPromptFunc := func() {
		if _, isCaptcha := PendingCaptchaList[userID]; isCaptcha && pending.BotPromptMessage != nil {
			err := bots.Delete(pending.BotPromptMessage)
			if err != nil {
				log.Sugar.Error(err)
			}
			pending.BotPromptMessage = nil
		}
	}
	time.AfterFunc(time.Second*time.Duration(afterPromptTime), afterPromptFunc)
	// 超时删除
	afterCaptchaFunc := func() {
		if _, isCaptcha := PendingCaptchaList[userID]; isCaptcha && pending.BotPromptMessage != nil {
			deleteRuntime(pending)
		}
	}
	afterCaptchaTime := config.TgConf.CaptchaTimeOut
	time.AfterFunc(
		time.Second*time.Duration(afterCaptchaTime),
		afterCaptchaFunc,
	)
}

// manuallyPass 管理员手动通过
func manuallyPass(c *tb.Callback) {
	user, err := bots.ChatMemberOf(c.Message.Chat, c.Sender)
	if err != nil {
		log.Sugar.Error(err)
		return
	}
	// 普通用户 无权限
	if admin := chatIsAdmin(user, c); !admin {
		return
	}
	userID, err := strconv.Atoi(c.Data)
	if err != nil {
		log.Sugar.Error(err)
		return
	}
	memberPass(userID)
}

// manualRejection 管理员手动拒绝
func manualRejection(c *tb.Callback) {
	user, err := bots.ChatMemberOf(c.Message.Chat, c.Sender)
	if err != nil {
		log.Sugar.Error(err)
		return
	}
	// 普通用户 无权限
	if admin := chatIsAdmin(user, c); !admin {
		return
	}
	userID, err := strconv.Atoi(c.Data)
	if err != nil {
		log.Sugar.Error(err)
		return
	}
	memberFail(userID)
}

// sendCaptcha 发送验证信息
func sendCaptcha(m *tb.Message) {
	// 不是私聊  直接return
	if !m.Private() {
		return
	}
	// 是否在待验证队列
	user, ok := PendingCaptchaList[m.Sender.ID]
	if !ok {
		return
	}
	// 获得一个验证码
	captchaCode, imgUrl, err := captcha.GetCaptcha()
	if err != nil {
		log.Sugar.Error(err)
		return
	}
	afterTime := config.TgConf.CaptchaMsgAfterDelTime
	captchaTpl := fmt.Sprintf(config.TgConf.CaptchaMsgTpl,
		user.GroupChat.Title,
		afterTime,
	)
	captchaMsg := &tb.Photo{
		File:    tb.FromDisk(imgUrl),
		Caption: captchaTpl,
	}
	refreshBtn := tb.InlineButton{
		Unique: fmt.Sprintf("refresh-%d", user.UserID),
		Text:   "刷新",
		Data:   fmt.Sprintf("%d", user.UserID),
	}
	inlineKeys := [][]tb.InlineButton{{
		refreshBtn,
	}}
	bots.Handle(&refreshBtn, refreshCaptchaCode)
	// 发送验证消息
	captchaResp, err := bots.Send(
		m.Sender,
		captchaMsg,
		&tb.ReplyMarkup{InlineKeyboard: inlineKeys},
		tb.ModeMarkdown,
	)
	// 赋值用户信息
	user.CaptchaMessage = captchaResp
	user.CaptchaImgID = captchaCode
	user.PrivateChat = m.Chat
	// 图片回收
	err = os.Remove(imgUrl)
	if err != nil {
		log.Sugar.Error(err)
	}
	afterFunc := func() {
		// 如果还未通过验证就删除这条消息
		if _, isVerify := PendingCaptchaList[user.UserID]; isVerify && user.CaptchaMessage != nil {
			err = bots.Delete(user.CaptchaMessage)
			if err != nil {
				log.Sugar.Error(err)
			}
			user.CaptchaMessage = nil
		}
	}
	time.AfterFunc(time.Second*time.Duration(afterTime), afterFunc)
}

// refreshCaptchaCode 刷新验证码
func refreshCaptchaCode(c *tb.Callback) {
	user, ok := PendingCaptchaList[c.Sender.ID]
	if !ok {
		return
	}
	// 获得一个新验证码
	captchaCode, imgUrl, err := captcha.GetCaptcha()
	if err != nil {
		log.Sugar.Error(err)
		return
	}
	afterTime := config.TgConf.CaptchaMsgAfterDelTime
	editMessage := &tb.Photo{
		File: tb.FromDisk(imgUrl),
		Caption: fmt.Sprintf(config.TgConf.CaptchaMsgTpl,
			user.GroupChat.Title,
			afterTime,
		),
	}
	editResp, err := bots.Edit(c.Message, editMessage, &tb.ReplyMarkup{InlineKeyboard: c.Message.ReplyMarkup.InlineKeyboard}, tb.ModeMarkdown)
	if err != nil {
		log.Sugar.Error(err)
		return
	}
	user.CaptchaMessage = editResp
	user.CaptchaImgID = captchaCode
	// 图片回收
	err = os.Remove(imgUrl)
	if err != nil {
		log.Sugar.Error(err)
	}
}

// memberPass 用户通过
func memberPass(userID int) {
	member, ok := PendingCaptchaList[userID]
	if !ok {
		return
	}
	// 解除禁言
	unbanUser := tb.ChatMember{
		User:   member.UserInfo,
		Rights: tb.NoRestrictions(),
	}
	err := bots.Restrict(member.GroupChat, &unbanUser)
	if err != nil {
		log.Sugar.Errorf("解禁用户失败，err:%v", err)
	}
	deleteRuntime(member)
}

// memberFail 用户未通过
func memberFail(userID int) {
	member, ok := PendingCaptchaList[userID]
	if !ok {
		return
	}
	deleteRuntime(member)
}

// deleteRuntime 删除运行时资源
func deleteRuntime(c *JoinCaptcha) {
	if c.BotPromptMessage != nil {
		// 删除引导消息
		err := bots.Delete(c.BotPromptMessage)
		if err != nil {
			log.Sugar.Errorf("删除引导消息失败，err:%v", err)
		}
	}
	if c.CaptchaMessage != nil {
		// 删除验证码消息
		err := bots.Delete(c.CaptchaMessage)
		if err != nil {
			log.Sugar.Error(err)
		}
	}
	// 从集合中删除
	delete(PendingCaptchaList, c.UserID)
}

// chatIsAdmin 会话事件是否为管理员
func chatIsAdmin(user *tb.ChatMember, c *tb.Callback) bool {
	// 普通用户 无权限
	if _, isAdmin := adminRole[user.Role]; !isAdmin {
		err := bots.Respond(c, &tb.CallbackResponse{
			CallbackID: c.MessageID,
			Text:       "无权限",
			ShowAlert:  true,
		})
		if err != nil {
			log.Sugar.Error(err)
		}
		return false
	}
	return true
}
