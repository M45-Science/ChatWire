package sclean

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
