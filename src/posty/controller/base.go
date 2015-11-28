package controller

import (
	"encoding/json"
	"net/http"

	log "github.com/Sirupsen/logrus"
)

const (
	//cErrClient indicates a client error
	cErrClient int = http.StatusBadRequest
	// cErrServer indicates a server error
	cErrServer = http.StatusInternalServerError
)

type errorResponse struct {
	Errors []controllerError `json:"errors"`
}

// jsonError writes a json error object with a message and status code to the the responsewriter.
func jsonError(w http.ResponseWriter, r *http.Request, code int, msg string) {
	log.Warnf("JSON Error: %d %s", code, msg)

	cerr := controllerError{
		Status: code,
		Title:  msg,
	}
	errResp := errorResponse{
		Errors: []controllerError{
			cerr,
		},
	}
	b, err := json.Marshal(errResp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
	w.WriteHeader(code)
	w.Write(b)
}

type controllerError struct {
	Status int    `json:"status,string"`
	Title  string `json:"title"`
}
