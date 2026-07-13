package services

import (
	"net/http"
	"slices"

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

	var video *types.ThreadsMedia
	if i := slices.IndexFunc(post.Medias, func(m *types.ThreadsMedia) bool { return m.IsVideo }); i != -1 {
		video = post.Medias[i]
	}

	oEmbed := &types.OEmbed{
		Version:      "1.0",
		Type:         "rich",
		ProviderName: "Fxthreads",
		ProviderURL:  "https://fx.akitsuki.me",
		AuthorName:   post.Author.Username,
		AuthorURL:    post.URL,
		Title:        post.Content,
	}

	if video != nil {
		oEmbed.Type = "video"
	}

	if provider != "" {
		oEmbed.AuthorName = provider
	}

	ctx.JSON(http.StatusOK, oEmbed)
}
