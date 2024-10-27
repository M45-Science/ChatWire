package factUpdater

type checkInstallData struct {
	name string
	good bool
}

type infoData struct {
	gVersion, gDistro, gBuild string
	xreleases, verboseMode    bool
	vInt                      versionInts
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
