package commands

import "ChatWire/glob"

// Fill in blank choice values from names
func init() {
	for c, command := range cmds {
		for o, option := range command.AppCmd.Options {
			for ch, choice := range option.Choices {
				if choice.Value == nil {
					cmds[c].AppCmd.Options[o].Choices[ch].Value = filterName(choice.Name)
				}
			}
		}
	}
}

var cl []glob.CommandData

var cmds = buildCommands()

func buildCommands() []glob.CommandData {
	var out []glob.CommandData
	out = append(out, adminCommands()...)
	out = append(out, moderatorCommands()...)
	out = append(out, userCommands()...)
	return out
}
