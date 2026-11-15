package web

import (
	"embed"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed index.html
var files embed.FS

func Register(router *gin.Engine) {
	router.GET("/", func(c *gin.Context) {
		content, err := files.ReadFile("index.html")
		if err != nil {
			c.String(http.StatusInternalServerError, "interface unavailable")
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", content)
	})
}
