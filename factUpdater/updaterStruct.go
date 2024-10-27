package factUpdater

type checkInstallData struct {
	name string
	good bool
}

type InfoData struct {
	Version, Distro, Build string
	Xreleases              bool
	VersInt                versionInts
}

type getLatestData struct {
	Experimental getLatestPartData `json:"experimental"`
	Stable       getLatestPartData `json:"stable"`
}

type getLatestPartData struct {
	Headless    string `json:"headless"`
	HeadlessInt versionInts
}

type versionInts struct {
	A, B, C int
}
