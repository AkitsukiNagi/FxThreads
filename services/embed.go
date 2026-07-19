package services

import (
	"fmt"
	"math"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"fxthreads/constants"
	"fxthreads/types"

	"github.com/gin-gonic/gin"
)

func ProvideEmbed(ctx *gin.Context) {
	postID := ctx.Param("postID")
	shareID := ctx.Param("shareID")
	userAgent := strings.ToLower(ctx.Request.Header.Get("user-agent"))

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

	hasVideo := slices.IndexFunc(post.Medias, func(m *types.ThreadsMedia) bool { return m.IsVideo }) != -1
	hasImage := slices.ContainsFunc(post.Medias, func(m *types.ThreadsMedia) bool { return !m.IsVideo })

	cardType := "summary"
	if hasVideo {
		cardType = "player"
	} else if hasImage {
		cardType = "summary_large_image"
	}

	socialProof := fmt.Sprintf("❤️ %s　💬 %s　🔁 %s　📤 %s",
		humanize(post.Stats.Likes),
		humanize(post.Stats.Comments),
		humanize(post.Stats.Reposts),
		humanize(post.Stats.Shares),
	)

	formattedContent := post.Content
	if hasVideo {
		formattedContent += "\n\n" + socialProof
	}

	formattedContent = sanitize(formattedContent)

	if isTelegram {
		formattedContent = strings.ReplaceAll(formattedContent, "\n", "<br>")
	}

	var video *types.ThreadsMedia
	if hasVideo {
		video = post.Medias[slices.IndexFunc(post.Medias, func(m *types.ThreadsMedia) bool { return m.IsVideo })]
	}

	t, _ := time.Parse(time.RFC3339, strconv.Itoa(post.Timestamp))

	ctx.Header("Cache-Control", "public, max-age=3600")

	ctx.HTML(http.StatusOK, "embed.tmpl", gin.H{
		"post":        post,
		"video":       video,
		"cardType":    cardType,
		"socialProof": socialProof,
		"isTelegram":  isTelegram,
		"isDiscord":   isDiscord,
		"content":     formattedContent,
		"title":       sanitize(fmt.Sprintf("%s (@%s)", post.Author.FullName, post.Author.Username)),
		"baseDomain":  constants.BaseDomain,
		"timestamp":   t.String(),
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

func humanize(num int) string {
	if num < 1000 {
		return strconv.Itoa(num)
	}

	units := []string{"K", "M", "B", "T"}

	f := float64(num)

	exp := math.Floor(math.Log10(f) / 3)

	index := int(exp - 1)

	if index >= len(units) {
		index = len(units) - 1
		exp = float64(len(units))
	}

	val := f / math.Pow(10, exp*3)

	res := fmt.Sprintf("%.1f%s", val, units[index])

	return strings.Replace(res, ".0", "", 1)
}
