package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/hrfee/jfa-go/common"
)

const (
	baseURL   = "https://builds.hrfee.pw"
	namespace = "hrfee"
	repo      = "jfa-go"
)

var buildTime time.Time = func() time.Time {
	i, _ := strconv.ParseInt(buildTimeUnix, 10, 64)
	return time.Unix(i, 0)
}()

type GHRelease struct {
	HTMLURL     string    `json:"html_url"`
	ID          int       `json:"id"`
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	CreatedAt   time.Time `json:"created_at"`
	PublishedAt time.Time `json:"published_at"`
	Assets      []GHAsset `json:"assets"`
	Body        string    `json:"body"`
}

type GHAsset struct {
	Name               string    `json:"name"`
	State              string    `json:"state"`
	UpdatedAt          time.Time `json:"updated_at"`
	BrowserDownloadURL string    `json:"browser_download_url"`
}

type UnixTime struct {
	time.Time
}

func (t *UnixTime) UnmarshalJSON(b []byte) (err error) {
	unix, err := strconv.ParseInt(strings.TrimPrefix(strings.TrimSuffix(string(b), "\""), "\""), 10, 64)
	if err != nil {
		return
	}
	t.Time = time.Unix(unix, 0)
	return
}

func (t UnixTime) MarshalJSON() ([]byte, error) {
	if t.Time == (time.Time{}) {
		return []byte("\"\""), nil
	}
	return []byte("\"" + strconv.FormatInt(t.Time.Unix(), 10) + "\""), nil
}

var updater string

type BuildType int

const (
	off      BuildType = iota
	internal           // Internal assets through go:embed, no data/.
	external           // External assets in data/, accesses through app.localFS.
	docker             // Only notify of new updates, no self-updating.
)

type ApplyUpdate func() error

type Update struct {
	Version      string      `json:"version"` // vX.X.X or git
	Commit       string      `json:"commit"`
	ReleaseDate  int64       `json:"date"`          // unix time
	Description  string      `json:"description"`   // Commit Name/Release title.
	Changelog    string      `json:"changelog"`     // Changelog, if applicable
	Link         string      `json:"link"`          // Link to commit/release page,
	DownloadLink string      `json:"download_link"` // Optional link to download page.
	CanUpdate    bool        `json:"can_update"`    // Whether or not update can be done automatically.
	update       ApplyUpdate `json:"-"`             // Function to apply update if possible.
}

type Tag struct {
	Ready       bool     `json:"ready"`             // Whether or not build on this tag has completed.
	Version     string   `json:"version,omitempty"` // Version/Commit
	ReleaseDate UnixTime `json:"date"`
}

var goos = map[string]string{
	"darwin":  "macOS",
	"linux":   "Linux",
	"windows": "Windows",
}

var goarch = map[string]string{
	"amd64": "x86_64",
	"arm64": "arm64",
	"arm":   "armv6",
}

//	func newDockerBuild() Update {
//		var tag string
//		if version == "git" {
//			tag = "docker-unstable"
//		} else {
//			tag = "docker-latest"
//		}
//	}
type Updater struct {
	version, commit, tag, url, namespace, name string
	stable                                     bool
	buildType                                  BuildType
	httpClient                                 *http.Client
	timeoutHandler                             common.TimeoutHandler
	binary                                     string
}

func newUpdater(buildroneURL, namespace, repo, version, commit, buildType string) *Updater {
	// fmt.Printf(`Updater intializing with "%s", "%s", "%s", "%s", "%s", "%s"\n`, buildroneURL, namespace, repo, version, commit, buildType)
	bType := off
	tag := ""
	switch buildType {
	case "binary":
		if binaryType == "internal" {
			bType = internal
			tag = "internal"
		} else {
			bType = external
			tag = "external"
		}
	case "docker":
		bType = docker
		if version == "git" {
			tag = "docker-unstable"
		} else {
			tag = "docker-latest"
		}
	default:
		bType = off
	}
	if commit == "unknown" {
		bType = off
	}
	if version == "git" && bType != docker {
		tag += "-git"
	}
	binary := "jfa-go"
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	return &Updater{
		httpClient:     &http.Client{Timeout: 10 * time.Second},
		timeoutHandler: common.NewTimeoutHandler("updater", buildroneURL, true),
		version:        version,
		commit:         commit,
		buildType:      bType,
		tag:            tag,
		url:            buildroneURL,
		namespace:      namespace,
		name:           repo,
		binary:         binary,
	}
}

type BuildDTO struct {
	ID      int64     // `json:"id"`
	Name    string    // `json:"name"`
	Date    time.Time // `json:"date"`
	Link    string    // `json:"link"`
	Message string
	Branch  string // `json:"branch"`
	Tags    map[string]Tag
}

// SetTransport sets the http.Transport to use for requests. Can be used to set a proxy.
func (ud *Updater) SetTransport(t *http.Transport) {
	ud.httpClient.Transport = t
}

func (ud *Updater) GetTag() (Tag, int, error) {
	if ud.buildType == off {
		return Tag{}, -1, nil
	}
	url := fmt.Sprintf("%s/repo/%s/%s/tag/latest/%s", ud.url, ud.namespace, ud.name, ud.tag)
	// fmt.Printf("Pinging URL \"%s\" for updates\n", url)
	req, _ := http.NewRequest("GET", url, nil)
	resp, err := ud.httpClient.Do(req)
	defer ud.timeoutHandler()
	if err != nil || resp.StatusCode != 200 {
		return Tag{}, resp.StatusCode, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Tag{}, -1, err
	}

	var tag Tag
	err = json.Unmarshal(body, &tag)
	if tag.Version == "" {
		err = errors.New("Tag at \"" + url + "\" was empty")
	}
	return tag, resp.StatusCode, err
}

func (t *Tag) IsNew() bool {
	// fmt.Printf("Build Time: %+v, Release Date: %+v", buildTime, t.ReleaseDate)
	// Add 20 minutes to account for build time
	return t.Version[:7] != commit && t.Ready && t.ReleaseDate.After(buildTime.Add(time.Duration(20)*time.Minute))
}

func (ud *Updater) getRelease() (release GHRelease, status int, err error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", ud.namespace, ud.name)
	req, _ := http.NewRequest("GET", url, nil)
	resp, err := ud.httpClient.Do(req)
	status = resp.StatusCode
	defer ud.timeoutHandler()
	if err != nil || resp.StatusCode != 200 {
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	err = json.Unmarshal(body, &release)
	return
}

func (ud *Updater) GetUpdate(tag Tag) (update Update, status int, err error) {
	switch ud.buildType {
	case internal:
		if ud.tag == "internal-git" {
			update, status, err = ud.getUpdateInternalGit(tag)
		} else if ud.tag == "internal" {
			update, status, err = ud.getUpdateInternal(tag)
		}
	case external, docker:
		if strings.Contains(ud.tag, "git") || ud.tag == "docker-unstable" {
			update, status, err = ud.getCommitGit(tag)
		} else {
			var release GHRelease
			release, status, err = ud.getRelease()
			if err != nil {
				return
			}
			update = Update{
				Changelog:   release.Body,
				Description: release.Name,
				Version:     release.TagName,
				Commit:      tag.Version,
				Link:        release.HTMLURL,
				ReleaseDate: release.PublishedAt.Unix(),
			}
		}
		if ud.buildType == docker {
			update.DownloadLink = fmt.Sprintf("https://hub.docker.com/r/%s/%s/tags", ud.namespace, ud.name)
		}
	}
	return
}

func (ud *Updater) getUpdateInternal(tag Tag) (update Update, status int, err error) {
	release, status, err := ud.getRelease()
	update = Update{
		Changelog:   release.Body,
		Description: release.Name,
		Version:     release.TagName,
		Commit:      tag.Version,
		Link:        release.HTMLURL,
		ReleaseDate: release.PublishedAt.Unix(),
	}
	if err != nil || status != 200 {
		return
	}
	updateFunc, status, err := ud.downloadInternal(&release.Assets, tag)
	if err == nil && status == 200 {
		update.CanUpdate = true
		update.update = updateFunc
	}
	return
}

func (ud *Updater) getCommitGit(tag Tag) (update Update, status int, err error) {
	url := fmt.Sprintf("%s/repo/%s/%s/build/%s", ud.url, ud.namespace, ud.name, tag.Version)
	req, _ := http.NewRequest("GET", url, nil)
	resp, err := ud.httpClient.Do(req)
	status = resp.StatusCode
	defer ud.timeoutHandler()
	if err != nil || resp.StatusCode != 200 {
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	var build BuildDTO
	err = json.Unmarshal(body, &build)
	if err != nil {
		return
	}
	update = Update{
		Description: build.Name,
		Version:     "git",
		Commit:      tag.Version,
		Link:        build.Link,
		ReleaseDate: tag.ReleaseDate.Unix(),
	}
	return
}

func (ud *Updater) getUpdateInternalGit(tag Tag) (update Update, status int, err error) {
	update, status, err = ud.getCommitGit(tag)
	if err != nil || status != 200 {
		return
	}
	updateFunc, status, err := ud.downloadInternalGit()
	if err == nil && status == 200 {
		update.CanUpdate = true
		update.update = updateFunc
	}
	return
}

func getBuildName() string {
	operatingSystem, ok := goos[runtime.GOOS]
	if !ok {
		for _, v := range goos {
			if strings.Contains(v, runtime.GOOS) {
				operatingSystem = v
				break
			}
		}
	}
	if operatingSystem == "" {
		return ""
	}
	arch, ok := goarch[runtime.GOARCH]
	if !ok {
		for _, v := range goarch {
			if strings.Contains(v, runtime.GOARCH) {
				arch = v
				break
			}
		}
	}
	if arch == "" {
		return ""
	}
	tray := ""
	if TRAY {
		tray = "TrayIcon_"
	}
	return tray + operatingSystem + "_" + arch
}

func (ud *Updater) downloadInternal(assets *[]GHAsset, tag Tag) (applyUpdate ApplyUpdate, status int, err error) {
	return ud.pullInternal(ud.getInternalURL(assets, tag))
}

func (ud *Updater) downloadInternalGit() (applyUpdate ApplyUpdate, status int, err error) {
	return ud.pullInternal(ud.getInternalGitURL())
}

func (ud *Updater) getInternalURL(assets *[]GHAsset, tag Tag) string {
	buildName := getBuildName()
	if buildName == "" {
		return ""
	}
	url := ""
	for _, asset := range *assets {
		if strings.Contains(asset.Name, buildName) {
			url = asset.BrowserDownloadURL
			break
		}
	}
	return url
}

func (ud *Updater) getInternalGitURL() string {
	buildName := getBuildName()
	if buildName == "" {
		return ""
	}
	return fmt.Sprintf("%s/repo/%s/%s/latest/file/%s", ud.url, ud.namespace, ud.name, buildName)
}

func (ud *Updater) pullInternal(url string) (applyUpdate ApplyUpdate, status int, err error) {
	if url == "" {
		return
	}
	req, _ := http.NewRequest("GET", url, nil)
	resp, err := ud.httpClient.Do(req)
	status = resp.StatusCode
	if err != nil || resp.StatusCode != 200 {
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		status = -1
		return
	}
	zp, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		status = -1
		return
	}
	for _, zf := range zp.File {
		if zf.Name != ud.binary {
			continue
		}
		var file string
		file, err = os.Executable()
		if err != nil {
			return
		}
		var path string
		path, err = filepath.EvalSymlinks(file)
		if err != nil {
			return
		}
		var info fs.FileInfo
		info, err = os.Stat(path)
		if err != nil {
			return
		}
		mode := info.Mode()
		var unzippedFile io.ReadCloser
		unzippedFile, err = zf.Open()
		if err != nil {
			return
		}
		defer unzippedFile.Close()
		var f *os.File
		f, err = os.OpenFile(path+"_", os.O_RDWR|os.O_CREATE|os.O_TRUNC, mode)
		if err != nil {
			return
		}
		defer f.Close()
		_, err = io.Copy(f, unzippedFile)
		if err != nil {
			return
		}
		applyUpdate = func() error {
			oldName := path + "-" + version + "-" + commit
			err := os.Rename(path, oldName)
			if err != nil {
				return err
			}
			err = os.Rename(path+"_", path)
			if err != nil {
				return err
			}
			return os.Remove(oldName)
		}
		return
	}
	// gz, err := gzip.NewReader(resp.Body)
	// if err != nil {
	// 	status = -1
	// 	return
	// }
	// defer gz.Close()
	// tarReader := tar.NewReader(gz)
	// var header *tar.Header
	// for {
	// 	header, err = tarReader.Next()
	// 	if err == io.EOF {
	// 		break
	// 	}
	// 	if err != nil {
	// 		status = -1
	// 		return
	// 	}
	// 	switch header.Typeflag {
	// 	case tar.TypeReg:
	// 		// Search only for file named ud.binary
	// 		if header.Name == ud.binary {
	// 			var file string
	// 			file, err = os.Executable()
	// 			if err != nil {
	// 				return
	// 			}
	// 			var path string
	// 			path, err = filepath.EvalSymlinks(file)
	// 			if err != nil {
	// 				return
	// 			}
	// 			var info fs.FileInfo
	// 			info, err = os.Stat(path)
	// 			if err != nil {
	// 				return
	// 			}
	// 			mode := info.Mode()
	// 			var f *os.File
	// 			f, err = os.OpenFile(path+"_", os.O_RDWR|os.O_CREATE|os.O_TRUNC, mode)
	// 			if err != nil {
	// 				return
	// 			}
	// 			defer f.Close()
	// 			_, err = io.Copy(f, tarReader)
	// 			if err != nil {
	// 				return
	// 			}
	// 			applyUpdate = func() error {
	// 				oldName := path + "-" + version + "-" + commit
	// 				err := os.Rename(path, oldName)
	// 				if err != nil {
	// 					return err
	// 				}
	// 				err = os.Rename(path+"_", path)
	// 				if err != nil {
	// 					return err
	// 				}
	// 				return os.Remove(oldName)
	// 			}
	// 			return
	// 		}
	// 	}
	// }
	err = errors.New("Couldn't find file: " + ud.binary)
	return
}

// func newInternalBuild() Update {
// 	tag := "internal"

//	func update(path string) err {
//		if
//		fp, err := os.Executable()
//		if err != nil {
//			return err
//		}
//		fullPath, err := filepath.EvalSymlinks(fp)
//		if err != nil {
//			return err
//		}
//		newBinary,
//	}
func (app *appContext) checkForUpdates() {
	for {
		go func() {
			tag, status, err := app.updater.GetTag()
			if status != 200 || err != nil {
				if err != nil && strings.Contains(err.Error(), "strconv.ParseInt") {
					app.err.Println("No new updates available.")
				} else if status != -1 { // -1 means updates disabled, we don't need to log it.
					app.err.Printf("Failed to get latest tag (%d): %v", status, err)
				}
				return
			}
			if tag != app.tag && tag.IsNew() {
				app.info.Println("Update found")
				update, status, err := app.updater.GetUpdate(tag)
				if status != 200 || err != nil {
					app.err.Printf("Failed to get update (%d): %v", status, err)
					return
				}
				app.tag = tag
				app.update = update
				app.newUpdate = true
			}
		}()
		time.Sleep(30 * time.Minute)
	}
}
