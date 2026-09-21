package moderator

import (
	"ChatWire/disc"
	"ChatWire/glob"
	"ChatWire/webcontrol"
	"context"
	"github.com/bwmarrin/discordgo"
	"net/http"
	"sync"
	"time"
)

var webBroker struct {
	sync.RWMutex
	client *http.Client
	secret string
}

// ConfigureWebBroker is called once at startup; no listener is enabled by default.
func ConfigureWebBroker(socket, credentialFile string) error {
	key, e := webcontrol.Credential(credentialFile)
	if e != nil {
		return e
	}
	webBroker.Lock()
	webBroker.client = webcontrol.UnixClient(socket)
	webBroker.secret = key
	webBroker.Unlock()
	return nil
}
func Web(cmd *glob.CommandData, i *discordgo.InteractionCreate) {
	webBroker.RLock()
	client, key := webBroker.client, webBroker.secret
	webBroker.RUnlock()
	if client == nil {
		disc.InteractionEphemeralResponse(i, "Web controls", "Web controls have not been configured on this instance.")
		return
	}
	if i.Member == nil || i.Member.User == nil || !(disc.CheckAdmin(i) || disc.CheckModerator(i)) {
		disc.InteractionEphemeralResponse(i, "Access denied", "Moderator access required.")
		return
	}
	action := "login"
	for _, o := range i.ApplicationCommandData().Options {
		if o.Name == "action" {
			action = o.StringValue()
		}
	}
	if e := disc.DS.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral}}); e != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	content := ""
	var components []discordgo.MessageComponent
	if action == "logout-all" {
		e := webcontrol.Call(ctx, client, key, "POST", "/internal/v1/revocations", map[string]string{"user_id": i.Member.User.ID}, nil)
		if e != nil {
			content = "Unable to revoke web sessions. Try again shortly."
		} else {
			content = "All your web sessions and unused login links have been revoked."
		}
	} else {
		var grant webcontrol.Grant
		e := webcontrol.Call(ctx, client, key, "POST", "/internal/v1/login-grants", webcontrol.GrantRequest{UserID: i.Member.User.ID, InteractionID: i.ID}, &grant)
		if e != nil {
			content = "Unable to issue a login link. Verify that the web service is available and your moderator role is configured."
		} else {
			content = "Open ChatWire using your private, single-use link. It expires in two minutes. Do not share it."
			components = []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.Button{Label: "Open ChatWire", Style: discordgo.LinkButton, URL: grant.URL}}}}
		}
	}
	_, _ = disc.DS.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content, Components: &components})
}
