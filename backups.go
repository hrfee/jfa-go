package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	BACKUP_PREFIX        = "jfa-go-db-"
	BACKUP_UPLOAD_PREFIX = "upload-"
	BACKUP_DATEFMT       = "2006-01-02T15-04-05"
	BACKUP_SUFFIX        = ".bak"
)

type BackupList struct {
	files []os.DirEntry
	dates []time.Time
	count int
}

func (bl BackupList) Len() int { return len(bl.files) }
func (bl BackupList) Swap(i, j int) {
	bl.files[i], bl.files[j] = bl.files[j], bl.files[i]
	bl.dates[i], bl.dates[j] = bl.dates[j], bl.dates[i]
}

func (bl BackupList) Less(i, j int) bool {
	// Push non-backup files to the end of the array,
	// Since they didn't have a date parsed.
	if bl.dates[i].IsZero() {
		return false
	}
	if bl.dates[j].IsZero() {
		return true
	}
	// Sort by oldest first
	return bl.dates[j].After(bl.dates[i])
}

// Get human-readable file size from f.Size() result.
// https://programming.guide/go/formatting-byte-size-to-human-readable-format.html
func fileSize(l int64) string {
	const unit = 1000
	if l < unit {
		return fmt.Sprintf("%dB", l)
	}
	div, exp := int64(unit), 0
	for n := l / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%c", float64(l)/float64(div), "KMGTPE"[exp])
}

func (app *appContext) getBackups() *BackupList {
	path := app.config.Section("backups").Key("path").String()
	err := os.MkdirAll(path, 0755)
	if err != nil {
		app.err.Printf("Failed to create backup directory \"%s\": %v\n", path, err)
		return nil
	}
	items, err := os.ReadDir(path)
	if err != nil {
		app.err.Printf("Failed to read backup directory \"%s\": %v\n", path, err)
		return nil
	}
	backups := &BackupList{}
	backups.files = items
	backups.dates = make([]time.Time, len(items))
	backups.count = 0
	for i, item := range items {
		if item.IsDir() || !(strings.HasSuffix(item.Name(), BACKUP_SUFFIX)) {
			continue
		}
		t, err := time.Parse(BACKUP_DATEFMT, strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(item.Name(), BACKUP_UPLOAD_PREFIX), BACKUP_PREFIX), BACKUP_SUFFIX))
		if err != nil {
			app.debug.Printf("Failed to parse backup filename \"%s\": %v\n", item.Name(), err)
			continue
		}
		backups.dates[i] = t
		backups.count++
	}
	return backups
}

func (app *appContext) makeBackup() (fileDetails CreateBackupDTO) {
	toKeep := app.config.Section("backups").Key("keep_n_backups").MustInt(20)
	fname := BACKUP_PREFIX + time.Now().Local().Format(BACKUP_DATEFMT) + BACKUP_SUFFIX
	path := app.config.Section("backups").Key("path").String()
	backups := app.getBackups()
	if backups == nil {
		return
	}
	toDelete := backups.count + 1 - toKeep
	// fmt.Printf("toDelete: %d, backCount: %d, keep: %d, length: %d\n", toDelete, backups.count, toKeep, len(backups.files))
	if toDelete > 0 && toDelete <= backups.count {
		sort.Sort(backups)
		for _, item := range backups.files[:toDelete] {
			fullpath := filepath.Join(path, item.Name())
			app.debug.Printf("Deleting old backup \"%s\"\n", item.Name())
			err := os.Remove(fullpath)
			if err != nil {
				app.err.Printf("Failed to delete old backup \"%s\": %v\n", fullpath, err)
				return
			}
		}
	}
	fullpath := filepath.Join(path, fname)
	f, err := os.Create(fullpath)
	if err != nil {
		app.err.Printf("Failed to open backup file \"%s\": %v\n", fullpath, err)
		return
	}
	defer f.Close()
	_, err = app.storage.db.Badger().Backup(f, 0)
	if err != nil {
		app.err.Printf("Failed to create backup: %v\n", err)
		return
	}

	fstat, err := f.Stat()
	if err != nil {
		app.err.Printf("Failed to get info on new backup: %v\n", err)
		return
	}
	fileDetails.Size = fileSize(fstat.Size())
	fileDetails.Name = fname
	fileDetails.Path = fullpath
	// fmt.Printf("Created backup %+v\n", fileDetails)
	return
}

func (app *appContext) loadPendingBackup() {
	if LOADBAK == "" {
		return
	}
	oldPath := filepath.Join(app.dataPath, "db-"+string(time.Now().Unix())+"-pre-"+filepath.Base(LOADBAK))
	app.info.Printf("Moving existing database to \"%s\"\n", oldPath)
	err := os.Rename(app.storage.db_path, oldPath)
	if err != nil {
		app.err.Fatalf("Failed to move existing database: %v\n", err)
	}

	app.ConnectDB()
	defer app.storage.db.Close()

	f, err := os.Open(LOADBAK)
	if err != nil {
		app.err.Fatalf("Failed to open backup file \"%s\": %v\n", LOADBAK, err)
	}
	err = app.storage.db.Badger().Load(f, 256)
	f.Close()
	if err != nil {
		app.err.Fatalf("Failed to restore backup file \"%s\": %v\n", LOADBAK, err)
	}
	app.info.Printf("Restored backup \"%s\".", LOADBAK)
	LOADBAK = ""
}

func newBackupDaemon(app *appContext) *housekeepingDaemon {
	interval := time.Duration(app.config.Section("backups").Key("every_n_minutes").MustInt(1440)) * time.Minute
	daemon := housekeepingDaemon{
		Stopped:         false,
		ShutdownChannel: make(chan string),
		Interval:        interval,
		period:          interval,
		app:             app,
	}
	daemon.jobs = []func(app *appContext){
		func(app *appContext) {
			app.debug.Println("Backups: Creating backup")
			app.makeBackup()
		},
	}
	return &daemon
}
