package scm_clients

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strconv"
	"testing"

	"github.com/drone/drone-go/drone"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const mockGitlabToken = "7535706b694c63526c6e4f5230374243"

func TestGitlabClient_GetFileContents(t *testing.T) {
	ts := httptest.NewServer(testMuxGitlab())
	defer ts.Close()
	client, err := createGitlabClient(ts.URL)
	if err != nil {
		t.Error(err)
		return
	}
	BaseTest_GetFileContents(t, client)
}

func TestGitlabClient_ChangedFilesInDiff(t *testing.T) {
	ts := httptest.NewServer(testMuxGitlab())
	defer ts.Close()
	client, err := createGitlabClient(ts.URL)
	if err != nil {
		t.Error(err)
		return
	}
	BaseTest_ChangedFilesInDiff(t, client)
}

func TestGitlabClient_ChangedFilesInPullRequest(t *testing.T) {
	ts := httptest.NewServer(testMuxGitlab())
	defer ts.Close()
	client, err := createGitlabClient(ts.URL)
	if err != nil {
		t.Error(err)
		return
	}

	BaseTest_ChangedFilesInPullRequest(t, client)
}

func TestGitlabClient_GetFileListing(t *testing.T) {
	ts := httptest.NewServer(testMuxGitlab())
	defer ts.Close()
	client, err := createGitlabClient(ts.URL)
	if err != nil {
		t.Error(err)
		return
	}

	BaseTest_GetFileListing(t, client)
}

func TestGitlabClient_GetFileListing_Large(t *testing.T) {
	ts := httptest.NewServer(testMuxGitlab())
	defer ts.Close()
	client, err := createGitlabClient(ts.URL)
	if err != nil {
		t.Error(err)
		return
	}

	actualFiles, err := client.GetFileListing(noContext, "alargefolder", "8ecad91991d5da985a2a8dd97cc19029dc1c2899")
	if err != nil {
		t.Error(err)
		return
	}

	expectedFiles := []FileListingEntry{
		{Type: "file", Path: "alargefolder/.drone.yml", Name: ".drone.yml"},
		{Type: "file", Path: "alargefolder/file01", Name: "file01"},
		{Type: "file", Path: "alargefolder/file02", Name: "file02"},
		{Type: "file", Path: "alargefolder/file03", Name: "file03"},
		{Type: "file", Path: "alargefolder/file04", Name: "file04"},
		{Type: "file", Path: "alargefolder/file05", Name: "file05"},
		{Type: "file", Path: "alargefolder/file06", Name: "file06"},
		{Type: "file", Path: "alargefolder/file07", Name: "file07"},
		{Type: "file", Path: "alargefolder/file08", Name: "file08"},
		{Type: "file", Path: "alargefolder/file09", Name: "file09"},
		{Type: "file", Path: "alargefolder/file10", Name: "file10"},
		{Type: "file", Path: "alargefolder/file11", Name: "file11"},
		{Type: "dir", Path: "alargefolder/folder01", Name: "folder01"},
		{Type: "dir", Path: "alargefolder/folder02", Name: "folder02"},
		{Type: "dir", Path: "alargefolder/folder03", Name: "folder03"},
		{Type: "dir", Path: "alargefolder/folder04", Name: "folder04"},
		{Type: "dir", Path: "alargefolder/folder05", Name: "folder05"},
		{Type: "dir", Path: "alargefolder/folder06", Name: "folder06"},
		{Type: "dir", Path: "alargefolder/folder07", Name: "folder07"},
		{Type: "dir", Path: "alargefolder/folder08", Name: "folder08"},
		{Type: "dir", Path: "alargefolder/folder09", Name: "folder09"},
		{Type: "dir", Path: "alargefolder/folder10", Name: "folder10"},
		{Type: "dir", Path: "alargefolder/folder11", Name: "folder11"},
	}

	if want, got := expectedFiles, actualFiles; !reflect.DeepEqual(want, got) {
		t.Errorf("Test failed:\n  want %q\n   got %q", want, got)
	}
}

func createGitlabClient(server string) (ScmClient, error) {
	someUUID := uuid.New()
	repo := drone.Repo{
		UID:       "1234",
		Namespace: "foosinn",
		Name:      "dronetest",
		Slug:      "foosinn/dronetest",
	}
	return NewGitLabClient(noContext, someUUID, server, mockGitlabToken, repo)
}

func testMuxGitlab() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v4/projects/1234/repository/tree",
		func(w http.ResponseWriter, r *http.Request) {
			if r.FormValue("path") == "afolder" && r.FormValue("ref") == "8ecad91991d5da985a2a8dd97cc19029dc1c2899" {
				f, _ := os.Open("../testdata/gitlab/afolder.json")
				_, _ = io.Copy(w, f)
			} else if r.FormValue("path") == "alargefolder" && r.FormValue("ref") == "8ecad91991d5da985a2a8dd97cc19029dc1c2899" {
				jsonList, _ := ioutil.ReadFile("../testdata/gitlab/alargefolder.json")
				var folderList []map[string]string
				json.Unmarshal(jsonList, &folderList)
				pageVal := r.FormValue("page")
				if pageVal == "" {
					pageVal = "1"
				}
				page, _ := strconv.Atoi(pageVal)
				perPageVal := r.FormValue("perPage")
				if perPageVal == "" {
					perPageVal = "20" // default value for GitLab API
				}
				perPage, _ := strconv.Atoi(perPageVal)
				first := (page - 1) * perPage
				last := page * perPage // last index (excluded)
				if last > len(folderList) {
					last = len(folderList)
				}
				sublist := make([]map[string]string, last-first)
				for i := first; i < last; i++ {
					sublist[i-first] = folderList[i]
				}
				var nextPage string
				if last < len(folderList) {
					nextPage = strconv.Itoa(page + 1)
				}
				var prevPage string
				if first != 0 {
					prevPage = strconv.Itoa(page - 1)
				}
				var totalPages int
				if len(folderList)%perPage == 0 {
					totalPages = len(folderList) / perPage
				} else {
					totalPages = (len(folderList) / perPage) + 1
				}
				sublistJson, _ := json.Marshal(sublist)
				respHeader := w.Header()
				respHeader.Set("x-next-page", nextPage)
				respHeader.Set("x-page", strconv.Itoa(page))
				respHeader.Set("x-per-page", strconv.Itoa(perPage))
				respHeader.Set("x-prev-page", prevPage)
				respHeader.Set("x-total", strconv.Itoa(len(folderList)))
				respHeader.Set("x-total-pages", strconv.Itoa(totalPages))
				_, _ = io.Copy(w, bytes.NewReader(sublistJson))
			}
		})
	mux.HandleFunc("/api/v4/projects/1234/repository/compare",
		func(w http.ResponseWriter, r *http.Request) {
			if r.FormValue("from") == "2897b31ec3a1b59279a08a8ad54dc360686327f7" && r.FormValue("to") == "8ecad91991d5da985a2a8dd97cc19029dc1c2899" {
				f, _ := os.Open("../testdata/gitlab/compare.json")
				_, _ = io.Copy(w, f)
			}
		})
	mux.HandleFunc("/api/v4/projects/1234/merge_requests/3/changes",
		func(w http.ResponseWriter, r *http.Request) {
			f, _ := os.Open("../testdata/gitlab/pull_3_files.json")
			_, _ = io.Copy(w, f)
		})
	mux.HandleFunc("/api/v4/projects/1234/repository/files/",
		func(w http.ResponseWriter, r *http.Request) {
			if r.URL.RawPath == "/api/v4/projects/1234/repository/files/.drone.yml" {
				f, _ := os.Open("../testdata/gitlab/.drone.yml.json")
				_, _ = io.Copy(w, f)
			}
			if r.URL.RawPath == "/api/v4/projects/1234/repository/files/a%2Fb%2F.drone.yml" {
				f, _ := os.Open("../testdata/gitlab/a_b_.drone.yml.json")
				_, _ = io.Copy(w, f)
			}
			if r.URL.RawPath == "/api/v4/projects/1234/repository/files/afolder%2F.drone.yml" && r.FormValue("ref") == "8ecad91991d5da985a2a8dd97cc19029dc1c2899" {
				f, _ := os.Open("../testdata/gitlab/afolder_.drone.yml.json")
				_, _ = io.Copy(w, f)
			}
		})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		logrus.Errorf("Url not found: %s", r.URL)
	})
	return mux
}
