package main

import (
	"fmt"
	"net/http"

	"crypto/sha256"
	"encoding/base64"

	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"

	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

type ShortenRequest struct {
	URL string `json:"url" validate:"required,url"`
}

type ShortenResponse struct {
	URL  string `json:"url"`
	Code string `json:"code"`
}

func initDB() {
	var err error
	DB, err = sql.Open("sqlite3", "./app.db")
	if err != nil {
		log.Fatal(err)
	}
	query := "CREATE TABLE IF NOT EXISTS urls (code TEXT PRIMARY KEY NOT NULL, url TEXT NOT NULL);"
	_, err = DB.Exec(query)
	if err != nil {
		log.Fatalf(err.Error())
	}
	log.Info("Database initialised")
}

func shortenURL(c echo.Context) error {
	var req ShortenRequest
	err := c.Bind(&req)
	if err != nil {
		return err
	}
	hash := sha256.Sum256([]byte(req.URL))
	shortCode := base64.RawURLEncoding.EncodeToString(hash[:6])
	_, err = DB.Exec("INSERT OR IGNORE INTO urls (code, url) VALUES (?, ?)", shortCode, req.URL)
	if err != nil {
		return err
	}
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
	result, err := DB.Exec("DELETE FROM urls WHERE code = ?", code)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "URL not found"})
	}
	c.Logger().Infof("DELETE /api/url/%s", code)
	return c.NoContent(http.StatusNoContent)
}

func redirectURL(c echo.Context) error {
	code := c.Param("code")
	row := DB.QueryRow("SELECT url FROM urls WHERE code = ?", code)
	var url string
	if err := row.Scan(&url); err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "URL not found"})
		}
		return err
	}
	c.Logger().Infof("GET /%s URL:%s", code, url)
	return c.Redirect(http.StatusFound, url)
}

func main() {
	initDB()
	defer DB.Close()
	e := echo.New()
	e.Logger.SetLevel(log.INFO)
	api := e.Group("/api")
	api.POST("/shorten", shortenURL)
	api.DELETE("/url/:code", deleteURL)
	e.GET("/:code", redirectURL)
	e.Logger.Info("Starting server on :1323")
	e.Logger.Fatal(e.Start(":1323"))
}
