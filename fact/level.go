package fact

//Player level names
type LevelNameData struct {
	Name  string
	Level int
}

var Levels = []LevelNameData{
	{
		Name:  "Deleted",
		Level: -255,
	},
	{
		Name:  "Banned",
		Level: -1,
	},
	{
		Name:  "New",
		Level: 0,
	},
	{
		Name:  "Member",
		Level: 1,
	},
	{
		Name:  "Regular",
		Level: 2,
	},
	{
		Name:  "Moderator",
		Level: 255,
	},
}
