package control

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/jimu-server/common/resp"
	"github.com/jimu-server/db"
	"github.com/jimu-server/middleware/auth"
	"github.com/jimu-server/model"
	"github.com/jimu-server/mq/mq_key"
	"github.com/jimu-server/mq/rabbmq"
	"github.com/jimu-server/util/uuidutils/uuid"
	jsoniter "github.com/json-iterator/go"
	amqp "github.com/rabbitmq/amqp091-go"
	"net/http"
	"time"
)

func Test(c *gin.Context) {
	var err error
	data := &model.AppNotify{
		Id:         uuid.String(),
		PubId:      "system",
		SubId:      "1",
		Title:      "登录通知",
		MsgType:    1,
		Text:       "成功登录",
		CreateTime: time.Now().Format(time.DateTime),
		UpdateTime: time.Now().Format(time.DateTime),
	}
	if err != nil {
		c.JSON(500, resp.Error(err))
		return
	}
	rabbmq.Notify(data)
}

func NotifyPull(c *gin.Context) {
	token := c.MustGet(auth.Key).(*auth.Token)
	var err error
	params := map[string]any{
		"UserId": token.Id,
	}
	var notify []*model.AppNotify
	if notify, err = NotifyMapper.SelectAllNotify(params); err != nil {
		c.JSON(http.StatusInternalServerError, resp.Error(err, resp.Msg("获取通知失败")))
		return
	}
	c.JSON(http.StatusOK, resp.Success(notify, resp.Msg("获取通知成功")))
}

func Notify(c *gin.Context) {
	token := c.MustGet(auth.Key).(*auth.Token)
	var upgrader = websocket.Upgrader{
		Subprotocols: []string{token.Value},
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	var con *websocket.Conn
	var err error
	if con, err = upgrader.Upgrade(c.Writer, c.Request, nil); err != nil {
		logs.Error("upgrade:" + err.Error())
		return
	}
	defer con.Close()
	openNotify(con, token)
}
func openNotify(con *websocket.Conn, token *auth.Token) {
	key := fmt.Sprintf("%s%s", mq_key.Notify, token.Id)
	var err error
	var ch *amqp.Channel
	if ch, err = rabbmq.Client.Channel(); err != nil {
		logs.Error(err.Error())
		return
	}
	defer ch.Close()

	var msgs <-chan amqp.Delivery
	msgs, err = ch.Consume(
		key,   // queue
		"",    // consumer
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		logs.Error(err.Error())
		return
	}
	ctx, cancel := context.WithCancel(context.Background())

	// 每 5 s 检查一次
	go func() {
		ticker := time.NewTicker(time.Second * 5)
		for range ticker.C {
			if err := con.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(time.Second)); err != nil {
				logs.Error("ws,ping error:" + err.Error())
				cancel()
				return
			}
		}
	}()

	// 接收消息 并处理消息
	for {
		select {
		case msg := <-msgs:
			// 数据库入库存储
			var data = &model.AppNotify{}
			if err = jsoniter.Unmarshal(msg.Body, data); err != nil {
				logs.Error(err.Error())
				continue
			}
			// 开启事务
			var tx *sql.Tx
			if tx, err = db.DB.Begin(); err != nil {
				logs.Error(err.Error())
				continue
			}
			if err = NotifyMapper.InsertNotify(data, tx); err != nil {
				logs.Error(err.Error())
				continue
			}
			if err = con.WriteMessage(websocket.TextMessage, msg.Body); err != nil {
				logs.Error("消息推送失败:" + err.Error())
				tx.Rollback()
				return
			}
			msg.Ack(false)
			tx.Commit()
		case <-ctx.Done():
			return
		}
	}
}
