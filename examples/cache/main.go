package main

import (
	"os"
	"time"

	"github.com/blend/go-sdk/cache"
	"github.com/blend/go-sdk/ex"
	"github.com/blend/go-sdk/graceful"
	"github.com/blend/go-sdk/logger"
	"github.com/blend/go-sdk/uuid"
	"github.com/blend/go-sdk/web"
)

type dataCacheKey struct{}

func getData() []string {
	time.Sleep(500 * time.Millisecond)
	var output []string
	for x := 0; x < 1024; x++ {
		output = append(output, uuid.V4().String())
	}
	return output
}

func main() {
	log := logger.Prod()
	log.Disable(logger.HTTPRequest, logger.HTTPResponse)
	app := web.New(
		web.OptConfigFromEnv(),
		web.OptLog(log),
		web.OptUse(web.GZip),
		web.OptShutdownGracePeriod(time.Second),
	)
	app.PanicAction = func(_ *web.Ctx, r interface{}) web.Result {
		return web.Text.InternalError(ex.New(r))
	}

	lc := cache.NewLocalCache(cache.OptLocalCacheSweepInterval(500 * time.Millisecond))
	go lc.Start()

	app.GET("/", func(r *web.Ctx) web.Result {
		if data, ok := lc.Get(dataCacheKey{}); ok {
			return web.JSON.Result(data)
		}
		data := getData()
		lc.Set(dataCacheKey{}, data,
			cache.OptValueTTL(1*time.Second),
			cache.OptValueOnRemove(func(_ cache.RemovalReason) {
				log.Infof("item removed")
			}),
		)
		return web.JSON.Result(data)
	})

	if err := graceful.Shutdown(app); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}