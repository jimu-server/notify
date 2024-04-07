package mapper

import (
	"database/sql"
	"github.com/jimu-server/model"
)

type NotifyMapper struct {

	// 查询所有通知
	SelectAllNotify func(any) ([]*model.AppNotify, error)

	InsertNotify func(any, *sql.Tx) error
}
