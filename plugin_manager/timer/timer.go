// Package timer 群管定时器
package timer

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fumiama/cron"
	"github.com/sirupsen/logrus"
	zero "github.com/wdvxdr1123/ZeroBot"
	"github.com/wdvxdr1123/ZeroBot/message"

	"github.com/FloatTech/ZeroBot-Plugin/utils/file"
	"github.com/FloatTech/ZeroBot-Plugin/utils/sql"
)

type Clock struct {
	db       *sql.Sqlite
	timers   *(map[uint32]*Timer)
	timersmu sync.RWMutex
	// cron 定时器
	cron *cron.Cron
	// entries key <-> cron
	entries map[uint32]cron.EntryID
	entmu   sync.Mutex
}

var (
	// @全体成员
	atall = message.MessageSegment{
		Type: "at",
		Data: map[string]string{
			"qq": "all",
		},
	}
)

func NewClock(db *sql.Sqlite) (c Clock) {
	c.loadTimers(db)
	c.cron = cron.New()
	c.entries = make(map[uint32]cron.EntryID)
	c.cron.Start()
	return
}

// RegisterTimer 注册计时器
func (c *Clock) RegisterTimer(ts *Timer, save bool) bool {
	var key uint32
	if save {
		key = ts.GetTimerID()
		ts.Id = key
	} else {
		key = ts.Id
	}
	t, ok := c.GetTimer(key)
	if t != ts && ok { // 避免重复注册定时器
		t.SetEn(false)
	}
	logrus.Println("[群管]注册计时器", key)
	if ts.Cron != "" {
		var ctx *zero.Ctx
		if ts.Selfid != 0 {
			ctx = zero.GetBot(ts.Selfid)
		} else {
			zero.RangeBot(func(id int64, c *zero.Ctx) bool {
				ctx = c
				ts.Selfid = id
				return false
			})
		}
		eid, err := c.cron.AddFunc(ts.Cron, func() { ts.sendmsg(ts.GrpId, ctx) })
		if err == nil {
			c.entmu.Lock()
			c.entries[key] = eid
			c.entmu.Unlock()
			if save {
				err = c.AddTimer(ts)
			}
			return err == nil
		}
		ts.Alert = err.Error()
	} else {
		if save {
			_ = c.AddTimer(ts)
		}
		for ts.En() {
			nextdate := ts.nextWakeTime()
			sleepsec := time.Until(nextdate)
			logrus.Printf("[群管]计时器%08x将睡眠%ds", key, sleepsec/time.Second)
			time.Sleep(sleepsec)
			if ts.En() {
				if ts.Month() < 0 || ts.Month() == time.Now().Month() {
					if ts.Day() < 0 || ts.Day() == time.Now().Day() {
						ts.judgeHM()
					} else if ts.Day() == 0 {
						if ts.Week() < 0 || ts.Week() == time.Now().Weekday() {
							ts.judgeHM()
						}
					}
				}
			}
		}
	}
	return false
}

// CancelTimer 取消计时器
func (c *Clock) CancelTimer(key uint32) bool {
	t, ok := c.GetTimer(key)
	if ok {
		if t.Cron != "" {
			c.entmu.Lock()
			e := c.entries[key]
			c.cron.Remove(e)
			delete(c.entries, key)
			c.entmu.Unlock()
		} else {
			t.SetEn(false)
		}
		c.timersmu.Lock()
		delete(*c.timers, key) // 避免重复取消
		e := c.db.Del("timer", "where id = "+strconv.Itoa(int(key)))
		c.timersmu.Unlock()
		return e == nil
	}
	return false
}

// ListTimers 列出本群所有计时器
func (c *Clock) ListTimers(grpID int64) []string {
	// 数组默认长度为map长度,后面append时,不需要重新申请内存和拷贝,效率很高
	if c.timers != nil {
		c.timersmu.RLock()
		keys := make([]string, 0, len(*c.timers))
		for _, v := range *c.timers {
			if v.GrpId == grpID {
				k := v.GetTimerInfo()
				start := strings.Index(k, "]")
				msg := strings.ReplaceAll(k[start+1:]+"\n", "-1", "每")
				msg = strings.ReplaceAll(msg, "月0日0周", "月周天")
				msg = strings.ReplaceAll(msg, "月0日", "月")
				msg = strings.ReplaceAll(msg, "日0周", "日")
				keys = append(keys, msg)
			}
		}
		c.timersmu.RUnlock()
		return keys
	} else {
		return nil
	}
}

func (c *Clock) GetTimer(key uint32) (t *Timer, ok bool) {
	c.timersmu.RLock()
	t, ok = (*c.timers)[key]
	c.timersmu.RUnlock()
	return
}

func (c *Clock) AddTimer(t *Timer) (err error) {
	c.timersmu.Lock()
	(*c.timers)[t.Id] = t
	err = c.db.Insert("timer", t)
	c.timersmu.Unlock()
	return
}

func (c *Clock) loadTimers(db *sql.Sqlite) {
	if file.IsExist(db.DBPath) {
		c.db = db
		err := c.db.Create("timer", &Timer{})
		if err == nil {
			var t Timer
			c.db.FindFor("timer", &t, "", func() error {
				tescape := t
				go c.RegisterTimer(&tescape, false)
				return nil
			})
		}
	}
}
