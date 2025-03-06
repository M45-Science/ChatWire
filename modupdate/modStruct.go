package modupdate

import "time"

type downloadData struct {
	Name,
	OldFilename, Filename,
	URL, Version string
	Ready bool
}

type modListData struct {
	Mods []modData
}
type modData struct {
	Name    string
	Enabled bool
}

type intVersion struct {
	parts [3]int
}

type modPortalFullData struct {
	Category          string        `json:"category"`
	Changelog         string        `json:"changelog"`
	CreatedAt         time.Time     `json:"created_at"`
	Description       string        `json:"description"`
	DownloadsCount    int           `json:"downloads_count"`
	GithubPath        string        `json:"github_path"`
	Homepage          string        `json:"homepage"`
	Images            []ModImages   `json:"images"`
	LastHighlightedAt string        `json:"last_highlighted_at"`
	License           ModLicense    `json:"license"`
	Name              string        `json:"name"`
	Owner             string        `json:"owner"`
	Releases          []ModReleases `json:"releases"`
	Score             float64       `json:"score"`
	SourceURL         string        `json:"source_url"`
	Summary           string        `json:"summary"`
	Tags              []string      `json:"tags"`
	Thumbnail         string        `json:"thumbnail"`
	Title             string        `json:"title"`
	UpdatedAt         time.Time     `json:"updated_at"`

	//Local
	oldFilename string `json:"-"`
}

type ModImages struct {
	ID        string `json:"id"`
	Thumbnail string `json:"thumbnail"`
	URL       string `json:"url"`
}

type ModLicense struct {
	Description string `json:"description"`
	ID          string `json:"id"`
	Name        string `json:"name"`
	Title       string `json:"title"`
	URL         string `json:"url"`
}

type ModInfoJSON struct {
	Dependencies    []string `json:"dependencies"`
	FactorioVersion string   `json:"factorio_version"`
}

type ModReleases struct {
	DownloadURL string      `json:"download_url"`
	FileName    string      `json:"file_name"`
	InfoJSON    ModInfoJSON `json:"info_json"`
	ReleasedAt  time.Time   `json:"released_at"`
	Sha1        string      `json:"sha1"`
	Version     string      `json:"version"`

	//Local
	oldFilename string `json:"-"`
}

type modZipInfo struct {
	Name                string   `json:"name"`
	Author              string   `json:"author"`
	Version             string   `json:"version"`
	Title               string   `json:"title"`
	Description         string   `json:"description"`
	Contact             string   `json:"contact"`
	FactorioVersion     string   `json:"factorio_version"`
	Dependencies        []string `json:"dependencies"`
	SpaceTravelRequired bool     `json:"space_travel_required"`

	//Local
	filename string `json:"-"`
}
