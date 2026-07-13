// Package routes defines api routes.
package routes

import (
	"fxthreads/services"

	"github.com/gin-gonic/gin"
)

func RegisterAPIRoutes(rg *gin.RouterGroup) {
	rg.GET("/post/:postID", services.PostCrawler)
	rg.GET("/share/:shareID", services.PostCrawler)
	rg.GET("/oembed/:postID", services.ProvideOEmbed)
	// rg.GET("/activity/:postID")
}
