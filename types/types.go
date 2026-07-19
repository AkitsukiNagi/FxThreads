// Package types defines custom types for the FxThreads project.
package types

type ThreadsAuthor struct {
	FullName  string `json:"full_name"`
	Username  string `json:"username"`
	AvatarURL string `json:"avatar_url"`
	URL       string `json:"url"`
}

type ThreadsMedia struct {
	IsVideo    bool   `json:"is_video"`
	URL        string `json:"url"`
	CoverImage string `json:"cover_image"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
}

type ThreadsStats struct {
	Likes    int `json:"likes"`
	Comments int `json:"comments"`
	Reposts  int `json:"reposts"`
	Shares   int `json:"shares"`
}

type ThreadsPost struct {
	Author    *ThreadsAuthor  `json:"author"`
	Content   string          `json:"content"`
	Medias    []*ThreadsMedia `json:"medias"`
	Stats     *ThreadsStats   `json:"stats"`
	ID        string          `json:"id"`
	URL       string          `json:"url"`
	Timestamp int             `json:"timestamp"`
}

type OEmbed struct {
	AuthorName   string `json:"author_name"`
	AuthorURL    string `json:"author_url"`
	ProviderName string `json:"provider_name"`
	ProviderURL  string `json:"provider_url"`
	Title        string `json:"title"`
	Type         string `json:"type"`
	Version      string `json:"version"`
}
