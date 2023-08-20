package main

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var (
	autoRecallSwitch = true
	callSUMsgList    []gocqMessage
	callSUMsgUnread  int
)

// 触发消息撤回时自动撤回回复消息(working...)
func checkBotAutoRecall(msgORrecall any) {
	var reg [][]string
	if ctx, ok := msgORrecall.(gocqMessage); ok {
		reg = regexp.MustCompile("(开启|启用|关闭|禁用)自动撤回").FindAllStringSubmatch(ctx.message, -1)
		if len(reg) > 0 && isPrivate(ctx) && matchSU(ctx) {
			switch reg[0][1] {
			case "开启", "启用":
				autoRecallSwitch = true
				sendMsgCTX(ctx, "自动撤回已启用")
			case "关闭", "禁用":
				autoRecallSwitch = false
				sendMsgCTX(ctx, "自动撤回已禁用")
			}
			return
		}
	}
	if !autoRecallSwitch {
		return
	}
	if poke, ok := msgORrecall.(gocqFriendRecall); ok {
		_ = poke
	} else if poke, ok := msgORrecall.(gocqGroupRecall); ok {
		_ = poke
	}
}

// 戳一戳回复
func checkBotInternalPoke(poke gocqPoke) {
	if poke.group_id != 0 {
		sendGroupMsg(poke.group_id, "[NothingBot] 在一条消息内只at我两次可以获取帮助信息~")
	}
}

// Bot内置逻辑
func checkBotInternal(ctx gocqMessage) {
	var reg [][]string
	//连续at两次获取帮助, 带文字则视为喊话超级用户
	reg = regexp.MustCompile(fmt.Sprintf(`^\[CQ:at\,qq=%d]\s*\[CQ:at\,qq=%d]\s*(.*)$`, selfID, selfID)).
		FindAllStringSubmatch(ctx.message, -1)
	if len(reg) > 0 {
		call := reg[0][1]
		if len(call) > 0 { //记录喊话
			callSUMsgList = append(callSUMsgList, ctx)
			callSUMsgUnread++
			sendMsgReplyCTX(ctx, "[NothingBot] 已记录此条喊话并通知超级用户")
			log2SU.Info("收到一条新的喊话，未读", callSUMsgUnread)
		} else { //输出帮助
			sendForwardMsgCTX(ctx, appendForwardNode([]map[string]any{}, gocqNodeData{content: []string{
				"github.com/Miuzarte/NothingBot",
				"符号说明：\n{}: 必要参数\n[]: 可选参数\n|: 或",
				"帮助信息：\n“@Bot@Bot”\n（“@Bot @Bot ”）\n输出帮助信息",
				"喊话超级用户：\n“@Bot@Bot{message}”\n（“@Bot @Bot 出bug辣”）\n转发喊话消息至Bot管理员",
				"at消息记录：\n“谁{@|at|AT|艾特}{我|@群友|QQ号}”\n（“谁at我”）\n输出群内at过某人的消息集合",
				"撤回消息记录：\n“让我康康[@群友|QQ号]撤回了什么”\n（“让我康康撤回了什么”）\n输出群内撤回的消息集合（可过滤）",
				"哔哩哔哩链接解析：\n短链、动态、视频、文章、空间、直播间\n（“space.bilibili.com/59442895”）\n解析内容信息",
				"哔哩哔哩快捷搜索：\n“B搜{视频|番剧|影视|直播间|直播|主播|专栏|用户}{keywords}”\n（“B搜用户謬紗特”）\n取决于类别，B站只会返回最多20或30条结果",
				fmt.Sprintf("注入消息：\n“@Botrun{text}”\n（“@Bot run[CQ:at,​qq=%d]”）\n输出相应消息，支持CQ码", selfID),
				"回复：\n“@Bot回复我[text]”\n（“@Bot 回复我114514”）\n回复对应消息，支持CQ码",
				"运行状态：\n“[@Bot]{检查身体|运行状态}”\n（“检查身体”）\n输出NothingBot运行信息",
			}}))
		}
	}
	//发送/清空收件箱
	reg = regexp.MustCompile(`^(清空)?(喊话列表|收件箱)$`).
		FindAllStringSubmatch(ctx.message, -1)
	if len(reg) > 0 && isPrivate(ctx) && matchSU(ctx) {
		callSUMsgUnread = 0  //清零未读
		if reg[0][1] == "" { //发送
			sort.Slice(callSUMsgList, func(i, j int) bool { //根据msg的时间戳由大到小排序
				return callSUMsgList[i].time > callSUMsgList[j].time
			})
			callSUMsgLen := len(callSUMsgList)
			if callSUMsgLen == 0 {
				sendMsgCTX(ctx, "[NothingBot] [Info] 收件箱为空！")
				return
			}
			if callSUMsgLen > 100 { //超过100条合并转发放不下
				callSUMsgLen = 100
			}
			var forwardNode []map[string]any
			for i := 0; i < callSUMsgLen; i++ {
				callSUMsg := callSUMsgList[i]
				name := fmt.Sprintf(
					`(%s)%s  (%d)`,
					callSUMsg.timeF,
					cardORnickname(callSUMsg),
					callSUMsg.group_id)
				content := strings.ReplaceAll(callSUMsg.message, "CQ:at,", "CQ:at,​") //插入零宽空格阻止CQ码解析
				forwardNode = appendForwardNode(forwardNode, gocqNodeData{
					name:    name,
					uin:     callSUMsg.user_id,
					content: []string{content},
				})
			}
			sendForwardMsgCTX(ctx, forwardNode)
		} else if reg[0][1] == "清空" { //清空
			callSUMsgList = []gocqMessage{}
			sendMsgCTX(ctx, "[NothingBot] [Info] 已清空")
		}
	}
	//注入消息
	reg = regexp.MustCompile(fmt.Sprintf(`^\[CQ:at\,qq=%d]\s*run(.*)$`, selfID)).
		FindAllStringSubmatch(ctx.message, -1)
	if len(reg) > 0 {
		sendMsgCTX(ctx, unescape.Replace(reg[0][1]))
	}
	//回复我
	reg = regexp.MustCompile(fmt.Sprintf(`^\[CQ:at\,qq=%d]\s*回复我(.*)?$`, selfID)).
		FindAllStringSubmatch(ctx.message, -1)
	if len(reg) > 0 {
		if reg[0][1] == "" {
			sendMsgReplyCTX(ctx, "回复你")
		} else {
			sendMsgReplyCTX(ctx, unescape.Replace(reg[0][1]))
		}
	}
}
