package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"setupforme/models"
	"setupforme/utils"
)

type AppHandler struct {
	db *sql.DB
}

func NewAppHandler(db *sql.DB) *AppHandler {
	return &AppHandler{db: db}
}

func (h *AppHandler) GetApps(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)

	rows, err := h.db.Query(`
		SELECT id, user_id, name, winget_id, download_url, args 
		FROM apps WHERE user_id = $1
	`, userID)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to fetch apps")
		return
	}
	defer rows.Close()

	var apps []models.App
	for rows.Next() {
		var app models.App
		var name, wingetID, downloadURL, args sql.NullString

		err := rows.Scan(&app.ID, &app.UserID, &name, &wingetID, &downloadURL, &args)
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to scan app")
			return
		}

		app.Name = name.String
		app.WingetID = wingetID.String
		app.DownloadURL = downloadURL.String
		app.Args = args.String

		apps = append(apps, app)
	}

	if apps == nil {
		apps = []models.App{}
	}

	json.NewEncoder(w).Encode(apps)
}

func (h *AppHandler) CreateApp(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)

	var req models.CreateAppRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate required fields
	if strings.TrimSpace(req.Name) == "" {
		writeErrorResponse(w, http.StatusBadRequest, "App name is required")
		return
	}

	// Try to auto-resolve winget id by name if both ID and URL are missing
	if strings.TrimSpace(req.WingetID) == "" && strings.TrimSpace(req.DownloadURL) == "" {
		if id, err := utils.ResolveWingetID(req.Name); err == nil && id != "" {
			req.WingetID = id
		} else {
			writeErrorResponse(w, http.StatusBadRequest, "Either winget_id or download_url is required (auto-resolve failed)")
			return
		}
	}

	// Validate download URL if provided
	if req.DownloadURL != "" {
		if !isValidURL(req.DownloadURL) {
			writeErrorResponse(w, http.StatusBadRequest, "Invalid download URL")
			return
		}
	}

	var appID int
	err := h.db.QueryRow(`
		INSERT INTO apps (user_id, name, winget_id, download_url, args) 
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, userID, req.Name, req.WingetID, req.DownloadURL, req.Args).Scan(&appID)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to create app")
		return
	}

	app := models.App{
		ID:          int(appID),
		UserID:      userID,
		Name:        req.Name,
		WingetID:    req.WingetID,
		DownloadURL: req.DownloadURL,
		Args:        req.Args,
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(app)
}

func (h *AppHandler) UpdateApp(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	appID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid app ID")
		return
	}

	// Check if app exists and belongs to user
	var existingUserID int
	err = h.db.QueryRow("SELECT user_id FROM apps WHERE id = $1", appID).Scan(&existingUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			writeErrorResponse(w, http.StatusNotFound, "App not found")
		} else {
			writeErrorResponse(w, http.StatusInternalServerError, "Database error")
		}
		return
	}

	if existingUserID != userID {
		writeErrorResponse(w, http.StatusForbidden, "You can only update your own apps")
		return
	}

	var req models.UpdateAppRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate required fields
	if strings.TrimSpace(req.Name) == "" {
		writeErrorResponse(w, http.StatusBadRequest, "App name is required")
		return
	}

	// Validate that at least winget_id or download_url is provided
	if strings.TrimSpace(req.WingetID) == "" && strings.TrimSpace(req.DownloadURL) == "" {
		writeErrorResponse(w, http.StatusBadRequest, "Either winget_id or download_url is required")
		return
	}

	// Validate download URL if provided
	if req.DownloadURL != "" {
		if !isValidURL(req.DownloadURL) {
			writeErrorResponse(w, http.StatusBadRequest, "Invalid download URL")
			return
		}
	}

	_, err = h.db.Exec(`
		UPDATE apps SET name = $1, winget_id = $2, download_url = $3, args = $4 
		WHERE id = $5
	`, req.Name, req.WingetID, req.DownloadURL, req.Args, appID)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to update app")
		return
	}

	app := models.App{
		ID:          appID,
		UserID:      userID,
		Name:        req.Name,
		WingetID:    req.WingetID,
		DownloadURL: req.DownloadURL,
		Args:        req.Args,
	}

	json.NewEncoder(w).Encode(app)
}

func (h *AppHandler) DeleteApp(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	appID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid app ID")
		return
	}

	// Check if app exists and belongs to user
	var existingUserID int
	err = h.db.QueryRow("SELECT user_id FROM apps WHERE id = $1", appID).Scan(&existingUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			writeErrorResponse(w, http.StatusNotFound, "App not found")
		} else {
			writeErrorResponse(w, http.StatusInternalServerError, "Database error")
		}
		return
	}

	if existingUserID != userID {
		writeErrorResponse(w, http.StatusForbidden, "You can only delete your own apps")
		return
	}

	_, err = h.db.Exec("DELETE FROM apps WHERE id = $1", appID)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to delete app")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AppHandler) GenerateScript(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)

	rows, err := h.db.Query(`
		SELECT name, winget_id, download_url, args 
		FROM apps WHERE user_id = $1
	`, userID)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to fetch apps")
		return
	}
	defer rows.Close()

	// PowerShell single-quote wrapper to safely include arbitrary text
	psSingle := func(s string) string {
		return "'" + strings.ReplaceAll(s, "'", "''") + "'"
	}

	var scriptLines []string
	scriptLines = append(scriptLines, "# SetupForMe - Generated Installation Script")
	scriptLines = append(scriptLines, fmt.Sprintf("# Generated on: %s", time.Now().Format("2006-01-02 15:04:05")))
	scriptLines = append(scriptLines, "")
	scriptLines = append(scriptLines, "$ErrorActionPreference = 'Stop'")
	scriptLines = append(scriptLines, "")
	// Helper functions to make installs robust
	scriptLines = append(scriptLines, "function Install-WingetApp { param([string]$Id, [string]$Args)")
	scriptLines = append(scriptLines, "  $argList = \"-e --id $Id --accept-source-agreements --accept-package-agreements\"")
	scriptLines = append(scriptLines, "  if ($Args -and $Args.Trim() -ne '') { $argList = \"$argList $Args\" }")
	scriptLines = append(scriptLines, "  Write-Host \"winget $argList\" -ForegroundColor Cyan")
	scriptLines = append(scriptLines, "  Start-Process 'winget' -ArgumentList $argList -Wait -NoNewWindow")
	scriptLines = append(scriptLines, "}")
	scriptLines = append(scriptLines, "")
	scriptLines = append(scriptLines, "function Install-FromUrl { param([string]$Url, [string]$Args)")
	scriptLines = append(scriptLines, "  $fileName = [System.IO.Path]::GetFileName(([System.Uri]$Url).AbsolutePath)")
	scriptLines = append(scriptLines, "  if ([string]::IsNullOrWhiteSpace($fileName)) { $fileName = 'installer.exe' }")
	scriptLines = append(scriptLines, "  $dest = Join-Path $env:TEMP (\"SetupForMe_\" + [guid]::NewGuid().ToString() + '_' + $fileName)")
	scriptLines = append(scriptLines, "  Write-Host \"Downloading $Url to $dest\" -ForegroundColor DarkCyan")
	scriptLines = append(scriptLines, "  Invoke-WebRequest -Uri $Url -OutFile $dest")
	scriptLines = append(scriptLines, "  $psi = New-Object System.Diagnostics.ProcessStartInfo")
	scriptLines = append(scriptLines, "  $psi.FileName = $dest")
	scriptLines = append(scriptLines, "  if ($Args -and $Args.Trim() -ne '') { $psi.Arguments = $Args }")
	scriptLines = append(scriptLines, "  $psi.UseShellExecute = $true")
	scriptLines = append(scriptLines, "  $p = [System.Diagnostics.Process]::Start($psi)")
	scriptLines = append(scriptLines, "  $p.WaitForExit()")
	scriptLines = append(scriptLines, "}")
	scriptLines = append(scriptLines, "")
	scriptLines = append(scriptLines, "Write-Host 'Starting application installation...' -ForegroundColor Green")
	scriptLines = append(scriptLines, "")

	appCount := 0
	for rows.Next() {
		var name, wingetID, downloadURL, args sql.NullString
		err := rows.Scan(&name, &wingetID, &downloadURL, &args)
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to scan app")
			return
		}

		appCount++
		appName := name.String
		if appName == "" {
			appName = "Unknown App"
		}

		// Prepare values wrapped for single-quoted PowerShell strings
		psWinget := psSingle(wingetID.String)
		psURL := psSingle(downloadURL.String)
		psArgs := psSingle(args.String)

		scriptLines = append(scriptLines, fmt.Sprintf("# App %d: %s", appCount, appName))
		scriptLines = append(scriptLines, fmt.Sprintf("Write-Host 'Installing %s...' -ForegroundColor Yellow", appName))
		scriptLines = append(scriptLines, "try {")
		if wingetID.String != "" {
			scriptLines = append(scriptLines, fmt.Sprintf("  Install-WingetApp %s %s", psWinget, psArgs))
		} else if downloadURL.String != "" {
			scriptLines = append(scriptLines, fmt.Sprintf("  Install-FromUrl %s %s", psURL, psArgs))
		} else {
			scriptLines = append(scriptLines, "  Write-Host 'No installer info provided.' -ForegroundColor DarkYellow")
		}
		scriptLines = append(scriptLines, fmt.Sprintf("  Write-Host 'Finished: %s' -ForegroundColor Green", appName))
		scriptLines = append(scriptLines, "} catch { Write-Host ('Failed: ' + '"+strings.ReplaceAll(appName, "'", "''")+"' + ' - ' + $_.Exception.Message) -ForegroundColor Red }")
		scriptLines = append(scriptLines, "")
	}

	if appCount == 0 {
		scriptLines = append(scriptLines, `Write-Host "No applications to install." -ForegroundColor Yellow`)
	} else {
		scriptLines = append(scriptLines, `Write-Host "Installation complete!" -ForegroundColor Green`)
	}

	script := strings.Join(scriptLines, "\n")

	response := models.SuccessResponse{
		Message: "Script generated successfully",
		Data:    map[string]string{"script": script},
	}

	json.NewEncoder(w).Encode(response)
}

func isValidURL(rawURL string) bool {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	// Ensure HTTPS for security
	if parsedURL.Scheme != "https" {
		return false
	}

	if parsedURL.Host == "" {
		return false
	}

	return true
}
