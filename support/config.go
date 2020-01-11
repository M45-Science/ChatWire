package support

import (
	"errors"
	"os"
	"strings"
)

// Config is a config interface.
var Config config

type config struct {
	DiscordToken      string
	FactorioChannelID string
	Executable        string
	LaunchParameters  []string
	AdminIDs          []string
	Prefix            string
	ModListLocation   string
	GameName          string
	ChannelName       string
	DBFile            string
	MaxFile           string
	ChannelPos        string
	MapPreset         string
	MapGenExec        string
	PreviewArgs       string
	PreviewPath       string
	NewMapPath        string
	PreviewRes        string
}

func (conf *config) LoadEnv() {
	if _, err := os.Stat(".env"); os.IsNotExist(err) {
		Log("Environment file not found, cannot continue!")
		Error := errors.New("Failed to load environment file")
		ErrorLog(Error)
		Exit(1)
	}

	Config = config{
		DiscordToken:      os.Getenv("DiscordToken"),
		FactorioChannelID: os.Getenv("FactorioChannelID"),
		LaunchParameters:  strings.Split(os.Getenv("LaunchParameters"), " "),
		Executable:        os.Getenv("Executable"),
		AdminIDs:          strings.Split(os.Getenv("AdminIDs"), " "),
		Prefix:            os.Getenv("Prefix"),
		ModListLocation:   os.Getenv("ModListLocation"),
		GameName:          os.Getenv("GameName"),
		ChannelName:       os.Getenv("ChannelName"),
		DBFile:            os.Getenv("DBFile"),
		MaxFile:           os.Getenv("MaxFile"),
		ChannelPos:        os.Getenv("ChannelPos"),
		MapPreset:         os.Getenv("MapPreset"),
		MapGenExec:        os.Getenv("MapGenExec"),
		PreviewArgs:       os.Getenv("PreviewArgs"),
		PreviewPath:       os.Getenv("PreviewPath"),
		NewMapPath:        os.Getenv("NewMapPath"),
		PreviewRes:        os.Getenv("PreviewRes"),
	}
	//Log(Config.AdminIDs[0])

}
