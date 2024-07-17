package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"text/template"

	"github.com/charmbracelet/glamour"
	"github.com/dtnp/go/grafana-api/pkg/simpletaxonomy"
)

const (
	grafanaUrl = "https://pantheon.grafana.net/api"
)

var foldersToIgnore = [...]string{
	"scratch",
	"depricated",
	"deprecated",
	"broken",
	"retired",
	"unknown",
	"delete",
	"test",
}

var tagsToIgnore = [...]string{
	"broken",
}

type dashboard struct {
	ID    int    `json:"id"`
	UID   string `json:"uid"`
	Title string `json:"title"`
	Owner string
	Uri   string `json:"uri"`
	Url   string `json:"url"`
	Slug  string `json:"slug"`
	//Type  string `json:"type"`
	L1   string
	L2   string
	Tags []string `json:"tags"`
	//IsStarred bool `json:"isstarred"`
	//FolderId    int    `json:"folderid"`
	//FolderUid   string `json:"folderuid"`
	FolderTitle string `json:"foldertitle"`
	FolderUrl   string `json:"folderurl"`
	//SortMeta int `json:"sortmeta"`
	Description string `json:"description"`
}

// Somewhere to hold L1 taxonomy and its children
type taxonomy struct {
	Name  string
	TaxL2 map[string]taxonomyL2
}

type taxonomyL2 struct {
	Name       string
	Dashboards []dashboard
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
	// Simplified Taxonomy Reference
	// ------------------------------------------------------------------------
	simpleTax, err := simpletaxonomy.ParseFile("./simplified-taxonomy.json")
	if err != nil {
		return err
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
		/*
			log.Debug("performing GET 'user' request")
			body, err := getUser()
			if err != nil {
				return fmt.Errorf("getUser: %v", err)
			}
			fmt.Println(body)
		*/
		break

	case "search":
		queryParam := "%"
		fmt.Println("Loading dashboards, please standby...")
		// Default to shwoing ALL dashboards, otherwise do a fuzzy search
		if len(argsWithoutProg) > 1 {
			queryParam = strings.TrimSpace(argsWithoutProg[1])
		}
		allDashboards, err := getAllDashboards(queryParam)
		if err != nil {
			return fmt.Errorf("getAllDashboards: %v", err)
		}

		pd, _ := parseDashboards(allDashboards)
		taxMap := mapDashboardTaxonomy(pd, simpleTax)
		printDashTaxMapCli(taxMap)
		break

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
		singleDashboard, _ := getDashboard(d.UID)
		desc := getDescription(singleDashboard)

		// Folks are going a little crazy with descriptions
		// killing newlines to help with formatting the output.
		desc = strings.ReplaceAll(desc, "\n", " ")

		d.Description = desc
		filteredDashboards = append(filteredDashboards, d)
	}

	return filteredDashboards, nil
}

// check our ignore list returns true if it should be ignored
func _folderIgnoreCheck(folderTitle string) bool {

	for _, f := range foldersToIgnore {
		if strings.Contains(folderTitle, f) {
			return true
		}
	}

	return false
}

// check if the dashboard is tagged with something from the ignore list
func _tagIgnoreCheck(tags []string) bool {

	for _, dashTag := range tags {
		for _, badTag := range tagsToIgnore {
			if strings.Compare(dashTag, badTag) == 0 {
				return true
			}
		}
	}

	return false
}

func mapDashboardTaxonomy(ad []dashboard, st simpletaxonomy.SimplifiedTaxonomy) map[string]taxonomy {
	//var mTax = make(map[string][]dashboard)
	var mTopTax = make(map[string]taxonomy)

	for i, d := range ad {
		// Filter out anything in an unwanted folder
		if _folderIgnoreCheck(d.FolderTitle) {
			continue
		}

		// Filter out anything with an unwanted tag
		if _tagIgnoreCheck(d.Tags) {
			continue
		}

		// Try and retrieve Simplified Taxonomy and owner tags
		tags := parseTags(d)
		level1 := tags["l1"]
		level2 := tags["l2"]

		// Stash the tags
		ad[i].Owner = tags["owner"]
		ad[i].L1 = level1
		ad[i].L2 = level2
		//mTax[level2] = append(mTax[level2], ad[i])

		// Do we have an entry for this level1 slug yet?
		_, ok := mTopTax[level1]
		if !ok {
			// Nope? Then make one.
			newTax := taxonomy{
				Name:  st.GetL1NameFromSlug(level1),
				TaxL2: make(map[string]taxonomyL2),
			}
			mTopTax[level1] = newTax
		}

		tax2, ok := mTopTax[level1].TaxL2[level2]
		if ok {
            // We already have an L2 for this slug

            // Since this is a map we have to modify a copy
            tax2.Dashboards = append(tax2.Dashboards, ad[i])
            //Then reassign the whole map entry.
            mTopTax[level1].TaxL2[level2] = tax2
		} else {
            // An L2 for this slug does not yet exist, create on
			newL2Tax := taxonomyL2{
				Name:       st.GetL2NameFromSlug(level2),
				Dashboards: make([]dashboard, 0),
			}
			mTopTax[level1].TaxL2[level2] = newL2Tax
		}
	}

	return mTopTax
}

func parseTags(d dashboard) map[string]string {
	var tags = make(map[string]string)

	// Loop through the tags and pull out l1 and l2
	for _, t := range d.Tags {
		if strings.Contains(t, "l1:") || strings.Contains(t, "l2:") {
			tags[t[:2]] = t[3:]
		}

		// Grab the owner while we're in here
		if strings.Contains(t, "owner:") {
			tags[t[:5]] = t[6:]
		}
	}

	return tags
}

func printDashTaxMapCli(dtm map[string]taxonomy) error {
	// Buffer to hold template output for later prettification
	var tmplOut bytes.Buffer

	// Define a template file to render the output.
	tmpl, err := template.ParseFiles("dashboards-md.tpl")
	if err != nil {
		return err
	}

	// Create an .md file to capture the raw markdown
	f, err := os.Create("./dashboard-taxonomy.md")
	if err != nil {
		return err
	}

	// Execute the template and write to our file
	err = tmpl.ExecuteTemplate(f, "dashboards-md.tpl", dtm)
	if err != nil {
		return err
	}
	// Don't forget!
	f.Close()

	// Execute the template with our built up map
	err = tmpl.ExecuteTemplate(&tmplOut, "dashboards-md.tpl", dtm)
	if err != nil {
		return err
	}

	// Customize our cli prettifier a little
	gr, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithPreservedNewLines(),
		glamour.WithWordWrap(120),
	)
	if err != nil {
		return err
	}

	// Make the cli output purdy
	out, err := gr.Render(tmplOut.String())
	if err != nil {
		return err
	}

	fmt.Print(out)
	return nil
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

	// "type=dash-db" excludes dash-folder
	endpoint := fmt.Sprintf("/search?query=%s&type=dash-db", queryParam)
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
