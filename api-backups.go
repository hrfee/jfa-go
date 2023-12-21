package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// @Summary Creates a backup of the database.
// @Router /backups [post]
// @Success 200 {object} CreateBackupDTO
// @Security Bearer
// @tags Backups
func (app *appContext) CreateBackup(gc *gin.Context) {
	backup := app.makeBackup()
	gc.JSON(200, backup)
}

// @Summary Download a specific backup file. Requires auth, so can't be accessed plainly in the browser.
// @Param fname path string true "backup filename"
// @Router /backups/{fname} [get]
// @Produce octet-stream
// @Produce json
// @Success 200 {body} file
// @Failure 400 {object} boolResponse
// @Security Bearer
// @tags Backups
func (app *appContext) GetBackup(gc *gin.Context) {
	fname := gc.Param("fname")
	// Hopefully this is enough to ensure the path isn't malicious. Hidden behind bearer auth anyway so shouldn't matter too much I guess.
	ok := (strings.HasPrefix(fname, BACKUP_PREFIX) || strings.HasPrefix(fname, BACKUP_UPLOAD_PREFIX+BACKUP_PREFIX)) && strings.HasSuffix(fname, BACKUP_SUFFIX)
	t, err := time.Parse(BACKUP_DATEFMT, strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(fname, BACKUP_UPLOAD_PREFIX), BACKUP_PREFIX), BACKUP_SUFFIX))
	if !ok || err != nil || t.IsZero() {
		app.debug.Printf("Ignoring backup DL request due to fname: %v\n", err)
		respondBool(400, false, gc)
		return
	}
	path := app.config.Section("backups").Key("path").String()
	fullpath := filepath.Join(path, fname)
	gc.FileAttachment(fullpath, fname)
}

// @Summary Get a list of backups.
// @Router /backups [get]
// @Produce json
// @Success 200 {object} GetBackupsDTO
// @Security Bearer
// @tags Backups
func (app *appContext) GetBackups(gc *gin.Context) {
	path := app.config.Section("backups").Key("path").String()
	backups := app.getBackups()
	sort.Sort(backups)
	resp := GetBackupsDTO{}
	resp.Backups = make([]CreateBackupDTO, backups.count)

	for i, item := range backups.files[:backups.count] {
		resp.Backups[i].Name = item.Name()
		fullpath := filepath.Join(path, item.Name())
		resp.Backups[i].Path = fullpath
		resp.Backups[i].Date = backups.dates[i].Unix()
		fstat, err := os.Stat(fullpath)
		if err == nil {
			resp.Backups[i].Size = fileSize(fstat.Size())
		}
	}
	gc.JSON(200, resp)
}

// @Summary Restore a backup file stored locally to the server.
// @Param fname path string true "backup filename"
// @Router /backups/restore/{fname} [post]
// @Produce json
// @Failure 400 {object} boolResponse
// @Security Bearer
// @tags Backups
func (app *appContext) RestoreLocalBackup(gc *gin.Context) {
	fname := gc.Param("fname")
	// Hopefully this is enough to ensure the path isn't malicious. Hidden behind bearer auth anyway so shouldn't matter too much I guess.
	ok := strings.HasPrefix(fname, BACKUP_PREFIX) && strings.HasSuffix(fname, BACKUP_SUFFIX)
	t, err := time.Parse(BACKUP_DATEFMT, strings.TrimSuffix(strings.TrimPrefix(fname, BACKUP_PREFIX), BACKUP_SUFFIX))
	if !ok || err != nil || t.IsZero() {
		app.debug.Printf("Ignoring backup DL request due to fname: %v\n", err)
		respondBool(400, false, gc)
		return
	}
	path := app.config.Section("backups").Key("path").String()
	fullpath := filepath.Join(path, fname)
	LOADBAK = fullpath
	app.restart(gc)
}

// @Summary Restore a backup file uploaded by the user.
// @Param file formData file true ".bak file"
// @Router /backups/restore [post]
// @Produce json
// @Failure 400 {object} boolResponse
// @Security Bearer
// @tags Backups
func (app *appContext) RestoreBackup(gc *gin.Context) {
	file, err := gc.FormFile("backups-file")
	if err != nil {
		app.err.Printf("Failed to get file from form data: %v\n", err)
		respondBool(400, false, gc)
		return
	}
	app.debug.Printf("Got uploaded file \"%s\"\n", file.Filename)
	path := app.config.Section("backups").Key("path").String()
	fullpath := filepath.Join(path, BACKUP_UPLOAD_PREFIX+BACKUP_PREFIX+time.Now().Local().Format(BACKUP_DATEFMT)+BACKUP_SUFFIX)
	gc.SaveUploadedFile(file, fullpath)
	app.debug.Printf("Saved to \"%s\"\n", fullpath)
	LOADBAK = fullpath
	app.restart(gc)
}
