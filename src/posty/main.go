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

func handle(ctx context.Context, handlerc xhandler.HandlerC) web.Handler {
	return web.HandlerFunc(func(c web.C, w http.ResponseWriter, r *http.Request) {
		newctx := context.WithValue(ctx, "urlparams", c.URLParams)
		handlerc.ServeHTTPC(newctx, w, r)
	})
}

func main() {
	flag.Parse()

	// Middleware
	c := xhandler.Chain{}
	c.UseC(xhandler.TimeoutHandler(2 * time.Second))
	// Router
	mux := web.New()
	mainContext := context.Background()
	mux.Get("/:name", handle(mainContext, c.HandlerC(xhandler.HandlerFuncC(Index))))
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
