// Package services defines API backend of the FxThreads project.
package services

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"sync"
	"time"

	"fxthreads/browser"
	"fxthreads/constants"
	"fxthreads/types"

	"github.com/chromedp/chromedp"
	"github.com/gin-gonic/gin"
)

const jsUtils = `
	window.cleanURL = (urlString) => {
		if (!urlString) return "";

		try {
			const url = new URL(urlString);

			const paramsToDelete = ["xmt", "slof", "_nc_cat", "ccb", "efg"];
			paramsToDelete.forEach(p => url.searchParams.delete(p));

			return url.toString();
		} catch {
			return urlString;
		}
	};

	window.findFirstPost = (obj) => {
		if (!obj || typeof obj !== "object") return null;

		if (obj.hasOwnProperty("post")) return obj.post;

		for (const key in obj) {
			const found = window.findFirstPost(obj[key]);
			if (found) return found;
		}
		return null;
	};
`

const fetchPost = `
((postID) => {
	const scriptEl = Array.from(document.querySelectorAll("script"))
		.find(s => s.textContent.includes("\"thread_items\"") && s.textContent.includes(` + "`\"code\":\"${postID}\"`" + `));
	if (!scriptEl) return null;

	const fullData = JSON.parse(scriptEl.textContent);

	const post = window.findFirstPost(fullData);
	if (!post) return null;

	const result = {
		author: {
			full_name: post.user.full_name,
			username: post.user.username,
			avatar_url: post.user.profile_pic_url,
		},
		content: post.caption?.text,
		medias: [],
		stats: {
			likes: post.like_count,
			comments: post.text_post_app_info.direct_reply_count,
			reposts: post.text_post_app_info.repost_count,
			shares: post.text_post_app_info.reshare_count
		},
		id: post.code,
		timestamp: post.taken_at,
	};

	if (post.carousel_media) {
		for (const m of post.carousel_media) {
			result.medias.push({
				is_video: m.video_versions != null,
				url: m.video_versions ? m.video_versions[0]?.url : m.image_versions2?.candidates[0]?.url,
				cover_image: m.video_versions ? m.image_versions2?.candidates[0]?.url : "",
				width: m.original_width,
				height: m.original_height
			});
		}
	} else if (post.image_versions2) {
	 	result.medias.push({
			is_video: post.video_versions != null,
			url: post.video_versions ? post.video_versions[0]?.url : post.image_versions2?.candidates[0]?.url,
			cover_image: post.video_versions ? post.image_versions2?.candidates[0]?.url : "",
			height: post.original_height,
			width: post.original_width
		});
	}

	return result;
})("%s");
`

var bp *browser.BrowserPool

func init() {
	bp = browser.NewBrowserPool()
}

type ThreadCache struct {
	post      *types.ThreadsPost
	expiresAt time.Time
}

type ThreadsCache struct {
	mu    sync.RWMutex
	items map[string]ThreadCache
}

func (c *ThreadsCache) Set(key string, post *types.ThreadsPost, duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = ThreadCache{
		post:      post,
		expiresAt: time.Now().Add(duration),
	}
}

func (c *ThreadsCache) Get(key string) (*types.ThreadsPost, bool) {
	c.mu.RLock()
	item, found := c.items[key]
	c.mu.RUnlock()

	if !found {
		return nil, false
	}

	if time.Now().After(item.expiresAt) {
		return nil, false
	}

	return item.post, true
}

func (c *ThreadsCache) startReaper(interval time.Duration) {
	ticker := time.NewTicker(interval)
	for range ticker.C {
		c.mu.Lock()
		for k, v := range c.items {
			if time.Now().After(v.expiresAt) {
				delete(c.items, k)
			}
		}
		c.mu.Unlock()
	}
}

func NewPostCache() *ThreadsCache {
	c := &ThreadsCache{
		items: make(map[string]ThreadCache),
	}

	go c.startReaper(1 * time.Minute)
	return c
}

var postCache = NewPostCache()

func GetPostByID(postID string) *types.ThreadsPost {
	if postID == "" {
		return nil
	}

	if cachedPost, found := postCache.Get(postID); found {
		return cachedPost
	}

	var post *types.ThreadsPost
	fetchURL := constants.ThreadsURL + fmt.Sprintf("/t/%s", postID)

	err := chromedp.Run(bp.Context,
		chromedp.Navigate(fetchURL),
		chromedp.Poll(`window.location.href !== "`+fetchURL+`"`, nil),
		chromedp.WaitReady(`body`, chromedp.ByQuery),
		chromedp.Evaluate(jsUtils, nil),
		chromedp.Evaluate(fmt.Sprintf(fetchPost, postID), &post),
	)
	if err != nil {
		slog.Error("Chromedp error", "error", err)
		return nil
	}

	if post == nil {
		slog.Debug("Got empty post", "request_url", constants.ThreadsURL+fmt.Sprintf("/t/%s", postID))
		return nil
	}

	post.Author.URL = constants.ThreadsURL + "/@" + post.Author.Username
	post.URL = constants.ThreadsURL + fmt.Sprintf("/@%s/post/%s", post.Author.Username, post.ID)

	postCache.Set(postID, post, 3*time.Minute)
	return post
}

func PostCrawler(ctx *gin.Context) {
	postID := ctx.Param("postID")
	shareID := ctx.Param("shareID")

	if postID == "" && shareID == "" {
		switch ctx.FullPath() {
		case "/api/post/:postID":
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "No postID was specified."})
		case "/api/share/:shareID":
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "No shareID was specified."})
		default:
			ctx.JSON(http.StatusBadRequest, gin.H{"message": "Neither postID nor shareID was specified."})
		}
		return
	}

	var post *types.ThreadsPost
	post = GetPostByID(postID)
	if shareID != "" {
		post = GetSharedPost(shareID)
	}

	if post == nil {
		ctx.JSON(http.StatusNotFound, gin.H{"message": "The post can't be fetched."})
	} else {
		ctx.JSON(http.StatusOK, post)
	}
}

func GetSharedPost(shareID string) *types.ThreadsPost {
	if shareID == "" {
		return nil
	}

	var postURL string
	chromedp.Run(bp.Context,
		chromedp.Navigate(constants.ThreadsURL+"/share/"+shareID),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.Location(&postURL),
	)

	slog.Debug("Parsed shared post", "parsed_url", postURL)
	p, err := url.Parse(postURL)
	if err != nil {
		slog.Error("Failed to parse URL", "error", err)
		return nil
	}

	postID := path.Base(p.Path)
	slog.Debug("Parsed postID from url", "parsed_id", postID)

	if postID == "" || postID == shareID {
		return nil
	}

	return GetPostByID(postID)
}
