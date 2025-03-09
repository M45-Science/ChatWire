package sclean

import (
	"regexp"
	"strings"
	"unicode"
)

func UnicodeCleanup(input string) string {
	input = strings.ToValidUTF8(input, "")
	transformed := func(r rune) rune {

		if r != ' ' && (unicode.IsSpace(r) || !unicode.IsPrint(r)) {
			return -1
		} else {
			return r
		}
	}
	input = strings.Map(transformed, input)

	return input
}

func UnixSafeFilename(input string) string {
	input = UnicodeCleanup(input)
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

/* Shorten strings, end with ellipsis */
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

/* Shorten strings */
func TruncateString(str string, num int) string {
	bnoden := str
	if len(str) > num {
		bnoden = str[0:num]
	}
	return bnoden
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
	/* Remove Discord markdown */
	regf := regexp.MustCompile(`\*+`)
	regg := regexp.MustCompile(`\~+`)
	regh := regexp.MustCompile(`\_+`)
	for regf.MatchString(input) || regg.MatchString(input) || regh.MatchString(input) {
		/* Filter Discord tags */
		input = regf.ReplaceAllString(input, "")
		input = regg.ReplaceAllString(input, "")
		input = regh.ReplaceAllString(input, "")
		input = strings.ReplaceAll(input, "`", "")
	}

	return input
}

func RemoveFactorioTags(input string) string {
	/* input = unidecode.Unidecode(input) */

	/* Remove Factorio tags */
	/* remove close tags [/color] */
	rega := regexp.MustCompile(`\[/[^][]+\]`)
	/* remove [color=*] */
	regc := regexp.MustCompile(`\[color=(.*?)\]`)
	/* remove [font=*] */
	regd := regexp.MustCompile(`\[font=(.*?)\]`)

	input = strings.Replace(input, "\n", " ", -1)
	input = strings.Replace(input, "\r", " ", -1)
	input = strings.Replace(input, "\t", " ", -1)

	for regc.MatchString(input) || regd.MatchString(input) {
		/* Remove colors/fonts */
		input = regc.ReplaceAllString(input, "")
		input = regd.ReplaceAllString(input, "")
	}
	for rega.MatchString(input) {
		/* Filter close tags */
		input = rega.ReplaceAllString(input, "")
	}
	return input
}
