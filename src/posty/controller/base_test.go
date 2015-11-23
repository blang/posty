package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJsonError(t *testing.T) {
	assert := assert.New(t)
	const output = `{"errors":[{"status":"400","title":"MyError"}]}`
	w := httptest.NewRecorder()
	jsonError(w, nil, cErrClient, "MyError")
	assert.Equal(output, w.Body.String(), "Invalid response")
	assert.Equal(http.StatusBadRequest, w.Code, "Invalid statuscode")
}
