package modupdate

import "time"

const (
	EO_ERROR = iota
	EO_LESS
	EO_LESSEQ
	EO_EQUAL
	EO_GREATEREQ
	EO_GREATER
)

type DownloadData struct {
	Name, Title, OldFilename string
	Data                     ModReleases
	Complete                 bool
	doDownload               bool
}

type ModListData struct {
	Mods []ModData
}
type ModData struct {
	Name    string
	Enabled bool
}

type IntVersion struct {
	parts [3]int
}

type ModPortalFullData struct {
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
}

type ModZipInfo struct {
	Name                string   `json:"name"`
	Author              string   `json:"author"`
	Version             string   `json:"version"`
	Title               string   `json:"title"`
	Description         string   `json:"description"`
	Contact             string   `json:"contact"`
	FactorioVersion     string   `json:"factorio_version"`
	Dependencies        []string `json:"dependencies"`
	SpaceTravelRequired bool     `json:"space_travel_required"`

	OldFilename string `json:"-"`
	Enabled     bool   `json:"-"`
}
