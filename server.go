package main

import (
	"fmt"
	"net/http"

	"crypto/sha256"
	"encoding/base64"

	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
)

type ShortenRequest struct {
	URL string `json:"url" validate:"required,url"`
}

type ShortenResponse struct {
	URL  string `json:"url"`
	Code string `json:"code"`
}

var ShortenedURLs = map[string]string{}

func shortenURL(c echo.Context) error {
	var req ShortenRequest
	if err := c.Bind(&req); err != nil {
		return err
	}
	hash := sha256.Sum256([]byte(req.URL))
	shortCode := base64.RawURLEncoding.EncodeToString(hash[:6])
	ShortenedURLs[shortCode] = req.URL
	c.Logger().Infof("POST /api/shorten URL:%s ShortCode:%s", req.URL, shortCode)
	host := c.Request().Host
	redirectURL := fmt.Sprintf("http://%s/%s", host, shortCode)
	response := ShortenResponse{
		URL:  redirectURL,
		Code: shortCode,
	}
	return c.JSON(http.StatusOK, response)
}

func deleteURL(c echo.Context) error {
	code := c.Param("code")
	if _, exists := ShortenedURLs[code]; exists {
		delete(ShortenedURLs, code)
		c.Logger().Infof("DELETE /api/url/%s", code)
		return c.NoContent(http.StatusNoContent)
	}
	return c.JSON(http.StatusNotFound, map[string]string{"error": "URL not found"})
}

func redirectURL(c echo.Context) error {
	code := c.Param("code")
	if url, exists := ShortenedURLs[code]; exists {
		c.Logger().Infof("GET /%s URL:%s", code, url)
		return c.Redirect(http.StatusFound, url)
	}
	return c.JSON(http.StatusNotFound, map[string]string{"error": "URL not found"})
}

func main() {
	e := echo.New()
	e.Logger.SetLevel(log.INFO)
	api := e.Group("/api")
	api.POST("/shorten", shortenURL)
	api.DELETE("/url/:code", deleteURL)
	e.GET("/:code", redirectURL)
	e.Logger.Info("Starting server on :1323")
	e.Logger.Fatal(e.Start(":1323"))
}
