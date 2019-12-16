package cdp

import (
	"context"
	"testing"

	"github.com/chromedp/chromedp"
)

func TestSliceRun(t *testing.T) {
	cdp := NewChromeDPWithHead()
	cdp.Fetch("", func(i int, item interface{}) {
		ctx := context.Background()
		timeoutCtx, cancel := context.WithTimeout(ctx, cdp.Timeout)
		defer cancel()

		allocCtx, cancel := chromedp.NewRemoteAllocator(timeoutCtx, cdp.WebSocketDebuggerURL)
		defer cancel()

		taskCtx, cancel := chromedp.NewContext(allocCtx)
		defer cancel()
	})
}
