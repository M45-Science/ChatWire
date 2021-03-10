package logs

import (
	"fmt"
	"strings"
	"time"

	"../cfg"
	"../glob"
	"../sclean"
)

func LogWithoutEcho(input string) {
	text := sclean.StripControl(input)

	t := time.Now()
	date := fmt.Sprintf("%02d-%02d-%04d_%02d-%02d-%02d", t.Month(), t.Day(), t.Year(), t.Hour(), t.Minute(), t.Second())

	_, err := glob.BotLogDesc.WriteString(fmt.Sprintf("%s: %s\n", date, text))
	if err != nil {
		fmt.Println(err)
	}
}

//Yuck, can't link package fact.. pasted.
func cms(channel string, text string) {

	//Split at newlines, so we can batch neatly
	lines := strings.Split(text, "\n")

	glob.CMSBufferLock.Lock()

	for _, line := range lines {

		if len(line) <= 2000 {
			var item glob.CMSBuf
			item.Channel = channel
			item.Text = line

			glob.CMSBuffer = append(glob.CMSBuffer, item)
		} else {
			LogWithoutEcho("logcms: Line too long! Discarding...")
		}
	}

	glob.CMSBufferLock.Unlock()
}
func Log(input string) {
	text := sclean.StripControl(input)

	t := time.Now()
	date := fmt.Sprintf("%02d-%02d-%04d_%02d-%02d-%02d", t.Month(), t.Day(), t.Year(), t.Hour(), t.Minute(), t.Second())

	buf := fmt.Sprintf("%s %s", date, text)
	_, err := glob.BotLogDesc.WriteString(buf + "\n")
	if err != nil {
		fmt.Println(err)
	}

	buf = fmt.Sprintf("`%s` %s", date, text)
	cms(cfg.Local.ChannelData.LogID, buf)
}
