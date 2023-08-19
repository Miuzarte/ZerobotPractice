package main

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

func formatAt(atID int, group int) []map[string]any {
	var forwardNode []map[string]any
	var atList []gocqMessage
	tables := func() []int {
		if group == 0 {
			var tables []int
			for i := range msgTableGroup {
				tables = append(tables, i)
			}
			return tables
		}
		return []int{group}
	}()
	for _, i := range tables {
		table := msgTableGroup[i]
		for _, msg := range table {
			for _, at := range msg.atWho {
				if atID == at {
					atList = append(atList, msg)
				}
			}
		}
	}
	sort.Slice(atList, func(i, j int) bool { //根据msg的时间戳由大到小排序
		return atList[i].time > atList[j].time
	})
	atListLen := len(atList)
	if atListLen > 99 { //超过100条合并转发放不下
		atListLen = 99
	}
	forwardNode = appendForwardNode(forwardNode, gocqNodeData{
		name: "NothingBot",
		uin:  selfID,
		content: []string{
			func() string {
				if group != 0 {
					return fmt.Sprintf("群%d中最近%d条at过%d的消息：", group, atListLen, atID)
				} else {
					return fmt.Sprintf("所有群中最近%d条at过%d的消息：", atListLen, atID)
				}
			}(),
		},
	})
	for i := 0; i < atListLen; i++ {
		atMsg := atList[i]
		name := fmt.Sprintf(
			`(%s)%s%s%s`,
			atMsg.timeF,
			cardORnickname(atMsg),
			func() string {
				if group != 0 {
					return ""
				} else {
					return fmt.Sprintf("  (%d)", atMsg.group_id)
				}
			}(),
			func() string {
				if atMsg.recalled {
					if atMsg.operator_id == atMsg.user_id {
						return "(已撤回)"
					} else {
						return "(已被他人撤回)"
					}
				} else {
					return ""
				}
			}())
		content := strings.ReplaceAll(atMsg.message, "CQ:at,", "CQ:at,​") //插入零宽空格阻止CQ码解析
		forwardNode = appendForwardNode(forwardNode, gocqNodeData{
			name:    name,
			uin:     atMsg.user_id,
			content: []string{content},
		})
	}
	return forwardNode
}

func checkAt(ctx gocqMessage) {
	reg := regexp.MustCompile(`^谁@?[aA艾]?[tT特]?(我|(\s?\[CQ:at,qq=)?([0-9]{1,11})?(\]\s?))$`).FindAllStringSubmatch(ctx.message, -1)
	if len(reg) > 0 {
		var atID int
		if reg[0][1] == "我" {
			atID = ctx.user_id
		} else {
			var err error
			atID, err = strconv.Atoi(reg[0][3])
			if err != nil {
				return
			}
		}
		sendForwardMsgCTX(ctx, func() []map[string]any {
			switch ctx.message_type {
			case "group":
				return formatAt(atID, ctx.group_id)
			case "private":
				return formatAt(atID, 0)
			}
			return []map[string]any{}
		}())
	}
}
