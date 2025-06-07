package support

import (
	"reflect"

	"github.com/bwmarrin/discordgo"

	"ChatWire/cfg"
)

// updateRoleCache checks if the discord role matches one of the configured role
// names and updates the cached ID if needed. It returns true when an update was
// made.
func updateRoleCache(r *discordgo.Role, roleMap map[string]*string) bool {
	if r == nil {
		return false
	}

	if idPtr, ok := roleMap[r.Name]; ok {
		if r.ID != "" && idPtr != nil && *idPtr != r.ID {
			*idPtr = r.ID
			return true
		}
	}

	return false
}

// buildRoleMap uses reflection to gather all configured role names from
// cfg.Global.Discord.Roles. It returns a map keyed by the role name with a
// pointer to the corresponding cached ID field.
func buildRoleMap() map[string]*string {
	rm := make(map[string]*string)

	rolesVal := reflect.ValueOf(&cfg.Global.Discord.Roles).Elem()
	cacheVal := reflect.ValueOf(&cfg.Global.Discord.Roles.RoleCache).Elem()

	for i := 0; i < rolesVal.NumField(); i++ {
		field := rolesVal.Type().Field(i)

		// Skip non-string fields such as Comment and RoleCache
		if field.Type.Kind() != reflect.String {
			continue
		}

		roleName := rolesVal.Field(i).String()
		if roleName == "" {
			continue
		}

		cacheField := cacheVal.FieldByName(field.Name)
		if cacheField.IsValid() {
			rm[roleName] = cacheField.Addr().Interface().(*string)
		}
	}

	return rm
}
