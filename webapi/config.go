package webapi

import (
	"ChatWire/webcontrol"
	"errors"
	"net"
	"net/url"
	"path/filepath"
	"strings"
)

type Config struct {
	StateDir             string                `json:"state_dir"`
	Listen               string                `json:"listen"`
	PublicOrigin         string                `json:"public_origin"`
	BrokerSocket         string                `json:"broker_socket"`
	BrokerCredentialFile string                `json:"broker_credential_file"`
	Primary              string                `json:"primary"`
	GuildID              string                `json:"guild_id"`
	AdminRoleIDs         []string              `json:"admin_role_ids"`
	ModeratorRoleIDs     []string              `json:"moderator_role_ids"`
	Instances            []webcontrol.Endpoint `json:"instances"`
}

func (c Config) Validate() error {
	if !filepath.IsAbs(c.StateDir) {
		return errors.New("state_dir must be absolute")
	}
	if len(c.Instances) > 64 {
		return errors.New("at most 64 local instances are supported")
	}
	u, e := url.Parse(c.PublicOrigin)
	if e != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "" {
		return errors.New("public_origin must be an HTTPS origin without a path")
	}
	host, _, e := net.SplitHostPort(c.Listen)
	if e != nil || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() {
		return errors.New("listen must be a literal loopback address and port; use an HTTPS reverse proxy")
	}
	if !filepath.IsAbs(c.BrokerSocket) || c.GuildID == "" || len(c.AdminRoleIDs) == 0 || len(c.ModeratorRoleIDs) == 0 {
		return errors.New("broker socket, guild ID, and role IDs are required")
	}
	seen := map[string]bool{}
	sockets := map[string]bool{}
	primary := false
	for _, s := range c.Instances {
		if !webcontrol.ValidID(s.ID) || seen[s.ID] || !filepath.IsAbs(s.Socket) || sockets[s.Socket] || s.Socket == c.BrokerSocket {
			return errors.New("instances require unique IDs and absolute unique socket paths")
		}
		seen[s.ID] = true
		sockets[s.Socket] = true
		if s.ID == c.Primary && s.Enabled {
			primary = true
		}
	}
	if !primary {
		return errors.New("primary must reference an enabled instance")
	}
	for _, r := range append(append([]string{}, c.AdminRoleIDs...), c.ModeratorRoleIDs...) {
		if strings.Trim(r, "0123456789") != "" || r == "" {
			return errors.New("role IDs must be Discord snowflakes")
		}
	}
	return nil
}
