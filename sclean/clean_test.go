package sclean

import (
	"fmt"
	"testing"
)

func TestUnixSafeFilename(t *testing.T) {

	expect := "_-_0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ_abcdefghijklmnopqrstuvwx"
	filtered := UnixSafeFilename(allChars(1024))

	if filtered != expect {
		t.Error("UnixSafeFilename failed: got: " + filtered + " want: " + expect)
	} else {
		println("UnixSafeFilename passed")
	}
}

func TestAlphaOnly(t *testing.T) {

	expect := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	filtered := AlphaOnly(allChars(1024))
	if filtered != expect {
		t.Error("AlphaOnly failed: got: " + filtered + " want: " + expect)
	} else {
		println("AlphaOnly passed")
	}
}

func TestNumOnly(t *testing.T) {

	expect := "0123456789"
	filtered := NumOnly(allChars(1024))
	if filtered != expect {
		t.Error("NumOnly failed: got: " + filtered + " want: " + expect)
	} else {
		println("NumOnly passed")
	}
}

func TestAlphaNumOnly(t *testing.T) {

	expect := "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	filtered := AlphaNumOnly(allChars(1024))
	if filtered != expect {
		t.Error("AlphaNumOnly failed: got: " + filtered + " want: " + expect)
	} else {
		println("AlphaNumOnly passed")
	}
}

func TestUnixPreFilter(t *testing.T) {

	expect := "-0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ_abcdefghijklmnopqrstuvwxyz"
	filtered := UnixPreFilter(allChars(1024))

	if filtered != expect {
		t.Error("UnixPreFilter failed: got: " + filtered + " want: " + expect)
	} else {
		println("UnixPreFilter passed")
	}
}

func TestTruncateStringEllipsis(t *testing.T) {

	expect := "The quick brown fox jumps over the lazy dog The rain in spain..."
	filtered := TruncateStringEllipsis(usePhrase(), 64)

	if filtered != expect || len(filtered) != 64 {
		t.Error("TruncateStringEllipsis failed: got: " + filtered + " want: " + expect)
	} else {
		println("TruncateStringEllipsis passed")
	}
}

func TestTruncateString(t *testing.T) {

	expect := "The quick brown fox jumps over the lazy dog The rain in spain fa"
	filtered := TruncateString(usePhrase(), 64)

	if filtered != expect || len(filtered) != 64 {
		t.Error("TruncateString failed: got: " + filtered + " want: " + expect)
	} else {
		println("TruncateString passed")
	}
}

func TestUnicodeCleanup(t *testing.T) {

	expectA := " !\"#$%&'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_`abcdefghijklmnopqrstuvwxyz{|}~"
	filteredA := UnicodeCleanup(allChars(128))

	expectB := "Falsches Üben, イロハニホヘト チリヌルヲ, חב, רה איך הקליטה,  жил бы цитрус 🇦🇶😃👻"
	filteredB := UnicodeCleanup(expectB)

	if filteredA != expectA {
		t.Error("UnicodeCleanup failed: got: " + filteredA + " want: " + expectA)
	} else if filteredB != expectB {
		t.Error("UnicodeCleanup failed: got: " + filteredB + " want: " + expectB)
	} else {
		println("UnicodeCleanup passed")
	}
}

func TestEscapeDiscordMarkdown(t *testing.T) {
	input := "This is *italics* and **bold** and ***bold italics***. Also, _underline_ and __underline bold__. And, __underline bold italics__ and _underline italics bold_. Also, `inline code` and ```preformatted code```."
	expect := "This is \\*italics\\* and \\*\\*bold\\*\\* and \\*\\*\\*bold italics\\*\\*\\*. Also, \\_underline\\_ and \\_\\_underline bold\\_\\_. And, \\_\\_underline bold italics\\_\\_ and \\_underline italics bold\\_. Also, \\`inline code\\` and \\`\\`\\`preformatted code\\`\\`\\`."
	filtered := EscapeDiscordMarkdown(input)

	if filtered != expect {
		t.Error("EscapeDiscordMarkdown failed: got: " + filtered + " want: " + expect)
	} else {
		println("EscapeDiscordMarkdown passed")
	}
}

func TestRemoveDiscordMarkdown(t *testing.T) {
	input := "This is *italics* and **bold** and ***bold italics***. Also, _underline_ and __underline bold__. And, __underline bold italics__ and _underline italics bold_. Also, `inline code` and ```preformatted code```."
	expect := "This is italics and bold and bold italics. Also, underline and underline bold. And, underline bold italics and underline italics bold. Also, inline code and preformatted code."
	filtered := RemoveDiscordMarkdown(input)

	if filtered != expect {
		t.Error("RemoveDiscordMarkdown failed: got: " + filtered + " want: " + expect)
	} else {
		println("RemoveDiscordMarkdown passed")
	}
}

func TestRemoveFactorioTags(t *testing.T) {
	input := "[color=red]Dude![/color] [font=default-bold]check this out![/font] [gps=0,0] is that a infinite chest of [item=iron-plate]iron plate?"
	expect := "Dude! check this out! [gps=0,0] is that a infinite chest of [item=iron-plate]iron plate?"
	filtered := RemoveFactorioTags(input)

	if filtered != expect {
		t.Error("RemoveFactorioTags failed: got: " + filtered + " want: " + expect)
	} else {
		println("RemoveFactorioTags passed")
	}
}

/* generate strings for tests */
func allChars(s int) string {
	testStr := ""
	for i := 1; i < s; i++ {
		t := fmt.Sprintf("%c", i)
		testStr = testStr + t
	}
	return testStr
}

func usePhrase() string {
	string := "The quick brown fox jumps over the lazy dog "
	string = string + "The rain in spain falls mainly on the plain "
	string = string + "lorum ipsum dolor sit amet "
	return string
}
