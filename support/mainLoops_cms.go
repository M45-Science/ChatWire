package support

import (
	"context"
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
		ctx := glob.RuntimeContext()
		tokens := make(chan struct{}, 5)
		for i := 0; i < cap(tokens); i++ {
			tokens <- struct{}{}
		}
		refill := time.NewTicker(5 * time.Second)
		defer refill.Stop()
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case <-refill.C:
				refillTokens:
					for {
						select {
						case tokens <- struct{}{}:
						default:
							break refillTokens
						}
					}
				}
			}
		}()

		for {
			if disc.DS == nil {
				if !waitForContext(ctx, time.Second) {
					return
				}
				continue
			}

			select {
			case <-ctx.Done():
				return
			case first := <-disc.CMSChan:
				lcopy := []disc.CMSBuf{first}
				timer := time.NewTimer(constants.CMSRate)

			collect:
				for {
					select {
					case <-ctx.Done():
						if !timer.Stop() {
							select {
							case <-timer.C:
							default:
							}
						}
						return
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
						if !takeCMSToken(ctx, tokens) {
							return
						}
						disc.SmartWriteDiscord(msg.Channel, msg.Text)
					}
				}

				/* Send out buffer, split up if needed */
				/* Factorio */
				buf := ""

				for _, line := range factmsg {
					next := appendCMSBufferLine(buf, line)
					if next == buf {
						continue
					}
					if buf != "" && len(next) >= constants.MaxDiscordMsgLen {
						if !takeCMSToken(ctx, tokens) {
							return
						}
						disc.SmartWriteDiscord(cfg.Local.Channel.ChatChannel, buf)
						glob.SetBootMessage(nil)
						glob.ResetUpdateMessage()
						buf = appendCMSBufferLine("", line)
					} else {
						buf = next
					}
				}
				if buf != "" {
					if !takeCMSToken(ctx, tokens) {
						return
					}
					disc.SmartWriteDiscord(cfg.Local.Channel.ChatChannel, buf)
					glob.SetBootMessage(nil)
					glob.ResetUpdateMessage()
				}

				/* Moderation */
				buf = ""
				for _, line := range moder {
					next := appendCMSBufferLine(buf, line)
					if next == buf {
						continue
					}
					if buf != "" && len(next) >= constants.MaxDiscordMsgLen {
						if !takeCMSToken(ctx, tokens) {
							return
						}
						disc.SmartWriteDiscord(cfg.Global.Discord.ReportChannel, buf)
						buf = appendCMSBufferLine("", line)
					} else {
						buf = next
					}
				}
				if buf != "" {
					if !takeCMSToken(ctx, tokens) {
						return
					}
					disc.SmartWriteDiscord(cfg.Global.Discord.ReportChannel, buf)
				}

				/* Don't send any more messages for a while (throttle) */
				if !waitForContext(ctx, constants.CMSRestTime) {
					return
				}
			}
		}
	}()
}

func waitForContext(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func takeCMSToken(ctx context.Context, tokens <-chan struct{}) bool {
	select {
	case <-ctx.Done():
		return false
	case <-tokens:
		return true
	}
}

func appendCMSBufferLine(buf, line string) string {
	line = strings.TrimRight(line, "\r")
	if strings.TrimSpace(line) == "" {
		return buf
	}
	if buf == "" {
		return line
	}
	return buf + "\n" + line
}
