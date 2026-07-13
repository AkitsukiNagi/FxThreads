// Package browser is a manager for chromedp browser
package browser

import (
	"context"
	"log/slog"

	"fxthreads/constants"

	"github.com/chromedp/chromedp"
)

type BrowserPool struct {
	Context context.Context
	Cancel  context.CancelFunc
}

func NewBrowserPool() *BrowserPool {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.NoSandbox,
		chromedp.Headless,
		chromedp.DisableGPU,
		chromedp.IgnoreCertErrors,
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.UserAgent(constants.Agent),
	)

	allocCtx, _ := chromedp.NewExecAllocator(context.Background(), opts...)

	ctx, cancel := chromedp.NewContext(allocCtx)

	if err := chromedp.Run(ctx); err != nil {
		slog.Error("Failed to start browser", "error", err)
		return nil
	}

	return &BrowserPool{
		Context: ctx,
		Cancel:  cancel,
	}
}

func (bp *BrowserPool) Shutdown() {
	bp.Cancel()
}
