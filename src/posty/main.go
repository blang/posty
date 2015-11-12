package main

import (
	"flag"
	"fmt"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/rs/xhandler"
	"github.com/zenazn/goji/web"
	"golang.org/x/net/context"
)

var listen = flag.String("http", ":8080", "Listen on")

func handle(ctx context.Context, handlerc xhandler.HandlerC) web.HandlerFunc {
	return func(c web.C, w http.ResponseWriter, r *http.Request) {
		newctx := context.WithValue(ctx, "urlparams", c.URLParams)
		handlerc.ServeHTTPC(newctx, w, r)
	}
}

// Obsolete pending pull request: https://github.com/rs/xhandler/pull/3
func handlerC(c xhandler.Chain, xh xhandler.HandlerC) xhandler.HandlerC {
	for i := len(c) - 1; i >= 0; i-- {
		xh = c[i](xh)
	}
	return xh
}

func main() {
	flag.Parse()

	// Middleware
	c := xhandler.Chain{}
	c.UseC(xhandler.TimeoutHandler(2 * time.Second))
	// Router
	mux := web.New()
	mainContext := context.Background()
	mux.Get("/:name", handle(mainContext, handlerC(c, xhandler.HandlerFuncC(Index))))
	log.Infof("Listening on %s", *listen)
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func Index(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	params := ctx.Value("urlparams").(map[string]string)
	name := params["name"]
	select {
	case <-time.After(100 * time.Millisecond):
	case <-ctx.Done():
		http.Error(w, "Timeout", 400)
		return
	}
	fmt.Fprintf(w, "Welcome! %s\n", name)
}
