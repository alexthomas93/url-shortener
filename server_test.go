package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func setupTest(t *testing.T) (*Server, *echo.Echo) {
	db := initDB(":memory:")
	server := NewServer(db)
	e := echo.New()
	return server, e
}

func shortenURL(t *testing.T, server *Server, e *echo.Echo, url string) ShortenResponse {
	requestBody := map[string]string{"url": url}
	jsonBody, err := json.Marshal(requestBody)
	assert.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBuffer(jsonBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	err = server.shortenURL(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)
	var resp ShortenResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Code)
	assert.NotEmpty(t, resp.URL)
	return resp
}

func TestShortenURL(t *testing.T) {
	server, e := setupTest(t)
	shortenURL(t, server, e, "https://example.com")
}

func TestDeleteURL(t *testing.T) {
	server, e := setupTest(t)
	resp := shortenURL(t, server, e, "https://example.com")

	req := httptest.NewRequest(http.MethodDelete, "/api/"+resp.Code, nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/:code")
	c.SetParamNames("code")
	c.SetParamValues(resp.Code)
	err := server.deleteURL(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestRedirectURL(t *testing.T) {
	server, e := setupTest(t)
	resp := shortenURL(t, server, e, "https://example.com")

	req := httptest.NewRequest(http.MethodGet, "/"+resp.Code, nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/:code")
	c.SetParamNames("code")
	c.SetParamValues(resp.Code)
	err := server.redirectURL(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusFound, rec.Code)
	assert.Equal(t, "https://example.com", rec.Header().Get(echo.HeaderLocation))
}
