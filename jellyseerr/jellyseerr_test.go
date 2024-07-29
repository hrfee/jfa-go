package jellyseerr

import (
	"testing"

	"github.com/hrfee/jfa-go/common"
)

const (
	API_KEY = "MTcyMjI2MDM2MTYyMzMxNDZkZmYyLTE4MzMtNDUyNy1hODJlLTI0MTZkZGUyMDg2Ng=="
	URI     = "http://localhost:5055"
)

func client() *Jellyseerr {
	return NewJellyseerr(URI, API_KEY, common.NewTimeoutHandler("Jellyseerr", URI, false))
}

func TestMe(t *testing.T) {
	js := client()
	u, err := js.Me()
	if err != nil {
		t.Fatalf("returned error %+v", err)
	}
	if u.ID < 0 {
		t.Fatalf("returned no user %+v\n", u)
	}
}

func TestImportFromJellyfin(t *testing.T) {
	js := client()
	list, err := js.ImportFromJellyfin("6b75e189efb744f583aa2e8e9cee41d3")
	if err != nil {
		t.Fatalf("returned error %+v", err)
	}
	if len(list) == 0 {
		t.Fatalf("returned no users")
	}
}

func TestMustGetUser(t *testing.T) {
	js := client()
	u, err := js.MustGetUser("8c9d25c070d641cd8ad9cf825f622a16")
	if err != nil {
		t.Fatalf("returned error %+v", err)
	}
	if u.ID < 0 {
		t.Fatalf("returned no users")
	}
}
