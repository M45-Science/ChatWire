package logs

import (
	"log"
	"strings"

	"../glob"
)

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
			log.Println("logcms: Line too long! Discarding...")
		}
	}

	glob.CMSBufferLock.Unlock()
}
