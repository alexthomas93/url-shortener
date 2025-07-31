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

func TestShortenURL(t *testing.T) {
	db := initDB(":memory:")
	defer db.Close()
	server := NewServer(db)
	e := echo.New()
	requestBody := map[string]string{"url": "https://example.com"}
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
}

func TestDeleteURL(t *testing.T) {
	db := initDB(":memory:")
	defer db.Close()
	server := NewServer(db)
	e := echo.New()
	requestBody := map[string]string{"url": "https://example.com"}
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

	reqDelete := httptest.NewRequest(http.MethodDelete, "/api/"+resp.Code, nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(reqDelete, rec)
	c.SetPath("/:code")
	c.SetParamNames("code")
	c.SetParamValues(resp.Code)
	err = server.deleteURL(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestRedirectURL(t *testing.T) {
	db := initDB(":memory:")
	defer db.Close()
	server := NewServer(db)
	e := echo.New()
	requestBody := map[string]string{"url": "https://example.com"}
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

	reqRedirect := httptest.NewRequest(http.MethodGet, "/"+resp.Code, nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(reqRedirect, rec)
	c.SetPath("/:code")
	c.SetParamNames("code")
	c.SetParamValues(resp.Code)
	err = server.redirectURL(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusFound, rec.Code)
	assert.Equal(t, "https://example.com", rec.Header().Get(echo.HeaderLocation))
}
