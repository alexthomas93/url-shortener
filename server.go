package main

import (
	"fmt"
	"net/http"

	"crypto/sha256"
	"encoding/base64"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
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
		log.Fatalf("Error opening database: %v", err)
	}
	query := "CREATE TABLE IF NOT EXISTS urls (code TEXT PRIMARY KEY NOT NULL, url TEXT NOT NULL);"
	_, err = DB.Exec(query)
	if err != nil {
		log.Fatalf("Error creating table: %v", err)
	}
	log.Info("Database initialised")
}

func shortenURL(c echo.Context) error {
	ctx := c.Request().Context()
	var req ShortenRequest
	err := c.Bind(&req)
	if err != nil {
		c.Logger().Errorf("POST /api/shorten Bind error: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}
	hash := sha256.Sum256([]byte(req.URL))
	shortCode := base64.RawURLEncoding.EncodeToString(hash[:6])
	result, err := DB.ExecContext(ctx, "INSERT OR IGNORE INTO urls (code, url) VALUES (?, ?)", shortCode, req.URL)
	if err != nil {
		c.Logger().Errorf("POST /api/shorten DB error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Database error"})
	}
	if result == nil {
		c.Logger().Errorf("POST /api/shorten No result returned from DB")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Database error"})
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		c.Logger().Errorf("POST /api/shorten RowsAffected error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Database error"})
	}
	host := c.Request().Host
	redirectURL := fmt.Sprintf("http://%s/%s", host, shortCode)
	response := ShortenResponse{
		URL:  redirectURL,
		Code: shortCode,
	}
	if rowsAffected == 0 {
		c.Logger().Infof("POST /api/shorten URL already shortened: %s", req.URL)
		return c.JSON(http.StatusOK, response)
	}
	return c.JSON(http.StatusCreated, response)
}

func deleteURL(c echo.Context) error {
	ctx := c.Request().Context()
	code := c.Param("code")
	result, err := DB.ExecContext(ctx, "DELETE FROM urls WHERE code = ?", code)
	if err != nil {
		c.Logger().Errorf("DELETE /api/%s DB error: %v", code, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to delete URL"})
	}
	if result == nil {
		c.Logger().Errorf("DELETE /api/%s No result returned from DB", code)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to delete URL"})
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		c.Logger().Errorf("DELETE /api/%s RowsAffected error: %v", code, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to delete URL"})
	}
	if rowsAffected == 0 {
		c.Logger().Infof("DELETE /api/%s No rows affected", code)
		return c.JSON(http.StatusNotFound, map[string]string{"error": "URL not found"})
	}
	return c.NoContent(http.StatusNoContent)
}

func redirectURL(c echo.Context) error {
	ctx := c.Request().Context()
	code := c.Param("code")
	row := DB.QueryRowContext(ctx, "SELECT url FROM urls WHERE code = ?", code)
	var url string
	if err := row.Scan(&url); err != nil {
		if err == sql.ErrNoRows {
			c.Logger().Infof("GET /%s URL not found", code)
			return c.JSON(http.StatusNotFound, map[string]string{"error": "URL not found"})
		}
		c.Logger().Errorf("GET /%s DB error: %v", code, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve URL"})
	}
	return c.Redirect(http.StatusFound, url)
}

func main() {
	initDB()
	defer DB.Close()
	e := echo.New()
	e.Logger.SetLevel(log.INFO)
	e.Use(middleware.RequestID())
	e.Use(middleware.BodyLimit("2M"))
	e.Use(middleware.Secure())
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	api := e.Group("/api")
	api.POST("/shorten", shortenURL)
	api.DELETE("/:code", deleteURL)
	e.GET("/:code", redirectURL)
	e.Logger.Info("Starting server on :1323")
	e.Logger.Fatal(e.Start(":1323"))
}
