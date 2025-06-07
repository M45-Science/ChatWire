package panel

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"github.com/M45-Science/rcon"
	"html/template"
	"math/big"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"ChatWire/banlist"
	"ChatWire/cfg"
	"ChatWire/commands/moderator"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/fact"
	"ChatWire/glob"
)

var panelHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8" />
    <title>ChatWire Panel</title>
    <link href="https://fonts.googleapis.com/icon?family=Material+Icons" rel="stylesheet" />
    <style>
    :root {
        --bg: #131313;
        --surface: #2b2b2b;
        --accent: #7efe83;
        --text: #ffffff;
        --radius: 1rem;
        --gap: 1rem;
        --shadow: 0 0.5rem 0.5rem rgba(0,0,0,0.8);
    }
    body {
        background: var(--bg);
        color: var(--text);
        font-family: 'Segoe UI', Roboto, sans-serif;
        margin: 0;
        padding: var(--gap);
    }
    .card {
        background: var(--surface);
        padding: var(--gap);
        border-radius: var(--radius);
        box-shadow: var(--shadow);
        margin-bottom: var(--gap);
    }
    button {
        background: var(--accent);
        border: none;
        border-radius: var(--radius);
        padding: 0.3rem 0.6rem;
        margin: 0.2rem 0;
        cursor: pointer;
    }
    input[type="text"] {
        width: 100%;
        border-radius: var(--radius);
        border: 1px solid var(--accent);
        background: var(--bg);
        color: var(--text);
        padding: 0.3rem 0.6rem;
        margin-bottom: 0.4rem;
    }
    form { margin: 0; }
    </style>
</head>
<body>
<div class="card">
<h2>ChatWire Status</h2>
<p>ChatWire version: {{.CWVersion}}</p>
<p>ChatWire up-time: {{.CWUp}}</p>
<p>Factorio version: {{.Factorio}}</p>
{{if ne .SoftMod ""}}<p>SoftMod version: {{.SoftMod}}</p>{{end}}
<p>Players online: {{.Players}}</p>
<p>Game time: {{.Gametime}}</p>
<p>Factorio up-time: {{.FactUp}}</p>
{{if ne .PlayHours ""}}<p>Play hours: {{.PlayHours}}</p>{{end}}
{{if .Paused}}<p>Server is paused</p>{{end}}
{{if ne .NextReset ""}}<p>Next map reset: {{.NextReset}} ({{.TimeTill}})</p>{{end}}
{{if ne .ResetInterval ""}}<p>Interval: {{.ResetInterval}}</p>{{end}}
<p>Total players: {{.Total}}</p>
<p>Moderators: {{.Mods}} | Banned: {{.Banned}}</p>
</div>

<div class="card">
<h3>Moderator Commands</h3>
{{range .Cmds}}
<form method="POST" action="/action">
    <input type="hidden" name="token" value="{{$.Token}}">
    <input type="hidden" name="cmd" value="{{.}}">
    <button type="submit">{{.}}</button>
</form>
{{end}}
</div>

<div class="card">
<h3>Change Map</h3>
{{range .Saves}}
<form method="POST" action="/action">
    <input type="hidden" name="token" value="{{$.Token}}">
    <input type="hidden" name="cmd" value="change-map">
    <input type="hidden" name="arg" value="{{.}}">
    <button type="submit">{{.}}</button>
</form>
{{end}}
<form method="POST" action="/action">
    <input type="hidden" name="token" value="{{$.Token}}">
    <input type="hidden" name="cmd" value="change-map">
    <input type="text" name="arg" placeholder="save name">
    <button type="submit">load</button>
</form>
</div>

<div class="card">
<h3>RCON Command</h3>
<form method="POST" action="/action">
    <input type="hidden" name="token" value="{{.Token}}">
    <input type="hidden" name="cmd" value="rcon">
    <input type="text" name="arg" placeholder="/command">
    <button type="submit">run</button>
</form>
</div>
</body></html>`

var modControls = []string{
	"force-reboot",
	"queue-reboot",
	"queue-fact-reboot",
	"reboot-chatwire",
	"reload-config",
	"start-factorio",
	"stop-factorio",
	"new-map",
	"archive-map",
	"update-mods",
	"sync-mods",
	"update-factorio",
	"install-factorio",
	"map-reset",
}

type panelData struct {
	CWVersion     string
	Factorio      string
	SoftMod       string
	Players       int
	Gametime      string
	CWUp          string
	FactUp        string
	NextReset     string
	TimeTill      string
	ResetInterval string
	Total         int
	Mods          int
	Banned        int
	PlayHours     string
	Paused        bool
	Token         string
	Cmds          []string
	Saves         []string
}

// Start runs the HTTPS status panel server.
func Start() {
	http.HandleFunc("/panel", handlePanel)
	http.HandleFunc("/action", handleAction)
	addr := fmt.Sprintf(":%v", cfg.Local.Port+constants.PanelPortOffset)
	go func() {
		cert, err := generateCert()
		if err != nil {
			cwlog.DoLogCW("Panel TLS error: %v", err)
			return
		}
		srv := &http.Server{Addr: addr, TLSConfig: &tls.Config{Certificates: []tls.Certificate{cert}}}
		cwlog.DoLogCW("Panel server listening on %v", addr)
		if err := srv.ListenAndServeTLS("", ""); err != nil {
			cwlog.DoLogCW("Panel server error: %v", err)
		}
	}()
}

// GenerateToken creates a temporary token for web access.
func GenerateToken(id string) string {
	token := glob.RandomBase64String(20)
	glob.PanelTokenLock.Lock()
	glob.PanelTokens[token] = &glob.PanelTokenData{Token: token, DiscID: id, Time: time.Now().Unix()}
	glob.PanelTokenLock.Unlock()
	return token
}

func handlePanel(w http.ResponseWriter, r *http.Request) {
	tok := r.URL.Query().Get("token")
	if tok == "" {
		http.Error(w, "token required", http.StatusUnauthorized)
		return
	}
	glob.PanelTokenLock.RLock()
	_, ok := glob.PanelTokens[tok]
	glob.PanelTokenLock.RUnlock()
	if !ok {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}
	t := template.Must(template.New("panel").Parse(panelHTML))

	cwUptime := time.Since(glob.Uptime.Round(time.Second)).Round(time.Second).String()

	factUptime := "not running"
	if !fact.FactorioBootedAt.IsZero() && fact.FactorioBooted {
		factUptime = time.Since(fact.FactorioBootedAt.Round(time.Second)).Round(time.Second).String()
	}

	nextReset := ""
	timeTill := ""
	resetInterval := ""
	if fact.HasResetTime() {
		nextReset = fact.FormatResetTime()
		timeTill = fact.TimeTillReset()
		resetInterval = fact.FormatResetInterval()
	}

	playHours := ""
	if cfg.Local.Options.PlayHourEnable {
		playHours = fmt.Sprintf("%d-%d GMT", cfg.Local.Options.PlayStartHour, cfg.Local.Options.PlayEndHour)
	}

	paused := false
	if fact.PausedTicks > 4 {
		paused = true
	}

	var mem, reg, vet, mods, ban int
	glob.PlayerListLock.RLock()
	for _, player := range glob.PlayerList {
		switch player.Level {
		case -1:
			ban++
		case 1:
			mem++
		case 2:
			reg++
		case 3:
			vet++
		case 255:
			mods++
		}
	}
	bCount := len(banlist.BanList)
	ban += bCount
	glob.PlayerListLock.RUnlock()
	total := ban + mem + reg + vet + mods

	softMod := ""
	if glob.SoftModVersion != constants.Unknown {
		softMod = glob.SoftModVersion
	}

	// gather last 24 saves
	var saves []string
	files, err := os.ReadDir(cfg.GetSavesFolder())
	if err == nil {
		var entries []os.DirEntry
		for _, f := range files {
			name := f.Name()
			if strings.HasSuffix(name, ".zip") && !strings.HasSuffix(name, "tmp.zip") && !strings.HasSuffix(name, cfg.Local.Name+"_new.zip") {
				entries = append(entries, f)
			}
		}
		sort.Slice(entries, func(i, j int) bool {
			iInfo, _ := entries[i].Info()
			jInfo, _ := entries[j].Info()
			return iInfo.ModTime().After(jInfo.ModTime())
		})
		if len(entries) > 24 {
			entries = entries[:24]
		}
		for _, f := range entries {
			saves = append(saves, strings.TrimSuffix(f.Name(), ".zip"))
		}
	}

	pd := panelData{CWVersion: constants.Version, Factorio: fact.FactorioVersion, SoftMod: softMod,
		Players: fact.NumPlayers, Gametime: fact.GametimeString, CWUp: cwUptime, FactUp: factUptime,
		NextReset: nextReset, TimeTill: timeTill, ResetInterval: resetInterval,
		Total: total, Mods: mods, Banned: ban, PlayHours: playHours, Paused: paused,
		Token: tok, Cmds: modControls, Saves: saves}
	_ = t.Execute(w, pd)
}

func handleAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	tok := r.FormValue("token")
	cmd := r.FormValue("cmd")
	if tok == "" || cmd == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	glob.PanelTokenLock.RLock()
	userInfo, ok := glob.PanelTokens[tok]
	glob.PanelTokenLock.RUnlock()
	if !ok {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	switch cmd {
	case "force-reboot":
		moderator.ForceReboot(nil, nil)
	case "queue-reboot":
		moderator.QueReboot(nil, nil)
	case "queue-fact-reboot":
		moderator.QueFactReboot(nil, nil)
	case "reboot-chatwire":
		moderator.RebootCW(nil, nil)
	case "reload-config":
		moderator.ReloadConfig(nil, nil)
	case "start-factorio":
		moderator.StartFact(nil, nil)
	case "stop-factorio":
		moderator.StopFact(nil, nil)
	case "new-map":
		moderator.NewMap(nil, nil)
	case "archive-map":
		moderator.ArchiveMap(nil, nil)
	case "update-mods":
		moderator.UpdateMods(nil, nil)
	case "sync-mods":
		moderator.SyncMods(nil, nil)
	case "update-factorio":
		moderator.UpdateFactorio(nil, nil)
	case "install-factorio":
		moderator.InstallFactorio(nil, nil)
	case "map-reset":
		moderator.MapReset(nil, nil)
	case "change-map":
		arg := r.FormValue("arg")
		if arg == "" {
			fmt.Fprint(w, "map name required")
			return
		}
		fact.DoChangeMap(arg)
	case "rcon":
		cmdStr := r.FormValue("arg")
		if cmdStr == "" {
			fmt.Fprint(w, "command required")
			return
		}
		portstr := fmt.Sprintf("%v", cfg.Local.Port+cfg.Global.Options.RconOffset)
		rc, err := rcon.Dial("localhost"+":"+portstr, glob.RCONPass)
		if err != nil || rc == nil {
			fmt.Fprint(w, "RCON connect failed")
			return
		}
		reqID, err := rc.Write(cmdStr)
		if err != nil {
			fmt.Fprint(w, "RCON write failed")
			return
		}
		resp, respID, err := rc.Read()
		if err != nil || reqID != respID {
			fmt.Fprint(w, "RCON read failed")
			return
		}
		if resp == "" {
			resp = "(Empty response)"
		}
		fmt.Fprint(w, resp)
		cwlog.DoLogAudit("%v: rcon %s", userInfo.Name, cmdStr)
		return
	default:
		fmt.Fprintf(w, "Command '%s' not supported via panel", cmd)
		return
	}

	fmt.Fprintf(w, "Command '%s' executed", cmd)
	cwlog.DoLogAudit("%v: ran %s", userInfo.Name, cmd)

	userInfo.Time = time.Now().Unix()
}

func generateCert() (tls.Certificate, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}
	tpl := x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{Organization: []string{"ChatWire"}},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(24 * time.Hour), DNSNames: []string{cfg.Global.Paths.URLs.Domain}, BasicConstraintsValid: true}
	der, err := x509.CreateCertificate(rand.Reader, &tpl, &tpl, &priv.PublicKey, priv)
	if err != nil {
		return tls.Certificate{}, err
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	return tls.X509KeyPair(certPEM, keyPEM)
}
