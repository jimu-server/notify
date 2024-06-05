package notify

import (
	"embed"
	"github.com/jimu-server/db"
	"github.com/jimu-server/middleware/auth"
	"github.com/jimu-server/notify/control"
	"github.com/jimu-server/web"
)

//go:embed mapper/file/*.xml
var mapperFile embed.FS

func init() {
	db.GoBatis.LoadByRootPath("mapper", mapperFile)
	db.GoBatis.ScanMappers(control.NotifyMapper)
	api := web.Engine.Group("/api", auth.Authorization("/api/notify", "/notify"))
	api.GET("/notify", control.Notify)          // 用户消息推送
	api.GET("/notify/pull", control.NotifyPull) // 用户消息拉取
	api.POST("/notify/clear", control.Clear)    // 清空当前消息
	api.POST("/notify/read", control.Read)      // 标记消息为已读
	api.POST("/notify/delete", control.Delete)  // 删除消息
}
