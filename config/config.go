package config

import (
	"os"
	"strings"
)

// Config is a config interface.
var Config config

type config struct {
	DiscordToken         string
	FactorioChannelID    string
	Executable           string
	LaunchParameters     []string
	AdminIDs             []string
	Prefix               string
	FactorioLocation     string
	GameName             string
	ChannelName          string
	DBFile               string
	MaxFile              string
	ChannelPos           string
	ServerLetter         string
	MapPreset            string
	PreviewArgs          string
	PreviewPath          string
	NewMapPath           string
	ConvertExec          string
	PreviewRes           string
	PreviewScale         string
	JpgQuality           string
	JpgScale             string
	SiteURL              string
	AutoStart            string
	ZipScript            string
	DoWhitelist          string
	GuildID              string
	RegularsRole         string
	MembersRole          string
	AdminsRole           string
	MapArchivePath       string
	AuxChannel           string
	UpdaterPath          string
	UpdaterCache         string
	UpdaterShell         string
	UpdateToExperimental string
	ZipBinary            string
	ModerationChannel    string
	MapGenJson           string
	CompressScript       string
}

func (conf *config) LoadEnv() {
	if _, err := os.Stat(".env"); os.IsNotExist(err) {
		println("Environment file not found, cannot continue!")
		println(err)
		os.Exit(1)
	}

	Config = config{
		DiscordToken:         os.Getenv("DiscordToken"),
		FactorioChannelID:    os.Getenv("FactorioChannelID"),
		LaunchParameters:     strings.Split(os.Getenv("LaunchParameters"), " "),
		Executable:           os.Getenv("Executable"),
		AdminIDs:             strings.Split(os.Getenv("AdminIDs"), " "),
		Prefix:               os.Getenv("Prefix"),
		FactorioLocation:     os.Getenv("FactorioLocation"),
		GameName:             os.Getenv("GameName"),
		ChannelName:          os.Getenv("ChannelName"),
		DBFile:               os.Getenv("DBFile"),
		MaxFile:              os.Getenv("MaxFile"),
		ChannelPos:           os.Getenv("ChannelPos"),
		ServerLetter:         os.Getenv("ServerLetter"),
		MapPreset:            os.Getenv("MapPreset"),
		PreviewArgs:          os.Getenv("PreviewArgs"),
		PreviewPath:          os.Getenv("PreviewPath"),
		NewMapPath:           os.Getenv("NewMapPath"),
		ConvertExec:          os.Getenv("ConvertExec"),
		PreviewRes:           os.Getenv("PreviewRes"),
		PreviewScale:         os.Getenv("PreviewScale"),
		JpgQuality:           os.Getenv("JpgQuality"),
		JpgScale:             os.Getenv("JpgScale"),
		SiteURL:              os.Getenv("SiteURL"),
		AutoStart:            os.Getenv("AutoStart"),
		ZipScript:            os.Getenv("ZipScript"),
		DoWhitelist:          os.Getenv("DoWhitelist"),
		GuildID:              os.Getenv("GuildID"),
		RegularsRole:         os.Getenv("RegularsRole"),
		MembersRole:          os.Getenv("MembersRole"),
		AdminsRole:           os.Getenv("AdminsRole"),
		MapArchivePath:       os.Getenv("MapArchivePath"),
		AuxChannel:           os.Getenv("AuxChannel"),
		UpdaterPath:          os.Getenv("UpdaterPath"),
		UpdaterCache:         os.Getenv("UpdaterCache"),
		UpdaterShell:         os.Getenv("UpdaterShell"),
		UpdateToExperimental: os.Getenv("UpdateToExperimental"),
		ZipBinary:            os.Getenv("ZipBinary"),
		ModerationChannel:    os.Getenv("ModerationChannel"),
		MapGenJson:           os.Getenv("MapGenJson"),
		CompressScript:       os.Getenv("CompressScript"),
	}

}
