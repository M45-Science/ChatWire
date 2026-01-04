package support

import (
	"strings"
	"time"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/disc"
	"ChatWire/glob"
)

func startCMSBuffer() {
	/*************************************************
	 *  Send buffered messages to Discord, batched.
	 *************************************************/
	go func() {
		tokens := make(chan struct{}, 5)
		for i := 0; i < cap(tokens); i++ {
			tokens <- struct{}{}
		}
		refill := time.NewTicker(5 * time.Second)
		defer refill.Stop()
		go func() {
			for range refill.C {
				for len(tokens) < cap(tokens) {
					tokens <- struct{}{}
				}
			}
		}()

		for glob.ServerRunning {

			if disc.DS != nil {
				select {
				case first := <-disc.CMSChan:
					lcopy := []disc.CMSBuf{first}
					timer := time.NewTimer(constants.CMSRate)

				collect:
					for {
						select {
						case msg := <-disc.CMSChan:
							lcopy = append(lcopy, msg)
						case <-timer.C:
							break collect
						}
					}
					if !timer.Stop() {
						select {
						case <-timer.C:
						default:
						}
					}

					var factmsg []string
					var moder []string

					/* Put messages into proper lists */
					for _, msg := range lcopy {
						if strings.EqualFold(msg.Channel, cfg.Local.Channel.ChatChannel) {
							factmsg = append(factmsg, msg.Text)
						} else if strings.EqualFold(msg.Channel, cfg.Global.Discord.ReportChannel) {
							moder = append(moder, msg.Text)
						} else {
							<-tokens
							disc.SmartWriteDiscord(msg.Channel, msg.Text)
						}
					}

					/* Send out buffer, split up if needed */
					/* Factorio */
					buf := ""

					for _, line := range factmsg {
						oldlen := len(buf) + 1
						addlen := len(line)
						if oldlen+addlen >= constants.MaxDiscordMsgLen {
							<-tokens
							disc.SmartWriteDiscord(cfg.Local.Channel.ChatChannel, buf)
							glob.SetBootMessage(nil)
							glob.ResetUpdateMessage()
							buf = line
						} else {
							buf = buf + "\n" + line
						}
					}
					if buf != "" {
						<-tokens
						disc.SmartWriteDiscord(cfg.Local.Channel.ChatChannel, buf)
						glob.SetBootMessage(nil)
						glob.ResetUpdateMessage()
					}

					/* Moderation */
					buf = ""
					for _, line := range moder {
						oldlen := len(buf) + 1
						addlen := len(line)
						if oldlen+addlen >= constants.MaxDiscordMsgLen {
							<-tokens
							disc.SmartWriteDiscord(cfg.Global.Discord.ReportChannel, buf)
							buf = line
						} else {
							buf = buf + "\n" + line
						}
					}
					if buf != "" {
						<-tokens
						disc.SmartWriteDiscord(cfg.Global.Discord.ReportChannel, buf)
					}

					/* Don't send any more messages for a while (throttle) */
					time.Sleep(constants.CMSRestTime)
				case <-time.After(constants.CMSPollRate):
				}
			} else {
				time.Sleep(constants.CMSPollRate)
			}
		}
	}()
}
