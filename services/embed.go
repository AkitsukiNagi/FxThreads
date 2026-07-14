package services

import (
	"fmt"
	"net/http"
	"slices"
	"strings"

	"fxthreads/constants"
	"fxthreads/types"

	"github.com/gin-gonic/gin"
)

func ProvideEmbed(ctx *gin.Context) {
	postID := ctx.Param("postID")
	shareID := ctx.Param("shareID")
	userAgent := strings.ToLower(ctx.Request.Header.Get("User-Agent"))

	isTelegram := strings.Contains(userAgent, "telegrambot")
	isDiscord := strings.Contains(userAgent, "discordbot")

	var post *types.ThreadsPost
	switch ctx.FullPath() {
	case "/@:username/post/:postID":
		post = GetPostByID(postID)
	case "/share/:shareID":
		post = GetSharedPost(shareID)
	default:
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Could not parse the request."})
		return
	}

	if post == nil {
		ctx.JSON(http.StatusNotFound, gin.H{"message": "Post not found. The post may have been deleted, does not exist, or is private."})
		return
	}

	hasVideo := false
	if i := slices.IndexFunc(post.Medias, func(m *types.ThreadsMedia) bool { return m.IsVideo }); i != -1 {
		hasVideo = post.Medias[i].CoverImage != ""
	}
	hasImage := slices.ContainsFunc(post.Medias, func(m *types.ThreadsMedia) bool { return !m.IsVideo })

	cardType := "summary"
	if hasVideo {
		cardType = "player"
	} else if hasImage {
		c := 0
		for _, m := range post.Medias {
			if !m.IsVideo {
				c += 1
			}
			if c > 1 {
				break
			}
		}

		if c == 1 {
			cardType = "summary_large_image"
		}
	}

	socialProof := fmt.Sprintf("❤️ %s　💬 %s　🔁 %s　📤 %s",
		post.Stats.Likes,
		post.Stats.Comments,
		post.Stats.Reposts,
		post.Stats.Shares,
	)

	formattedContent := sanitize(post.Content)

	if isTelegram {
		formattedContent = strings.ReplaceAll(formattedContent, "\n", "<br>")
	}

	var (
		video     *types.ThreadsMedia
		mainImage = post.Author.AvatarURL
	)
	if hasVideo {
		video = post.Medias[slices.IndexFunc(post.Medias, func(m *types.ThreadsMedia) bool { return m.IsVideo })]
	}

	if slices.ContainsFunc(post.Medias, func(m *types.ThreadsMedia) bool { return !m.IsVideo }) {
		mainImage = post.Medias[slices.IndexFunc(post.Medias, func(m *types.ThreadsMedia) bool { return !m.IsVideo })].URL
	}

	ctx.Header("Cache-Control", "public, max-age=3600")

	ctx.HTML(http.StatusOK, "embed.tmpl", gin.H{
		"post":        post,
		"video":       video,
		"cardType":    cardType,
		"socialProof": socialProof,
		"isTelegram":  isTelegram,
		"isDiscord":   isDiscord,
		"content":     formattedContent,
		"title":       sanitize(post.Author.Username),
		"mainImage":   mainImage,
		"baseDomain":  constants.BaseDomain,
	})
}

func sanitize(text string) string {
	chars := map[string]string{
		"&":  "&amp;",
		"<":  "&lt;",
		">":  "&gt;",
		"\"": "&quot;",
		"'":  "&#039;",
	}

	for old, new := range chars {
		text = strings.ReplaceAll(text, old, new)
	}

	return text
}
