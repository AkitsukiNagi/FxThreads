// Package services defines API backend of the FxThreads project.
package services

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"slices"
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
`

const fetchPost = `
(
	(postID) => {
	const container = document.querySelector(".OuterContainerFull");
	if (!container) return null;

	const AvatarEl = container.querySelector(".AvatarContainer img");
	const NameEl = container.querySelector(".NameContainer a");
	const ContentEl = container.querySelector(".BodyTextContainer");

	const results = {
		author: {
			username: NameEl?.innerText?.trim() || AvatarEl?.alt || "",
			avatarURL: window.cleanURL(AvatarEl?.src),
			url: window.cleanURL(NameEl?.href),
		},
		content: ContentEl?.innerText?.trim() || "",
		medias: [],
		stats: { likes: 0, comments: 0, reposts: 0, shares: 0 },
		id: postID,
		url: "",
	};

	results.url = ` + "`https://www.threads.com/@${results?.author.username}/post/${postID}`" + `;

	const mediaElements = container.querySelectorAll("img, video source");
	const seenUrls = new Set();

	for (const el of mediaElements) {
		if (el.closest(".AvatarContainer")) continue;

		let src = el.src;
		if (el.tagName === "SOURCE") src = el.src;

		if (src && !seenUrls.has(src)) {
			seenUrls.add(src);
			results.medias.push({
				url: window.cleanURL(src),
				isVideo: el.tagName === "SOURCE" || el.tagName === "VIDEO"
			});
		}
		if (results.medias.length >= 4) break;
	}

	const statElements = container.querySelectorAll("span.ActionBarIcon");
	statElements.forEach((el, index) => {
		const countEl = el.querySelector("span.ActionBarCount");

		if (countEl) {
			const val = countEl?.textContent;
			switch (index) {
				case 0:
					results.stats.likes = val;
					break;
				case 1:
					results.stats.comments = val;
					break;
				case 2:
					results.stats.reposts = val;
					break;
				case 3:
					results.stats.shares = val;
					break;
			}
		}
	});

	return results;
	}
)("%s");
`

const fetchVideo = `
(() => {
	const result = {
		coverImage: "",
		height: 0,
		width: 0,
	}

	const firstVideoCover = document.querySelector("meta[property=\"og:image\"]");
	if (firstVideoCover) result.coverImage = window.cleanURL(firstVideoCover.content);

	const coverHeight = document.querySelector("meta[property=\"og:image:height\"]");
	if (coverHeight) result.height = window.parseCount(coverHeight.content);

	const coverWidth = document.querySelector("meta[property=\"og:image:width\"]");
	if (coverWidth) result.width = window.parseCount(coverWidth.content);

	return result;
})();
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

	err := chromedp.Run(bp.Context,
		chromedp.Navigate(constants.ThreadsURL+fmt.Sprintf("/t/%s/embed", postID)),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.Evaluate(jsUtils, nil),
		chromedp.Evaluate(fmt.Sprintf(fetchPost, postID), &post),
	)
	if err != nil {
		slog.Error("Chromedp error", "error", err)
		return nil
	}

	if post == nil {
		slog.Debug("Got empty post", "request_url", constants.ThreadsURL+fmt.Sprintf("/t/%s/embed", postID))
		return nil
	}

	if post.Medias != nil {
		if i := slices.IndexFunc(post.Medias, func(m *types.ThreadsMedia) bool { return m.IsVideo }); i != -1 {
			media := post.Medias[i]

			var partialVideo *types.ThreadsMedia
			err = chromedp.Run(bp.Context,
				chromedp.Navigate(post.URL),
				chromedp.WaitReady("body", chromedp.ByQuery),
				chromedp.Evaluate(jsUtils, nil),
				chromedp.Evaluate(fetchVideo, &partialVideo),
			)
			if err != nil {
				slog.Error("Chromedp error (video cover fetching)")
			} else {
				avatarURL, _ := url.Parse(post.Author.AvatarURL)
				coverURL, _ := url.Parse(partialVideo.CoverImage)

				if path.Base(avatarURL.Path) != path.Base(coverURL.Path) {
					media.CoverImage = partialVideo.CoverImage
					media.Height = partialVideo.Height
					media.Width = partialVideo.Width
					post.Medias[i] = media
				}
			}
		}
	}

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
