package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"

	stat "github.com/gonum/stat"
)

var cache string

func init() {
	home := os.Getenv("HOME")

	if len(home) == 0 {
		panic("HOME env not set")
	}

	cache = filepath.Join(home, ".testchart")

	if ok, _ := exists(cache); ok == false {
		os.MkdirAll(cache, os.ModePerm)
	}
}

// FilterFunc used to filter TestStats
type FilterFunc func(ts TestStats) bool

// filter removes any TestStats which does not return true for every FilterFunc supplied
func filter(teststats []TestStats, filterFuncs ...FilterFunc) []TestStats {
	var tmp []TestStats

	for _, t := range teststats {
		for _, f := range filterFuncs {
			if ok := f(t); ok == true {
				tmp = append(tmp, t)
			}
		}
	}

	return tmp
}

func main() {
	var project string
	var branch string
	var output string
	var start int
	var end int
	var renderHtml bool

	flag.StringVar(&project, "project", "", "Project to scan (required)")
	flag.StringVar(&branch, "branch", "", "Branch to scan (required)")
	flag.StringVar(&output, "output", "", "File to write output too (index.html|index.json)")
	flag.BoolVar(&renderHtml, "render", true, "Render output as html (true)")
	flag.IntVar(&start, "start", 0, "Starting job run (required)")
	flag.IntVar(&end, "end", 0, "Ending job run (required)")
	flag.Parse()

	if len(project) == 0 {
		fmt.Println("No project set")
		os.Exit(1)
	}

	if len(branch) == 0 {
		fmt.Println("No branch set")
		os.Exit(1)
	}

	if end == 0 || start == 0 {
		fmt.Printf("start or end must not be 0")
		os.Exit(1)
	}

	if end < start {
		fmt.Printf("end %d is less than start %d\n", end, start)
		os.Exit(1)
	}

	if len(output) == 0 {
		if renderHtml {
			output = "index.html"
		} else {
			output = "index.json"
		}
	}

	if err := fetch(project, branch, start, end); err != nil {
		panic(err)
	}

	stats, err := analyze(project, branch, start, end)
	if err != nil {
		panic(err)
	}

	// Filter out anything we don't want to show
	stats = filter(stats, []FilterFunc{
		func(ts TestStats) bool {
			return ts.StdDev > 1
		},
	}...)

	t, err := template.New("t").Parse(htmlTemplate)
	if err != nil {
		panic(err)
	}

	f, err := os.Create(output)
	if err != nil {
		panic(err)
	}

	defer f.Close()

	if renderHtml {
		if err := render(stats, project, branch, start, end, t, f); err != nil {
			panic(err)
		}
	} else {
		enc := json.NewEncoder(f)
		if err := enc.Encode(stats); err != nil {
			panic(err)
		}
	}

	fmt.Println("Done!")
}

// JenkinsTestResult a single test result from jenkins
type JenkinsTestResult struct {
	Name string `json:"name"`
}

// TestStats collection of information about a test over a given set of runs
type TestStats struct {
	Run      []int
	Name     string
	FailedOn map[string]int
	Count    int
	StdDev   float64
}

func analyze(project, branch string, start, end int) ([]TestStats, error) {
	cacheDir := filepath.Join(cache, project, branch)

	reTestName, err := regexp.Compile(`Tests \/ ([a-z]*) - (\d+.\d+.\d+) - test \/ (.*)`)
	if err != nil {
		return []TestStats{}, err
	}

	results := make(map[string]TestStats, end-start+1)

	for run := start; run <= end; run++ {
		output := filepath.Join(cacheDir, fmt.Sprintf("run-%d.json", run))

		f, err := os.Open(output)
		if err != nil {
			return []TestStats{}, err
		}

		defer f.Close()

		dec := json.NewDecoder(f)

		jenkinsTestList := []JenkinsTestResult{}
		dec.Decode(&jenkinsTestList)

		for _, test := range jenkinsTestList {
			res := reTestName.FindStringSubmatch(test.Name)
			platform := res[1]
			version := res[2]
			name := res[3]

			result := results[name]

			// First time we encounter the test it will be nil
			if result.FailedOn == nil {
				result.FailedOn = make(map[string]int)
			}

			platform_version := fmt.Sprintf("%s %s", platform, version)

			result.Name = name
			result.Count += 1
			result.FailedOn[platform_version] += 1
			result.Run = append(result.Run, run)

			results[name] = result
		}
	}

	// Add StdDev for of the test run numbers. The large the value, the further
	// spread out the text failures are.
	var stats []TestStats
	for _, v := range results {
		var in []float64

		v.Run = unique(v.Run)

		for _, r := range v.Run {
			in = append(in, float64(r))
		}

		sd := stat.StdDev(in, nil)
		if !math.IsNaN(sd) {
			v.StdDev = sd
		}

		stats = append(stats, v)
	}

	// Sort the list by name, and then failure count

	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Name < stats[j].Name
	})

	sort.SliceStable(stats, func(i, j int) bool {
		return stats[i].Count > stats[j].Count
	})

	return stats, nil
}

type Row struct {
	Name   string
	Values []bool
}

type TemplateData struct {
	Title   string
	Headers []int
	Rows    []Row
}

func render(stats []TestStats, project, branch string, start, end int, t *template.Template, out io.Writer) error {

	var testRuns []int
	for i := start; i <= end; i++ {
		testRuns = append(testRuns, i)
	}

	var rows []Row
	for _, v := range stats {
		r := Row{}
		r.Name = v.Name

		set := make(map[int]struct{})
		for _, v := range v.Run {
			set[v] = struct{}{}
		}

		var values []bool
		for _, run := range testRuns {
			if _, ok := set[run]; ok {
				values = append(values, true)
			} else {
				values = append(values, false)
			}
		}

		r.Values = values

		rows = append(rows, r)
	}

	d := TemplateData{
		Title:   fmt.Sprintf("%s - %s", project, branch),
		Headers: testRuns,
		Rows:    rows,
	}

	return t.Execute(out, d)
}
func unique(arr []int) []int {
	set := make(map[int]struct{})

	var narr []int

	for _, v := range arr {
		set[v] = struct{}{}
	}

	for k := range set {
		narr = append(narr, k)
	}

	return narr
}

func fetch(project, branch string, start, end int) error {
	cacheDir := filepath.Join(cache, project, branch)
	if ok, _ := exists(cacheDir); ok == false {
		os.MkdirAll(cacheDir, os.ModePerm)
	}

	for run := start; run <= end; run++ {
		output := filepath.Join(cacheDir, fmt.Sprintf("run-%d.json", run))
		if ok, _ := exists(output); ok == true {
			fmt.Printf("Run %d for %s on %s already exists, skipping\n", run, project, branch)
			continue
		}

		fmt.Printf("Fetching run %d for %s on %s\n", run, project, branch)

		f, err := os.Create(output)
		if err != nil {
			return err
		}

		defer f.Close()

		uri := fmt.Sprintf("https://ci.ipfs.team/blue/rest/organizations/jenkins/pipelines/IPFS/pipelines/%s/branches/%s/runs/%d/tests/?status=FAILED&start=0&limit=101", project, branch, run)

		resp, err := http.Get(uri)
		if err != nil {
			return err
		}

		defer resp.Body.Close()

		io.Copy(f, resp.Body)

		time.Sleep(5 * time.Second)
	}

	return nil
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)

	if err == nil {
		return true, nil
	}

	if os.IsNotExist(err) {
		return false, nil
	}

	return true, err
}
