package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
)

const (
	grafanaUrl = "https://pantheon.grafana.net/api"
)

var foldersToIgnore = [...]string{
	"scratch",
	"dev",
}

type dashboard struct {
	ID    int      `json:"id"`
	UID   string   `json:"uid"`
	Title string   `json:"title"`
	Uri   string   `json:"uri"`
	Url   string   `json:"url"`
	Slug  string   `json:"slug"`
	Type  string   `json:"type"`
	Tags  []string `json:"tags"`
	//IsStarred bool `json:"isstarred"`
	FolderId    int    `json:"folderid"`
	FolderUid   string `json:"folderuid"`
	FolderTitle string `json:"foldertitle"`
	FolderUrl   string `json:"folderurl"`
	//SortMeta int `json:"sortmeta"`
	Description string `json:"description"`
}

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if err := run(log); err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	// ------------------------------------------------------------------------
	// ENV Variables
	// ------------------------------------------------------------------------
	log.Debug("loading ENV variables")
	requiredEnvs := []string{"GRAFANA_TOKEN"}
	errs := ""
	for _, env := range requiredEnvs {
		if os.Getenv(env) == "" {
			errs += fmt.Sprintf("%s, ", env)
		}
	}

	if errs != "" {
		return errors.New(fmt.Sprintf("missing env variables: %s", strings.Trim(errs, " ,")))
	}

	// ------------------------------------------------------------------------
	// CLI Args
	// ------------------------------------------------------------------------
	argsWithoutProg := os.Args[1:]

	if len(argsWithoutProg) < 1 {
		log.Debug("no arg specified setting endpoint to 'user'")
		argsWithoutProg = append(argsWithoutProg, "user")
	}
	// ------------------------------------------------------------------------
	// API Magics
	// TODO:
	// 	1. get all dashboards
	// 	2. filter out any dev/scratch/whatever (make a list)
	// 	3. loop through and pull out needed pieces (title, description, tags?)
	// 	4. return all
	// ------------------------------------------------------------------------
	switch argsWithoutProg[0] {
	case "user":
		log.Debug("performing GET 'user' request")
        body, err := getUser()
        if err != nil {
            return fmt.Errorf("getUser: %v", err)
        }
        fmt.Println(body)

	case "search":
		queryParam := "%"
		// Default to shwoing ALL dashboards, otherwise do a fuzzy search
		if len(argsWithoutProg) > 1 {
			queryParam = strings.TrimSpace(argsWithoutProg[1])
		}
		allDashboards, err := getAllDashboards(queryParam)
		if err != nil {
			return fmt.Errorf("getAllDashboards: %v", err)
		}
		//fmt.Println(allDashboards[0].Title)

		pd, _ := parseDashboards(allDashboards)
		j, _ := json.MarshalIndent(pd, "", " ")
		fmt.Println(string(j))

	default:
		body, err := getDashboard(argsWithoutProg[0])
		if err != nil {
			return fmt.Errorf("getDashboards [%s]: %v", argsWithoutProg, err)
		}
		fmt.Println(body)
	}

	return nil
}

func getUser() (string, error) {
	req, err := _getReq("user")
	if err != nil {
		return "", fmt.Errorf("getUser request: %v", err)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("do request: %v", err)
	}
	defer res.Body.Close()
	body, readErr := io.ReadAll(res.Body)
	if readErr != nil {
		return "", fmt.Errorf("read body: %v", err)
	}

	return string(body), nil
}

func parseDashboards(ad []dashboard) ([]dashboard, error) {
	var filteredDashboards []dashboard
	for _, d := range ad {
		// skip folders
		if d.Type == "dash-folder" {
			continue
		}

		fmt.Println(d.Title)
		singleDashboard, _ := getDashboard(d.UID)
		desc := getDescription(singleDashboard)

		d.Description = desc
		filteredDashboards = append(filteredDashboards, d)
	}

	return filteredDashboards, nil
}

func getDashboard(dashboardUID string) (string, error) {
	// TODO: DON'T Fix - no one is going to use this in prod ... right?  RIGHT!?
	// CLI Injection RISK!  YEA!!
	endpoint := fmt.Sprintf("dashboards/uid/%s", dashboardUID)
	// Inline debugging?  Damn skippy!
	//fmt.Println(url)
	req, err := _getReq(endpoint)
	if err != nil {
		return "", fmt.Errorf("getDashboard request: %v", err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("do request: %v", err)
	}
	defer res.Body.Close()
	body, readErr := io.ReadAll(res.Body)
	if readErr != nil {
		return "", fmt.Errorf("read body: %v", err)
	}

	return string(body), nil
}

// Setup the GET request to the grafana api
func _getReq(endpoint string) (*http.Request, error) {
	url := fmt.Sprintf("%s/%s", grafanaUrl, endpoint)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("get request: %v", err)
	}
	bearer := "Bearer " + os.Getenv("GRAFANA_TOKEN")
	req.Header.Add("Authorization", bearer)

	return req, err
}

// parseDashboard - technically, this is only pulling out a description for now
func getDescription(body string) string {
	res := make(map[string]interface{})
	json.Unmarshal([]byte(body), &res)

	// mUahhaha - this is fantastically gross looking
	dashboard := res["dashboard"]
	desc := dashboard.(map[string]interface{})["description"]
	if desc == nil {
		return ""
	}

	return desc.(string)
}

func getAllDashboards(queryParam string) ([]dashboard, error) {

	endpoint := fmt.Sprintf("/search?query=%s", queryParam)
	// Inline debugging?  Damn skippy!
	//fmt.Println(url)
    req, err := _getReq(endpoint)
	if err != nil {
		return nil, fmt.Errorf("getAllDashboards request: %v", err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %v", err)
	}
	defer res.Body.Close()
	body, readErr := io.ReadAll(res.Body)
	if readErr != nil {
		return nil, fmt.Errorf("read body: %v", err)
	}

	var allDashboards []dashboard
	json.Unmarshal(body, &allDashboards)

	return allDashboards, nil
}
