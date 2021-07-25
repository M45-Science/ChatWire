package constants

import "time"

const Version = "471-7-24-2021-0623p"
const Unknown = "Unknown"

//Config files
const CWGlobalConfig = "../cw-global-config.json"
const CWLocalConfig = "cw-local-config.json"
const WhitelistName = "server-whitelist.json"

//Max player database size, pre-allocated
const MaxPlayers = 5000

//Number of repeated time reports before we assume server is paused
const PauseThresh = 5

//Minimum time between logout saves
const SaveThresh = 300
const WhoisResults = 20
const AdminWhoisResults = 40

//Max number of registration passwords at once
const MaxPasswords = 128

//Maximum time to wait for factorio update download
const FactorioUpdateCheckLimit = 15 * time.Minute

//Maximum time before giving up on patching
const FactorioUpdateProcessLimit = 10 * time.Minute

//Maximum time before giving up on checking zipfile integrity
const ZipIntegrityLimit = 5 * time.Minute

//Used for fuzzy delay/timers
const MinuteInMicro = 60000000
const SecondInMicro = 1000000
const TenthInMicro = 100000
const HundrethInMicro = 10000

const WatchdogInterval = time.Second

//Throttle to about 5 every 6 seconds
const CMSRate = 500 * time.Millisecond
const CMSRestTime = 6000 * time.Millisecond
const CMSPollRate = 100 * time.Millisecond

const NumColors = 17

var Colors = [...]struct {
	R float32
	G float32
	B float32
}{
	{1, 0, 0},          //RED
	{1, 0.25, 0},       //RED-ORANGE
	{1, 0.5, 0},        //ORANGE
	{1, 0.66, 0},       //ORANGE-YELLOW
	{1, 1, 0},          //YELLOW
	{0.66, 1, 0},       //YELLOW-GREEN
	{0, 1, 0},          //GREEN
	{0, 1, 0.66},       //GREEN-BLUE
	{0, 1, 1},          //CYAN
	{0, 0.66, 1},       //CYAN-BLUE
	{0, 0, 1},          //BLUE
	{0.33, 0, 1},       //BLUE-PURPLE
	{0.66, 0, 1},       //PURPLE
	{0.85, 0, 1},       //PURPLE-MAGENTA
	{1, 0, 1},          //MAGENTA
	{1, 0, 0.66},       //MAGENTA-RED
	{0.66, 0.66, 0.66}, //GRAY
}

var MapTypes = [...]string{"custom", "default", "rich-resources", "marathon", "death-world", "death-world-marathon", "rail-world", "ribbon-world", "island"}
