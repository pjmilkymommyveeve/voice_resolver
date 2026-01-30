package main

import (
	"database/sql"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/lib/pq"
)

// config holds database configuration
type config struct {
	host     string
	port     int
	user     string
	password string
	dbname   string
}

// getEnv gets environment variable with fallback
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// voiceRecording represents a single recording with category
type voiceRecording struct {
	VoiceCategory string `json:"voice_category"`
	Recording     string `json:"recording"`
}

// response represents the api response
type response struct {
	VoiceName       string           `json:"voice_name"`
	VoiceCategories []voiceRecording `json:"voice_categories"`
}

var db *sql.DB

func main() {
	// initialize random seed
	rand.Seed(time.Now().UnixNano())

	// database configuration from environment variables
	dbPort, _ := strconv.Atoi(getEnv("DB_PORT", "5432"))
	cfg := config{
		host:     getEnv("DB_HOST", "localhost"),
		port:     dbPort,
		user:     getEnv("DB_USER", "xdialcore"),
		password: getEnv("DB_PASSWORD", "xdialcore"),
		dbname:   getEnv("DB_NAME", "xdialcore"),
	}

	// connect to database
	var err error
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.host, cfg.port, cfg.user, cfg.password, cfg.dbname)

	db, err = sql.Open("postgres", connStr)
	if err != nil {
		panic(fmt.Sprintf("failed to connect to database: %v", err))
	}
	defer db.Close()

	// test connection
	if err = db.Ping(); err != nil {
		panic(fmt.Sprintf("failed to ping database: %v", err))
	}

	fmt.Printf("database connected: %s@%s/%s\n", cfg.user, cfg.host, cfg.dbname)

	// setup echo
	e := echo.New()
	e.HidePort = true
	e.HideBanner = true

	// middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// routes
	e.GET("/resolve/:campaign_model_id", resolveVoice)

	// start server
	port := getEnv("PORT", "8081")
	fmt.Printf("server starting on port %s\n", port)
	e.Logger.Fatal(e.Start(":" + port))
}

// resolveVoice handles the voice resolution endpoint
func resolveVoice(c echo.Context) error {
	// get campaign_model_id from url params
	campaignModelID := c.Param("campaign_model_id")

	id, err := strconv.Atoi(campaignModelID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid campaign_model_id",
		})
	}

	// get random active campaign_model_voice
	var campaignModelVoiceID int
	var voiceName string

	query := `
		SELECT cmv.id, v.name
		FROM campaign_model_voice cmv
		JOIN voices v ON cmv.voice_id = v.id
		WHERE cmv.campaign_model_id = $1 AND cmv.active = true
		ORDER BY RANDOM()
		LIMIT 1
	`

	err = db.QueryRow(query, id).Scan(&campaignModelVoiceID, &voiceName)
	if err == sql.ErrNoRows {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "no active voices found for this campaign model",
		})
	}
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "database query failed",
		})
	}

	// get all recordings for this campaign_model_voice with their categories
	recordings := []voiceRecording{}

	recordingsQuery := `
		SELECT vc.name, vr.name
		FROM voice_recordings vr
		JOIN voice_recording_categories vrc ON vr.id = vrc.voice_recording_id
		JOIN voice_categories vc ON vrc.voice_category_id = vc.id
		WHERE vr.campaign_model_voice_id = $1
		ORDER BY vc.name
	`

	rows, err := db.Query(recordingsQuery, campaignModelVoiceID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to fetch recordings",
		})
	}
	defer rows.Close()

	// collect all recordings
	for rows.Next() {
		var rec voiceRecording
		if err := rows.Scan(&rec.VoiceCategory, &rec.Recording); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "failed to scan recording",
			})
		}
		recordings = append(recordings, rec)
	}

	if err = rows.Err(); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "error iterating recordings",
		})
	}

	// build response
	res := response{
		VoiceName:       voiceName,
		VoiceCategories: recordings,
	}

	return c.JSON(http.StatusOK, res)
}
