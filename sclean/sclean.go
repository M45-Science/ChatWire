package sclean

import (
	"regexp"
	"strings"
)

//TruncateString Actually shorten strings
func TruncateString(str string, num int) string {
	bnoden := str
	if len(str) > num {
		if num > 3 {
			num -= 3
		}
		bnoden = str[0:num] + "..."
	}
	return bnoden
}

//Strip all but a-z
func StripControlAndSpecial(str string) string {
	b := make([]byte, len(str))
	var bl int
	for i := 0; i < len(str); i++ {
		c := str[i]
		if c >= 32 && c < 127 {
			b[bl] = c
			bl++
		}
	}
	return string(b[:bl])
}

//Sub specials with '?', sub newlines, returns and tabs with ' '
func SubControlAndSpecial(str string) string {
	b := make([]byte, len(str))
	var bl int
	for i := 0; i < len(str); i++ {
		c := str[i]
		if c >= 32 && c < 127 {
			b[bl] = c
			bl++
		} else if c == '\n' || c == '\r' || c == '\t' {
			b[bl] = ' '
			bl++
		} else {
			b[bl] = '?'
			bl++
		}
	}
	return string(b[:bl])
}

//Strip lower ascii codes, sub newlines, returns and tabs with ' '
func StripControlAndSubSpecial(str string) string {
	b := make([]byte, len(str))
	var bl int
	for i := 0; i < len(str); i++ {
		c := str[i]
		if c == '\n' || c == '\r' || c == '\t' {
			b[bl] = ' '
			bl++
		} else if c >= 32 && c != 127 {
			b[bl] = c
			bl++
		}
	}
	return string(b[:bl])
}

//Strip lower ascii codes
func StripControl(str string) string {
	b := make([]byte, len(str))
	var bl int
	for i := 0; i < len(str); i++ {
		c := str[i]
		if c >= 32 && c != 127 {
			b[bl] = c
			bl++
		}
	}
	return string(b[:bl])
}

func RemoveDiscordMarkdown(input string) string {
	//Remove discord markdown
	regf := regexp.MustCompile(`\*+`)
	regg := regexp.MustCompile(`\~+`)
	regh := regexp.MustCompile(`\_+`)
	for regf.MatchString(input) || regg.MatchString(input) || regh.MatchString(input) {
		//Filter discord tags
		input = regf.ReplaceAllString(input, "")
		input = regg.ReplaceAllString(input, "")
		input = regh.ReplaceAllString(input, "")
	}

	return input
}

func RemoveFactorioTags(input string) string {
	//input = unidecode.Unidecode(input)

	//Remove factorio tags
	rega := regexp.MustCompile(`\[/[^][]+\]`) //remove close tags [/color]

	regc := regexp.MustCompile(`\[color=(.*?)\]`) //remove [color=*]
	regd := regexp.MustCompile(`\[font=(.*?)\]`)  //remove [font=*]

	rege := regexp.MustCompile(`\[(.*?)=(.*?)\]`) //Sub others

	input = strings.Replace(input, "\n", " ", -1)
	input = strings.Replace(input, "\r", " ", -1)
	input = strings.Replace(input, "\t", " ", -1)

	for regc.MatchString(input) || regd.MatchString(input) {
		//Remove colors/fonts
		input = regc.ReplaceAllString(input, "")
		input = regd.ReplaceAllString(input, "")
	}
	for rege.MatchString(input) {
		//Sub
		input = rege.ReplaceAllString(input, " [${1}: ${2}] ")
	}
	for rega.MatchString(input) {
		//Filter close tags
		input = rega.ReplaceAllString(input, "")
	}
	return input
}
