package clock

import (
	"encoding/base64"
	"fmt"
	"github.com/FloatTech/ZeroBot-Plugin/control"
	"github.com/fumiama/cron"
	zero "github.com/wdvxdr1123/ZeroBot"
	"github.com/wdvxdr1123/ZeroBot/message"
	"io/ioutil"
	"time"
)

func init() { // 插件主体
	// 定时任务每天10点执行一次0 0 8-0 * *
	c := cron.New()
	_, err := c.AddFunc("0 * * * *", func() { sendNotice() })
	if err == nil {
		c.Start()
	}

	control.Register("clock", &control.Options{
		DisableOnDefault: true,
		Help: "clock\n" +
			"- 添加定时提醒\n" +
			"- 删除定时提醒",
	}).OnFullMatch("删除定时提醒", zero.OnlyGroup).SetBlock(true).SetPriority(20).
		Handle(func(ctx *zero.Ctx) {
			m, ok := control.Lookup("clock")
			if ok {
				if m.IsEnabledIn(ctx.Event.GroupID) {
					m.Disable(ctx.Event.GroupID)
					ctx.Send(message.Text("删除成功！"))
				} else {
					ctx.Send(message.Text("未启用！"))
				}
			} else {
				ctx.Send(message.Text("找不到该服务！"))
			}
		})

	zero.OnFullMatch("添加定时提醒", zero.OnlyGroup).SetBlock(true).SetPriority(20).
		Handle(func(ctx *zero.Ctx) {
			m, ok := control.Lookup("clock")
			if ok {
				if m.IsEnabledIn(ctx.Event.GroupID) {
					ctx.Send(message.Text("已启用！"))
				} else {
					m.Enable(ctx.Event.GroupID)
					ctx.Send(message.Text("添加成功！"))
				}
			} else {
				ctx.Send(message.Text("找不到该服务！"))
			}
		})
}

// 获取数据拼接消息链并发送
func sendNotice() {
	m, ok := control.Lookup("clock")
	if ok {
		zero.RangeBot(func(id int64, ctx *zero.Ctx) bool {
			for _, g := range ctx.GetGroupList().Array() {
				grp := g.Get("group_id").Int()
				if m.IsEnabledIn(grp) {
					hour := time.Now().Hour()
					hourStr := fmt.Sprintf("%02d", hour)
					f,_ := ioutil.ReadFile("D:/ZeroBot-Plugin/plugin_clock/hibiki/" + hourStr + ".wav")
					bufSotre := base64.StdEncoding.EncodeToString(f)
					ctx.SendGroupMessage(grp,message.Record(fmt.Sprintf("base64://%v",bufSotre)))
				}
			}
			return true
		})
	}
}