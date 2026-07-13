# FxThreads
FxThreads is a proxy service designed to fix Threads.net embeds on platforms like Discord and Telegram. It generates rich metadata previews and automatically strips intrusive tracking parameters from URLs.


## Features
* **Fixes Embeds**: Generates beautiful OpenGraph previews for Discord and other platforms.

* **Privacy Focused**: Automatically removes trackers (e.g., ?xmt=...) from Threads URLs.

* **Modern Support**: Handles both standard post URLs and the new threads.com/share format.

* **oEmbed Support**: Compatible with various social media aggregators.

## How to Use
Simply replace `threads.com` with `fx.akitsuki.me` (or `fxthreads.com` once available) in any Threads URL.

## Endpoints
### User-Facing
| Endpoint | Description |
| --- | --- |
| /@:user/post/:postID | Generates a rich embed for a specific post. |
| /share/:shareID | Generates an embed using the new Threads share ID format. |
| /@:user | Redirects directly to the user's profile on Threads. |

### API
| Endpoint | Description |
| --- | --- |
| `GET` /api/post/:postID | Returns raw post data in JSON format.
| `GET` /api/share/:shareID | Returns post data in JSON format via share ID.
| `GET` /api/oembed?provider=... | Returns oEmbed compliant data for social platforms. |

## Technical Implementation

Unlike traditional crawlers, Threads relies heavily on dynamic JavaScript rendering and currently lacks a public, unauthenticated API.

To solve this, FxThreads utilizes chromedp to drive a headless browser instance. This allows the service to accurately "see" the content as a user would, ensuring embeds remain functional even when Threads updates its frontend logic.

> [!NOTE]
> Because this project uses a headless browser (Chrome) to parse data, it is more resource-intensive than a standard HTTP crawler. Please use the public instance responsibly.

> [!NOTE]
> Link are likely failed to load because the post is private, was deleted, or being marked as sensitive content.

##  Credits & License

* Inspired by the excellent FxEmbed.

* Built with Go and chromedp.

This project is licensed under the AGPL v3.

> [!IMPORTANT]
> We are currently in the process of requesting the fxthreads.com domain for a more permanent and branded experience. For now, please use the fx.akitsuki.me instance.

## TODO
* [ ] Add quoted block
* [ ] Prevent content being cut