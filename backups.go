package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	lm "github.com/hrfee/jfa-go/logmessages"
)

const (
	BACKUP_PREFIX        = "jfa-go-db"
	BACKUP_COMMIT_PREFIX = "-c-"
	BACKUP_DATE_PREFIX   = "-d-"
	BACKUP_UPLOAD_PREFIX = "upload-"
	BACKUP_DATEFMT       = "2006-01-02T15-04-05"
	BACKUP_SUFFIX        = ".bak"
)

type Backup struct {
	Date   time.Time
	Commit string
	Upload bool
}

func (b Backup) IsZero() bool { return b.Date.IsZero() && b.Commit == "" && b.Upload == false }

func (b Backup) Equals(a Backup) bool {
	return a.Date.Equal(b.Date) && a.Commit == b.Commit && a.Upload == b.Upload
}

// Pre 21/03/25 format: "{BACKUP_PREFIX}{date in BACKUP_DATEFMT}{BACKUP_SUFFIX}" = "jfa-go-db-2006-01-02T15-04-05.bak"
// Post 21/03/25 format: "{BACKUP_PREFIX}-c-{commit}-d-{date in BACKUP_DATEFMT}{BACKUP_SUFFIX}" = "jfa-go-db-c-0b92060-d-2006-01-02T15-04-05.bak"

func (b Backup) String() string {
	t := b.Date
	if t.IsZero() {
		t = time.Now()
	}
	out := BACKUP_PREFIX
	if b.Upload {
		out = BACKUP_UPLOAD_PREFIX + out
	}
	if b.Commit != "" {
		out += BACKUP_COMMIT_PREFIX + b.Commit
	}
	out += BACKUP_DATE_PREFIX + t.Local().Format(BACKUP_DATEFMT) + BACKUP_SUFFIX
	return out
}

func (b *Backup) FromString(f string) error {
	of := f
	if strings.HasPrefix(f, BACKUP_UPLOAD_PREFIX) {
		b.Upload = true
		f = f[len(BACKUP_UPLOAD_PREFIX):]
	}
	if !strings.HasPrefix(f, BACKUP_PREFIX) {
		return fmt.Errorf("file doesn't have correct prefix (\"%s\")", BACKUP_PREFIX)
	}
	f = f[len(BACKUP_PREFIX):]
	if !strings.HasSuffix(f, BACKUP_SUFFIX) {
		return fmt.Errorf("file doesn't have correct suffix (\"%s\")", BACKUP_SUFFIX)
	}
	for range 2 {
		if strings.HasPrefix(f, BACKUP_COMMIT_PREFIX) {
			f = f[len(BACKUP_COMMIT_PREFIX):]
			commitEnd := strings.Index(f, BACKUP_DATE_PREFIX)
			if commitEnd == -1 {
				commitEnd = strings.Index(f, BACKUP_SUFFIX)
			}
			if commitEnd == -1 {
				return fmt.Errorf("end of commit (\"%s\" or \"%s\") not found in \"%s\"", BACKUP_DATE_PREFIX, BACKUP_PREFIX, f)
			}
			b.Commit = f[:commitEnd]
			f = f[commitEnd:]
		} else if strings.HasPrefix(f, BACKUP_DATE_PREFIX) {
			f = f[len(BACKUP_DATE_PREFIX):]
			dateEnd := strings.Index(f, BACKUP_COMMIT_PREFIX)
			if dateEnd == -1 {
				dateEnd = strings.Index(f, BACKUP_SUFFIX)
			}
			if dateEnd == -1 {
				return fmt.Errorf("end of date (\"%s\" or \"%s\") not found in \"%s\"", BACKUP_COMMIT_PREFIX, BACKUP_PREFIX, f)
			}
			t, err := time.Parse(BACKUP_DATEFMT, f[:dateEnd])
			if err != nil {
				return err
			}
			b.Date = t
			f = f[dateEnd:]
		}
	}
	if b.Date.IsZero() {
		return b.FromOldString(of)
	}
	return nil
}

func (b *Backup) FromOldString(f string) error {
	t, err := time.Parse(BACKUP_DATEFMT, strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(f, BACKUP_UPLOAD_PREFIX), BACKUP_PREFIX+"-"), BACKUP_SUFFIX))
	if err != nil {
		return fmt.Errorf(lm.FailedParseTime, err)
	}
	b.Date = t
	return nil

}

type BackupList struct {
	files []os.DirEntry
	info  []Backup
	count int
}

func (bl BackupList) Len() int { return len(bl.files) }
func (bl BackupList) Swap(i, j int) {
	bl.files[i], bl.files[j] = bl.files[j], bl.files[i]
	bl.info[i], bl.info[j] = bl.info[j], bl.info[i]
}

func (bl BackupList) Less(i, j int) bool {
	// Push non-backup files to the end of the array,
	// Since they didn't have a date parsed.
	if bl.info[i].Date.IsZero() {
		return false
	}
	if bl.info[j].Date.IsZero() {
		return true
	}
	// Sort by oldest first
	return bl.info[j].Date.After(bl.info[i].Date)
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
		app.err.Printf(lm.FailedCreateDir, path, err)
		return nil
	}
	items, err := os.ReadDir(path)
	if err != nil {
		app.err.Printf(lm.FailedReading, path, err)
		return nil
	}
	backups := &BackupList{}
	backups.files = items
	backups.info = make([]Backup, len(items))
	backups.count = 0
	for i, item := range items {
		// Even though Backup{} can parse and check validity, still check if the file ends in .bak, we don't need to print an error if a file isn't a .bak.
		if item.IsDir() || !(strings.HasSuffix(item.Name(), BACKUP_SUFFIX)) {
			continue
		}
		b := Backup{}
		if err := b.FromString(item.Name()); err != nil {
			app.debug.Printf(lm.FailedParseBackup, item.Name(), err)
			continue
		}
		backups.info[i] = b
		backups.count++
	}
	return backups
}

func (app *appContext) makeBackup() (fileDetails CreateBackupDTO) {
	toKeep := app.config.Section("backups").Key("keep_n_backups").MustInt(20)
	keepPreviousVersions := app.config.Section("backups").Key("keep_previous_version_backup").MustBool(true)

	b := Backup{Commit: commit}
	fname := b.String()
	path := app.config.Section("backups").Key("path").String()
	backups := app.getBackups()
	if backups == nil {
		return
	}
	toDelete := backups.count + 1 - toKeep
	if toDelete > 0 || keepPreviousVersions {
		sort.Sort(backups)
	}
	backupsByCommit := map[string]int{}
	if keepPreviousVersions {
		// Count backups by commit
		for _, b := range backups.info {
			if b.IsZero() {
				continue
			}
			// If b.Commit is empty, the backup is pre-versions-in-backup-names.
			// Still use the empty string as a key, considering these as a single version.
			count, ok := backupsByCommit[b.Commit]
			if !ok {
				count = 0
			}
			count += 1
			backupsByCommit[b.Commit] = count
		}
		fmt.Printf("remaining:%+v\n", backupsByCommit)
	}
	// fmt.Printf("toDelete: %d, backCount: %d, keep: %d, length: %d\n", toDelete, backups.count, toKeep, len(backups.files))
	if toDelete > 0 && toDelete <= backups.count {
		for i := range toDelete {
			backupsRemaining, ok := backupsByCommit[backups.info[i].Commit]
			app.debug.Println("item", backups.files[i], "remaining", backupsRemaining)
			if keepPreviousVersions && ok && backupsRemaining <= 1 {
				continue
			}

			item := backups.files[i]
			fullpath := filepath.Join(path, item.Name())
			err := os.Remove(fullpath)
			if err != nil {
				app.err.Printf(lm.FailedDeleteOldBackup, fullpath, err)
				return
			}
			app.debug.Printf(lm.DeleteOldBackup, fullpath)
			if keepPreviousVersions && ok {
				backupsRemaining -= 1
				backupsByCommit[backups.info[i].Commit] = backupsRemaining
			}
		}
	}
	fullpath := filepath.Join(path, fname)
	f, err := os.Create(fullpath)
	if err != nil {
		app.err.Printf(lm.FailedOpen, fullpath, err)
		return
	}
	defer f.Close()
	_, err = app.storage.db.Badger().Backup(f, 0)
	if err != nil {
		app.err.Printf(lm.FailedCreateBackup, err)
		return
	}

	fstat, err := f.Stat()
	if err != nil {
		app.err.Printf(lm.FailedStat, fullpath, err)
		return
	}
	fileDetails.Size = fileSize(fstat.Size())
	fileDetails.Name = fname
	fileDetails.Path = fullpath
	app.debug.Printf(lm.CreateBackup, fileDetails)
	return
}

func (app *appContext) loadPendingBackup() {
	if LOADBAK == "" {
		return
	}
	oldPath := filepath.Join(app.dataPath, "db-"+strconv.FormatInt(time.Now().Unix(), 10)+"-pre-"+filepath.Base(LOADBAK))
	err := os.Rename(app.storage.db_path, oldPath)
	if err != nil {
		app.err.Fatalf(lm.FailedMoveOldDB, oldPath, err)
	}
	app.info.Printf(lm.MoveOldDB, oldPath)

	app.ConnectDB()
	defer app.storage.db.Close()

	f, err := os.Open(LOADBAK)
	if err != nil {
		app.err.Fatalf(lm.FailedOpen, LOADBAK, err)
	}
	err = app.storage.db.Badger().Load(f, 256)
	f.Close()
	if err != nil {
		app.err.Fatalf(lm.FailedRestoreDB, LOADBAK, err)
	}
	app.info.Printf(lm.RestoreDB, LOADBAK)
	LOADBAK = ""
}

func newBackupDaemon(app *appContext) *GenericDaemon {
	interval := time.Duration(app.config.Section("backups").Key("every_n_minutes").MustInt(1440)) * time.Minute
	d := NewGenericDaemon(interval, app,
		func(app *appContext) {
			app.makeBackup()
		},
	)
	d.Name("Backup")
	return d
}
