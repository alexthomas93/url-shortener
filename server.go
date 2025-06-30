package main

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"

	"database/sql"

	"github.com/teris-io/shortid"

	_ "github.com/mattn/go-sqlite3"
)

type Server struct {
	E  *echo.Echo
	DB *sql.DB
}

func NewServer(db *sql.DB) *Server {
	e := echo.New()
	server := &Server{
		E:  e,
		DB: db,
	}
	e.Use(middleware.RequestID())
	e.Use(middleware.BodyLimit("2M"))
	e.Use(middleware.Secure())
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	api := e.Group("/api")
	api.POST("/shorten", server.shortenURL)
	api.DELETE("/:code", server.deleteURL)
	e.GET("/:code", server.redirectURL)
	return server
}

type ShortenRequest struct {
	URL string `json:"url" validate:"required,url"`
}

type ShortenResponse struct {
	URL  string `json:"url"`
	Code string `json:"code"`
}

func initDB(dataSourceName string) *sql.DB {
	var err error
	db, err := sql.Open("sqlite3", dataSourceName)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	query := "CREATE TABLE IF NOT EXISTS urls (code TEXT PRIMARY KEY NOT NULL, url TEXT NOT NULL);"
	_, err = db.Exec(query)
	if err != nil {
		log.Fatalf("Error creating table: %v", err)
	}
	log.Info("Database initialised")
	return db
}

func (s *Server) shortenURL(c echo.Context) error {
	ctx := c.Request().Context()
	var req ShortenRequest
	err := c.Bind(&req)
	if err != nil {
		c.Logger().Errorf("POST /api/shorten Bind error: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}
	host := c.Request().Host
	var existingCode string
	err = s.DB.QueryRowContext(ctx, "SELECT code FROM urls WHERE url = ?", req.URL).Scan(&existingCode)
	if err == nil {
		c.Logger().Infof("POST /api/shorten URL already shortened: %s", req.URL)
		redirectURL := fmt.Sprintf("http://%s/%s", host, existingCode)
		response := ShortenResponse{
			URL:  redirectURL,
			Code: existingCode,
		}
		return c.JSON(http.StatusOK, response)
	}
	shortCode, err := shortid.Generate()
	if err != nil {
		c.Logger().Errorf("POST /api/shorten shortid error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to generate short code"})
	}
	result, err := s.DB.ExecContext(ctx, "INSERT OR IGNORE INTO urls (code, url) VALUES (?, ?)", shortCode, req.URL)
	if err != nil {
		c.Logger().Errorf("POST /api/shorten DB error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Database error"})
	}
	if result == nil {
		c.Logger().Errorf("POST /api/shorten No result returned from DB")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Database error"})
	}
	redirectURL := fmt.Sprintf("http://%s/%s", host, shortCode)
	response := ShortenResponse{
		URL:  redirectURL,
		Code: shortCode,
	}
	return c.JSON(http.StatusCreated, response)
}

func (s *Server) deleteURL(c echo.Context) error {
	ctx := c.Request().Context()
	code := c.Param("code")
	result, err := s.DB.ExecContext(ctx, "DELETE FROM urls WHERE code = ?", code)
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

func (s *Server) redirectURL(c echo.Context) error {
	ctx := c.Request().Context()
	code := c.Param("code")
	row := s.DB.QueryRowContext(ctx, "SELECT url FROM urls WHERE code = ?", code)
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
	db := initDB("./app.db")
	defer db.Close()
	server := NewServer(db)
	server.E.Logger.SetLevel(log.INFO)
	server.E.Logger.Info("Starting server on :1323")
	server.E.Logger.Fatal(server.E.Start(":1323"))
}
