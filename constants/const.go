package constants

import "time"

const Version = "513-01072022-0539"
const Unknown = "Unknown"

//Config files
const CWGlobalConfig = "../cw-global-config.json"
const CWLocalConfig = "cw-local-config.json"
const WhitelistName = "server-whitelist.json"
const ServSettingsName = "server-settings.json"
const ModsQueueFolder = "mods-queue"
const ModsFolder = "mods"
const RoleListFile = "../RoleList.dat"

const PauseThresh = 5               //Number of repeated time reports before we assume server is paused
const ErrorDelayShutdown = 30       //If we close on error, sleep this long before exiting
const RestartLimitMinutes = 5       //If cw.lock is newer than this, sleep
const RestartLimitSleepMinutes = 15 //cw.lock is new, sleep this long then exit.

//Max results to return
const WhoisResults = 20
const AdminWhoisResults = 40

//Maximum time to wait for factorio update download
const FactorioUpdateCheckLimit = 15 * time.Minute

//Maximum time before giving up on patching
const FactorioUpdateProcessLimit = 10 * time.Minute

//Maximum time before giving up on checking zipfile integrity
const ZipIntegrityLimit = 5 * time.Minute

//Maximum time to wait for Factorio to close
const MaxFactorioCloseWait = 60

//Maximum number of autosaves to display in the 'rewind list' command
const MaxRewindResults = 40

//How often to check if Factorio server is alive
const WatchdogInterval = time.Second

//Throttle chat, 1.5 seconds per message.
const CMSRate = 500 * time.Millisecond
const CMSRestTime = 1000 * time.Millisecond
const CMSPollRate = 100 * time.Millisecond

//Used for chat colors in-game
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

//Factorio map preset names
var MapTypes = []string{"custom", "default", "rich-resources", "marathon", "death-world", "death-world-marathon", "rail-world", "ribbon-world", "island"}
