package middleware

import (
	"net/http"

	"github.com/rs/xhandler"
	"golang.org/x/net/context"
)

func JSONWrapper() func(next xhandler.HandlerC) xhandler.HandlerC {
	return func(next xhandler.HandlerC) xhandler.HandlerC {
		return xhandler.HandlerFuncC(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/vnd.api+json")
			next.ServeHTTPC(ctx, w, r)
		})
	}
}
