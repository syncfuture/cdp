package cdp

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"runtime"
	"time"

	"github.com/chromedp/chromedp"

	"github.com/syncfuture/go/task"

	"os/exec"

	"github.com/syncfuture/go/config"
	log "github.com/syncfuture/go/slog"
	"github.com/syncfuture/go/u"
)

type ChromeDPWithHead struct {
	ConfigProvider       config.IConfigProvider
	ChromeCMD            string
	ExcelFile            string
	WebSocketDebuggerURL string
	WaitChromeDelay      int
	BatchSize            int
	BatchInterval        int
	Timeout              time.Duration
	SliceScheduler       task.ISliceScheduler
}

func NewChromeDPWithHead(batchEvents ...func(int, int)) (r *ChromeDPWithHead) {
	r = new(ChromeDPWithHead)
	r.ConfigProvider = config.NewJsonConfigProvider()
	if runtime.GOOS == "windows" {
		r.ChromeCMD = r.ConfigProvider.GetString("WinChromeCMD")
	} else if runtime.GOOS == "darwin" {
		r.ChromeCMD = r.ConfigProvider.GetString("MacChromeCMD")
	}
	r.WaitChromeDelay = r.ConfigProvider.GetIntDefault("WaitChromeDelay", 2000)
	r.BatchSize = r.ConfigProvider.GetIntDefault("BatchSize", 4)
	r.BatchInterval = r.ConfigProvider.GetIntDefault("BatchInterval", 500)
	timeout := r.ConfigProvider.GetIntDefault("Timeout", 10000)
	r.Timeout = time.Duration(timeout) * time.Millisecond
	r.SliceScheduler = task.NewBatchScheduler(r.BatchSize, r.BatchInterval, batchEvents...)
	return r
}

func (x *ChromeDPWithHead) Init() {
	portOpen := isPortOpen(9222)

	if !portOpen {
		log.Info("Starting chrome...")

		cmd := exec.Command(x.ChromeCMD, "--remote-debugging-port=9222")
		err := cmd.Start()
		time.Sleep(time.Duration(x.WaitChromeDelay) * time.Millisecond)
		if u.LogError(err) {
			return
		}
	}

	resp, err := http.Get("http://localhost:9222/json/version")
	if u.LogError(err) {
		return
	}
	defer resp.Body.Close()
	configJson, err := ioutil.ReadAll(resp.Body)
	if u.LogError(err) {
		return
	}

	config := make(map[string]string)
	json.Unmarshal(configJson, &config)

	x.WebSocketDebuggerURL = config["webSocketDebuggerUrl"]
	if x.WebSocketDebuggerURL == "" {
		log.Fatal("get webSocketDebuggerUrl failed")
	}
	log.Debug("Connecting to ", x.WebSocketDebuggerURL)
}

func (x *ChromeDPWithHead) Fetch(action func(ctx context.Context)) {
	x.Init()

	ctx := context.Background()
	timeoutCtx, cancel := context.WithTimeout(ctx, x.Timeout)
	defer cancel()

	allocCtx, cancel := chromedp.NewRemoteAllocator(timeoutCtx, x.WebSocketDebuggerURL)
	defer cancel()

	taskCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	action(taskCtx)
}

func (x *ChromeDPWithHead) SliceFetch(slicePtr interface{}, action func(ctx context.Context, i int, v interface{})) {
	x.Init()

	x.SliceScheduler.SliceRun(slicePtr, func(i int, item interface{}) {
		ctx := context.Background()
		timeoutCtx, cancel := context.WithTimeout(ctx, x.Timeout)
		defer cancel()

		allocCtx, cancel := chromedp.NewRemoteAllocator(timeoutCtx, x.WebSocketDebuggerURL)
		defer cancel()

		taskCtx, cancel := chromedp.NewContext(allocCtx)
		defer cancel()

		action(taskCtx, i, item)
	})
}

func (x *ChromeDPWithHead) Cancel() {
	x.SliceScheduler.Cancel()
}
