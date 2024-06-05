package mapper

import (
	"database/sql"
	"github.com/jimu-server/model"
)

type NotifyMapper struct {

	// 查询用户所有通知
	SelectAllNotify func(any) ([]*model.AppNotify, error)

	// 用户消息入库
	InsertNotify func(any, *sql.Tx) error

	// 清空用户当前消息通知
	ClearNotify func(any) error

	// 更新消息为已读状态
	ReadNotify func(any) error

	// 删除用户消息
	DeleteNotify func(any) error
}
