package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/catallo/misterclaw/pkg/mister"
)

const Version = "0.1.0"

// GitHub API types

type ghRepo struct {
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Archived bool   `json:"archived"`
	Fork     bool   `json:"fork"`
}

type ghContent struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	DownloadURL string `json:"download_url"`
	Type        string `json:"type"`
	SHA         string `json:"sha"`
}

// Repos to skip — these are infrastructure, not FPGA cores.
var skipRepos = map[string]bool{
	"Main_MiSTer":         true,
	"Distribution_MiSTer": true,
	"Downloader_MiSTer":   true,
	"Menu_MiSTer":         true,
	"Hardware_MiSTer":     true,
	"Wiki_MiSTer":         true,
	"Setup_MiSTer":        true,
	"Updater_MiSTer":      true,
	"Filters_MiSTer":      true,
	"Presets_MiSTer":      true,
	"Scripts_MiSTer":      true,
	"Template_MiSTer":     true,
	"MkDocs_MiSTer":       true,
	"mr-fusion":           true,
	"Linux-Kernel_MiSTer": true,
	"Quartus_Compile":     true,
	"SD-Installer-Win64_MiSTer": true,
	"Fonts_MiSTer":              true,
	"Gamecontrollerdb_MiSTer":   true,
	"WebMenu_MiSTer":            true,
}

func main() {
	output := flag.String("output", "confstr_db.json", "Output JSON file path")
	tokenFile := flag.String("token", "", "Path to file containing GitHub token")
	cacheDir := flag.String("cache-dir", "", "Directory to cache downloaded .sv files")
	versionFlag := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("confstr-update v%s\n", Version)
		os.Exit(0)
	}

	log.SetOutput(os.Stderr)
	log.SetFlags(log.Ltime | log.Lmsgprefix)
	log.SetPrefix("[confstr-update] ")

	token := ""
	if *tokenFile != "" {
		data, err := os.ReadFile(*tokenFile)
		if err != nil {
			log.Fatalf("reading token file: %v", err)
		}
		token = strings.TrimSpace(string(data))
	}

	if *cacheDir != "" {
		if err := os.MkdirAll(*cacheDir, 0755); err != nil {
			log.Fatalf("creating cache dir: %v", err)
		}
	}

	client := &http.Client{Timeout: 30 * time.Second}

	var allCores []mister.CoreOSD

	// Scan MiSTer-devel org
	log.Println("scanning MiSTer-devel org...")
	repos := listOrgRepos(client, token, "MiSTer-devel")
	log.Printf("found %d repos in MiSTer-devel", len(repos))

	for _, repo := range repos {
		if skipRepos[repo.Name] || repo.Archived {
			continue
		}
		if !strings.HasSuffix(repo.Name, "_MiSTer") {
			continue
		}
		core := processRepo(client, token, *cacheDir, repo)
		if core != nil {
			allCores = append(allCores, *core)
			log.Printf("  [OK] %s -> %s (%d menu items)", repo.FullName, core.CoreName, len(core.Menu))
		}
	}

	// Scan jotego org
	log.Println("scanning jotego org...")
	jtRepos := listOrgRepos(client, token, "jotego")
	log.Printf("found %d repos in jotego", len(jtRepos))

	for _, repo := range jtRepos {
		if repo.Archived || repo.Fork {
			continue
		}
		if !strings.HasPrefix(repo.Name, "jt") {
			continue
		}
		// Skip non-core repos
		if repo.Name == "jtframe" || repo.Name == "jtbin" || repo.Name == "jtpremium" || repo.Name == "jtutil" {
			continue
		}
		core := processRepo(client, token, *cacheDir, repo)
		if core != nil {
			allCores = append(allCores, *core)
			log.Printf("  [OK] %s -> %s (%d menu items)", repo.FullName, core.CoreName, len(core.Menu))
		}
	}

	db := mister.ConfStrDB{
		Version: time.Now().UTC().Format("2006-01-02"),
		Cores:   allCores,
	}

	data, err := json.MarshalIndent(db, "", "  ")
	if err != nil {
		log.Fatalf("marshaling JSON: %v", err)
	}

	if err := os.WriteFile(*output, data, 0644); err != nil {
		log.Fatalf("writing output: %v", err)
	}

	log.Printf("wrote %d cores to %s", len(allCores), *output)
}

// listOrgRepos lists all repos in a GitHub org (paginated).
func listOrgRepos(client *http.Client, token, org string) []ghRepo {
	var all []ghRepo
	page := 1
	for {
		url := fmt.Sprintf("https://api.github.com/orgs/%s/repos?per_page=100&page=%d&type=public", org, page)
		var repos []ghRepo
		if err := ghGet(client, token, url, &repos); err != nil {
			log.Printf("listing repos page %d: %v", page, err)
			break
		}
		if len(repos) == 0 {
			break
		}
		all = append(all, repos...)
		page++
	}
	return all
}

// processRepo tries to find and parse CONF_STR from a repo's top-level .sv files.
func processRepo(client *http.Client, token, cacheDir string, repo ghRepo) *mister.CoreOSD {
	// List top-level files
	url := fmt.Sprintf("https://api.github.com/repos/%s/contents/", repo.FullName)
	var contents []ghContent
	if err := ghGet(client, token, url, &contents); err != nil {
		return nil
	}

	// Find .sv files at root level
	var svFiles []ghContent
	for _, c := range contents {
		if c.Type == "file" && strings.HasSuffix(strings.ToLower(c.Name), ".sv") {
			svFiles = append(svFiles, c)
		}
	}

	if len(svFiles) == 0 {
		// Try hdl/ or rtl/ subdirectory (common in jotego cores)
		for _, subdir := range []string{"hdl", "rtl", "src"} {
			subURL := fmt.Sprintf("https://api.github.com/repos/%s/contents/%s", repo.FullName, subdir)
			var subContents []ghContent
			if err := ghGet(client, token, subURL, &subContents); err != nil {
				continue
			}
			for _, c := range subContents {
				if c.Type == "file" && strings.HasSuffix(strings.ToLower(c.Name), ".sv") {
					svFiles = append(svFiles, c)
				}
			}
		}
	}

	// Try each .sv file for CONF_STR
	for _, svFile := range svFiles {
		source := downloadFile(client, token, cacheDir, repo.FullName, svFile)
		if source == "" {
			continue
		}
		raw := mister.ExtractConfStr(source)
		if raw == "" {
			continue
		}

		coreName := mister.ExtractCoreName(raw)
		if coreName == "" {
			coreName = mister.RepoToCoreName(repo.Name)
		}

		menu := mister.ParseConfStr(raw)
		return &mister.CoreOSD{
			CoreName:   coreName,
			Repo:       repo.FullName,
			ConfStrRaw: raw,
			Menu:       menu,
		}
	}

	// Also check .v files (plain Verilog)
	var vFiles []ghContent
	for _, c := range contents {
		if c.Type == "file" && strings.HasSuffix(strings.ToLower(c.Name), ".v") {
			vFiles = append(vFiles, c)
		}
	}
	for _, vFile := range vFiles {
		source := downloadFile(client, token, cacheDir, repo.FullName, vFile)
		if source == "" {
			continue
		}
		raw := mister.ExtractConfStr(source)
		if raw == "" {
			continue
		}

		coreName := mister.ExtractCoreName(raw)
		if coreName == "" {
			coreName = mister.RepoToCoreName(repo.Name)
		}

		menu := mister.ParseConfStr(raw)
		return &mister.CoreOSD{
			CoreName:   coreName,
			Repo:       repo.FullName,
			ConfStrRaw: raw,
			Menu:       menu,
		}
	}

	return nil
}

// downloadFile downloads a file, using cache if available.
func downloadFile(client *http.Client, token, cacheDir, repoFullName string, file ghContent) string {
	// Check cache
	if cacheDir != "" {
		cachePath := filepath.Join(cacheDir, strings.ReplaceAll(repoFullName, "/", "_")+"_"+file.Name)
		shaPath := cachePath + ".sha"

		// If cached SHA matches, use cached file
		if cachedSHA, err := os.ReadFile(shaPath); err == nil && strings.TrimSpace(string(cachedSHA)) == file.SHA {
			if data, err := os.ReadFile(cachePath); err == nil {
				return string(data)
			}
		}

		// Download and cache
		source := downloadRaw(client, token, file.DownloadURL)
		if source != "" {
			os.WriteFile(cachePath, []byte(source), 0644)
			os.WriteFile(shaPath, []byte(file.SHA), 0644)
		}
		return source
	}

	return downloadRaw(client, token, file.DownloadURL)
}

// downloadRaw downloads a file by URL.
func downloadRaw(client *http.Client, token, url string) string {
	if url == "" {
		return ""
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return ""
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Accept", "application/vnd.github.v3.raw")

	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return ""
	}

	// Limit to 2MB to avoid huge files
	data, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return ""
	}
	return string(data)
}

// ghGet makes an authenticated GitHub API GET request and decodes JSON.
func ghGet(client *http.Client, token, url string, result interface{}) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 403 {
		return fmt.Errorf("rate limited (403) — provide a --token for higher limits")
	}
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("GET %s: status %d: %s", url, resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(result)
}
