// Package types defines custom types for the FxThreads project.
package types

type ThreadsAuthor struct {
	Username  string `json:"username"`
	AvatarURL string `json:"avatarUrl"`
	URL       string `json:"url"`
}

type ThreadsMedia struct {
	IsVideo    bool   `json:"isVideo"`
	URL        string `json:"url"`
	CoverImage string `json:"coverImage"`
	Height     int    `json:"height"`
	Width      int    `json:"width"`
}

type ThreadsStats struct {
	Likes    string `json:"likes"`
	Comments string `json:"comments"`
	Reposts  string `json:"reposts"`
	Shares   string `json:"shares"`
}

type ThreadsPost struct {
	Author  *ThreadsAuthor  `json:"author"`
	Content string          `json:"content"`
	Medias  []*ThreadsMedia `json:"medias"`
	Stats   *ThreadsStats   `json:"stats"`
	ID      string          `json:"id"`
	URL     string          `json:"url"`
}

type OEmbed struct {
	Version      string `json:"version"`
	Type         string `json:"type"`
	ProviderName string `json:"provider_name"`
	ProviderURL  string `json:"provider_url"`
	AuthorName   string `json:"author_name"`
	AuthorURL    string `json:"author_url"`
	Title        string `json:"title"`
}
