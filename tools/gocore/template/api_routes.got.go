package template

import "bytes"

func FromApiRoutes(name, routes string, buffer *bytes.Buffer) {
	buffer.WriteString(`
package routes

import (
	"github.com/gin-gonic/gin"
	"net/http"
	_ "net/http/pprof"

	"`)
	buffer.WriteString(name)
	buffer.WriteString(`/api"
)

func Routes(router *gin.Engine) {

	// 根目录健康检查
	router.Any("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Welcome GoCore Service")
	})

    `)
	buffer.WriteString(routes)
	buffer.WriteString(`
}
`)

}
