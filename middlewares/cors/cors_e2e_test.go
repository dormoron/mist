package cors

import (
	"fmt"
	"github.com/dormoron/mist"
	"github.com/dormoron/mist/middlewares/accesslog"
	"testing"
)

func TestInitMiddlewareBuilder(t *testing.T) {
	server := mist.InitHTTPServer()
	server.Use(InitMiddlewareBuilder().Build())
	server.Use(accesslog.InitMiddleware().LogFunc(func(log string) {
		fmt.Println(log)
	}).Build())
	server.POST("/users/login", func(ctx *mist.Context) {
		ctx.RespJSONOK("hello")
	})
	server.Start(":8080")
}
