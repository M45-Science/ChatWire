package sclean

import (
	"fmt"
	"regexp"
	"strings"
)

func UnixSafeFilename(input string) string {
	input = StripControl(input)
	input = strings.ReplaceAll(input, " ", "_")
	input = strings.ReplaceAll(input, "..", "_")
	input = strings.ReplaceAll(input, ".", "_")
	input = UnixPreFilter(input)
	input = strings.TrimPrefix(input, ".")
	input = strings.TrimPrefix(input, ".")
	input = strings.TrimSuffix(input, ".sh")
	input = TruncateString(input, 64)
	return input
}

func AlphaOnly(str string) string {
	alphafilter, _ := regexp.Compile("[^a-zA-Z]+")
	str = alphafilter.ReplaceAllString(str, "")
	return str
}

func NumOnly(str string) string {
	alphafilter, _ := regexp.Compile("[^0-9]+")
	str = alphafilter.ReplaceAllString(str, "")
	return str
}

func AlphaNumOnly(str string) string {
	alphafilter, _ := regexp.Compile("[^a-zA-Z0-9]+")
	str = alphafilter.ReplaceAllString(str, "")
	return str
}

func UnixPreFilter(str string) string {
	alphafilter, _ := regexp.Compile("[^a-zA-Z0-9-_]+")
	str = alphafilter.ReplaceAllString(str, "")
	return str
}

//TruncateString Actually shorten strings
func TruncateStringEllipsis(str string, num int) string {
	bnoden := str
	if len(str) > num {
		if num > 3 {
			num -= 3
		}
		bnoden = str[0:num] + "..."
	}
	return bnoden
}

//TruncateString Actually shorten strings
func TruncateString(str string, num int) string {
	bnoden := str
	if len(str) > num {
		bnoden = str[0:num]
	}
	return bnoden
}

//Strip all but a-z
func StripControlAndSpecial(str string) string {
	b := make([]byte, len(str))
	var bl int
	for i := 0; i < len(str); i++ {
		c := fmt.Sprintf("%c", i)
		if c[0] >= 32 && c[0] < 127 {
			b[bl] = c[0]
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
		c := fmt.Sprintf("%c", i)
		if c[0] >= 32 && c[0] < 127 {
			b[bl] = c[0]
			bl++
		} else if c[0] == '\n' || c[0] == '\r' || c[0] == '\t' {
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
		c := fmt.Sprintf("%c", i)
		if c[0] == '\n' || c[0] == '\r' || c[0] == '\t' {
			b[bl] = ' '
			bl++
		} else if c[0] >= 32 && c[0] != 127 {
			b[bl] = c[0]
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
		c := fmt.Sprintf("%c", i)
		if c[0] >= 32 && c[0] != 127 {
			b[bl] = c[0]
			bl++
		}
	}
	return string(b[:bl])
}

func EscapeDiscordMarkdown(input string) string {
	input = strings.ReplaceAll(input, "\\", "\\\\")
	input = strings.ReplaceAll(input, "_", "\\_")
	input = strings.ReplaceAll(input, "*", "\\*")
	input = strings.ReplaceAll(input, "~", "\\~")
	input = strings.ReplaceAll(input, "`", "\\`")
	input = strings.ReplaceAll(input, "|", "\\|")
	return input
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
		input = strings.ReplaceAll(input, "`", "")
	}

	return input
}

func RemoveFactorioTags(input string) string {
	//input = unidecode.Unidecode(input)

	//Remove factorio tags
	rega := regexp.MustCompile(`\[/[^][]+\]`) //remove close tags [/color]

	regc := regexp.MustCompile(`\[color=(.*?)\]`) //remove [color=*]
	regd := regexp.MustCompile(`\[font=(.*?)\]`)  //remove [font=*]

	input = strings.Replace(input, "\n", " ", -1)
	input = strings.Replace(input, "\r", " ", -1)
	input = strings.Replace(input, "\t", " ", -1)

	for regc.MatchString(input) || regd.MatchString(input) {
		//Remove colors/fonts
		input = regc.ReplaceAllString(input, "")
		input = regd.ReplaceAllString(input, "")
	}
	for rega.MatchString(input) {
		//Filter close tags
		input = rega.ReplaceAllString(input, "")
	}
	return input
}
