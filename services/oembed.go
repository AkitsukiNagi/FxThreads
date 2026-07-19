package services

import (
	"fmt"
	"net/http"

	"fxthreads/types"

	"github.com/gin-gonic/gin"
)

func ProvideOEmbed(ctx *gin.Context) {
	postID := ctx.Param("postID")
	provider := ctx.Query("provider")

	if postID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "Post id not provided"})
		return
	}

	post := GetPostByID(postID)
	if post == nil {
		ctx.JSON(http.StatusNotFound, gin.H{"message": "Post can't be fetched"})
		return
	}

	oEmbed := &types.OEmbed{
		AuthorName:   fmt.Sprintf("%s (@%s)", post.Author.FullName, post.Author.Username),
		AuthorURL:    post.URL,
		ProviderName: "Fxthreads",
		ProviderURL:  "https://fx.akitsuki.me",
		Title:        post.Content,
		Type:         "rich",
		Version:      "1.0",
	}

	if provider != "" {
		oEmbed.AuthorName = provider
	}

	ctx.JSON(http.StatusOK, oEmbed)
}
