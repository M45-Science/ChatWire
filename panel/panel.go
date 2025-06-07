package panel

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"html/template"
	"math/big"
	"net/http"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/M45-Science/rcon"

	"ChatWire/banlist"
	"ChatWire/cfg"
	"ChatWire/commands"
	"ChatWire/commands/moderator"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/support"
	"github.com/hako/durafmt"
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
        --accent: #b22020;
        --text: #ffffff;
        --radius: 0.4rem;
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
    .areas {
        display: grid;
        gap: var(--gap);
    }
    @media (min-width: 1500px) {
        .areas {
            grid-template-columns: repeat(3, 1fr);
        }
    }
    .area {
        display: flex;
        flex-direction: column;
        gap: var(--gap);
    }
    .section {
        background: var(--surface);
        border-radius: var(--radius);
        box-shadow: var(--shadow);
        padding: var(--gap);
    }
    .section-header {
        display: flex;
        align-items: center;
        background: var(--accent);
        color: var(--text);
        border-radius: var(--radius);
        padding: 0.5rem;
    }
    .section-header .title {
        flex-grow: 1;
        margin-left: 0.4rem;
    }
    .section-content {
        margin-top: var(--gap);
    }
    .section.collapsed .section-content {
        display: none;
    }
    .card {
        background: var(--surface);
        border-radius: var(--radius);
        box-shadow: var(--shadow);
        margin-bottom: var(--gap);
    }
    .card-header {
        display: flex;
        align-items: center;
        background: var(--accent);
        color: var(--text);
        border-radius: var(--radius) var(--radius) 0 0;
        padding: 0.4rem;
    }
    .card-header .title {
        flex-grow: 1;
        margin-left: 0.4rem;
    }
    .card-content {
        padding: var(--gap);
    }
    .card.collapsed .card-content {
        display: none;
    }
    .response-card {
        position: fixed;
        left: 50%;
        bottom: var(--gap);
        transform: translateX(-50%);
        z-index: 1000;
        max-width: 80%;
        min-width: 20rem;
    }
    button {
        background: var(--accent);
        color: var(--text);
        border: none;
        border-radius: var(--radius);
        padding: 0.4rem;
        margin: 0.2rem;
        cursor: pointer;
        width: 100%;
    }
    .button-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(10rem, 1fr));
        gap: var(--gap);
    }
    .button-grid form {
        margin: 0;
    }
    .button-grid button {
        display: flex;
        align-items: center;
        justify-content: center;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
    }
    .save-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(10rem, 1fr));
        gap: var(--gap);
    }
    .save-grid button {
        display: flex;
        align-items: center;
        justify-content: center;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
    }
    .cmd-grid {
        display: grid;
        grid-template-columns: repeat(4, 1fr);
        gap: var(--gap);
    }
    .cmd {
        background: var(--surface);
        padding: 0.3rem;
        border-radius: var(--radius);
        text-align: center;
    }
    .minimize {
        background: none;
        border: none;
        color: inherit;
        cursor: pointer;
        padding: 0 0.2rem;
        font-size: 1rem;
        margin-left: auto;
    }
    .close {
        background: none;
        border: none;
        color: inherit;
        cursor: pointer;
        padding: 0 0.2rem;
        font-size: 1rem;
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
    .bool-display {
        display: flex;
        align-items: center;
        gap: 0.5rem;
        padding: 0.2rem 0.4rem;
    }
    .bool-display span {
        flex: 0 0 70%;
        text-align: right;
    }
    .bool-display label {
        flex: 0 0 auto;
        display: flex;
        justify-content: flex-start;
    }
    .kv-display {
        display: flex;
        align-items: center;
        gap: 0.5rem;
        padding: 0.2rem 0.4rem;
    }
    .kv-display span {
        flex: 0 0 70%;
        text-align: right;
    }
    .kv-display input {
        flex: 0 1 30%;
        max-width: 30%;
        text-align: left;
        box-sizing: border-box;
        overflow: hidden;
        text-overflow: ellipsis;
    }
    .value-box {
        border: 1px solid var(--accent);
        background: var(--bg);
        color: var(--text);
        border-radius: var(--radius);
        padding: 0.2rem 0.4rem;
        min-width: 4rem;
        width: auto;
        margin: 0;
        text-decoration: none;
        display: inline-block;
    }
    .switch {
        position: relative;
        display: inline-block;
        width: 2.5rem;
        height: 1.3rem;
    }
    .switch input { opacity: 0; width: 0; height: 0; }
    .slider {
        position: absolute;
        cursor: default;
        top: 0; left: 0; right: 0; bottom: 0;
        background-color: #b22020;
        transition: .4s;
        border-radius: 1.3rem;
    }
    .slider:before {
        position: absolute;
        content: "";
        height: 1.1rem;
        width: 1.1rem;
        left: 0.1rem;
        bottom: 0.1rem;
        background-color: white;
        transition: .4s;
        border-radius: 50%;
    }
    .switch input:checked + .slider { background-color: #0a8a0a; }
    .switch input:checked + .slider:before { transform: translateX(1.2rem); }
    .cfg-group {
        border: 1px solid var(--accent);
        border-radius: var(--radius);
        margin-bottom: var(--gap);
    }
    .cfg-content > div:nth-child(odd),
    .cfg-content > p:nth-child(odd),
    .info-list > div:nth-child(odd),
    .info-list > p:nth-child(odd) {
        background: rgba(255,255,255,0.05);
    }
    .cfg-content > div,
    .cfg-content > p,
    .info-list > div,
    .info-list > p {
        padding: 0.2rem 0.4rem;
        font-size: 1.1rem;
    }
    .save-item {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 0.2rem;
    }
    .save-age {
        font-size: 0.8rem;
        color: #ccc;
    }
    .cfg-title {
        background: var(--accent);
        padding: 0.2rem 0.4rem;
        border-radius: var(--radius) var(--radius) 0 0;
    }
    .cfg-content { padding: 0.4rem; }
    form { margin: 0; }
    </style>
</head>
<body>
<div class="areas">
<div class="area section" id="info-area">
<div class="section-header"><span class="material-icons">info</span><span class="title">Info</span><button class="minimize">&#8211;</button></div>
<div class="section-content">
<div class="card">
<div class="card-header"><span class="material-icons">dashboard</span><span class="title">ChatWire Status</span><button class="minimize">&#8211;</button></div>
<div class="card-content info-list">
<div class="kv-display"><span>ChatWire version</span><input class="value-box" type="text" readonly value="{{.CWVersion}}"></div>
<div class="kv-display"><span>Up-time</span><input class="value-box" type="text" readonly value="{{.CWUp}}"></div>
<div class="kv-display"><span>Factorio version</span><input class="value-box" type="text" readonly value="{{.Factorio}}"></div>
{{if ne .SoftMod ""}}<div class="kv-display"><span>SoftMod version</span><input class="value-box" type="text" readonly value="{{.SoftMod}}"></div>{{end}}
<div class="kv-display"><span>Players online</span><input class="value-box" type="text" readonly value="{{.Players}}"></div>
<div class="kv-display"><span>Game time</span><input class="value-box" type="text" readonly value="{{.Gametime}}"></div>
<div class="kv-display"><span>Last save</span><input class="value-box" type="text" readonly value="{{.SaveName}}"></div>
<div class="kv-display"><span>Factorio up-time</span><input class="value-box" type="text" readonly value="{{.FactUp}}"></div>
<div class="kv-display"><span>UPS 10m/30m/1h</span><input class="value-box" type="text" readonly value="{{.UPS10}}/{{.UPS30}}/{{.UPS60}}"></div>
{{if ne .PlayHours ""}}<div class="kv-display"><span>Play hours</span><input class="value-box" type="text" readonly value="{{.PlayHours}}"></div>{{end}}
<div class="bool-display"><span>Paused</span><label class="switch"><input type="checkbox" disabled {{if .Paused}}checked{{end}}><span class="slider"></span></label></div>
{{if ne .NextReset ""}}<div class="kv-display"><span>Next map reset</span><input class="value-box" type="text" readonly value="{{.NextReset}} ({{.TimeTill}})"></div>{{end}}
{{if ne .ResetInterval ""}}<div class="kv-display"><span>Interval</span><input class="value-box" type="text" readonly value="{{.ResetInterval}}"></div>{{end}}
<div class="kv-display"><span>Total players</span><input class="value-box" type="text" readonly value="{{.Total}}"></div>
<div class="kv-display"><span>Members/Regulars/Veterans</span><input class="value-box" type="text" readonly value="{{.Mem}} | {{.Reg}} | {{.Vet}}"></div>
<div class="kv-display"><span>Moderators/Banned</span><input class="value-box" type="text" readonly value="{{.Mods}} | {{.Banned}}"></div>
</div>
</div>

<div class="card">
<div class="card-header"><span class="material-icons">storage</span><span class="title">Server Info</span><button class="minimize">&#8211;</button></div>
<div class="card-content">
<pre>{{.Info}}</pre>
</div>
</div>
</div>
</div>

<div class="area section" id="command-area">
<div class="section-header"><span class="material-icons">terminal</span><span class="title">Commands</span><button class="minimize">&#8211;</button></div>
<div class="section-content">
<div class="card">
<div class="card-header"><span class="material-icons">rule</span><span class="title">Moderator Commands</span><button class="minimize">&#8211;</button></div>
<div class="card-content">
{{range .CmdGroups}}
<h4>{{.Name}}</h4>
<div class="button-grid">
{{range .Cmds}}
<form method="POST" action="/action" class="cmd-form">
    <input type="hidden" name="token" value="{{$.Token}}">
    <input type="hidden" name="cmd" value="{{.Cmd}}">
    <button type="submit" title="{{.Cmd}}"><span class="material-icons">{{.Icon}}</span>{{.Label}}</button>
</form>
{{end}}
</div>
{{end}}
</div>
</div>

<div class="card">
<div class="card-header"><span class="material-icons">map</span><span class="title">Change Map</span><button class="minimize">&#8211;</button></div>
<div class="card-content">
<div class="save-grid">
{{range .Saves}}
<div class="save-item">
    <div class="save-age">{{.Age}}</div>
    <form method="POST" action="/action" class="cmd-form">
        <input type="hidden" name="token" value="{{$.Token}}">
        <input type="hidden" name="cmd" value="change-map">
        <input type="hidden" name="arg" value="{{.Name}}">
        <button type="submit">{{.Name}}</button>
    </form>
</div>
{{end}}
</div>
<form method="POST" action="/action" class="cmd-form">
    <input type="hidden" name="token" value="{{$.Token}}">
    <input type="hidden" name="cmd" value="change-map">
    <input type="text" name="arg" placeholder="save name">
<button type="submit">load</button>
</form>
</div>
</div>

<div class="card">
<div class="card-header"><span class="material-icons">terminal</span><span class="title">RCON Command</span><button class="minimize">&#8211;</button></div>
<div class="card-content">
<form method="POST" action="/action" class="cmd-form">
    <input type="hidden" name="token" value="{{.Token}}">
    <input type="hidden" name="cmd" value="rcon">
    <input type="text" name="arg" placeholder="/command">
    <label><input type="checkbox" name="all"> all servers</label>
<button type="submit">run</button>
</form>
</div>
</div>
<div class="card">
<div class="card-header"><span class="material-icons">schedule</span><span class="title">Set Play Hours</span><button class="minimize">&#8211;</button></div>
<div class="card-content">
<form method="POST" action="/action" class="cmd-form">
    <input type="hidden" name="token" value="{{.Token}}">
    <input type="hidden" name="cmd" value="config-hours">
    <div class="bool-display"><span>enable</span><label class="switch"><input type="checkbox" name="enabled" {{if .HoursEnabled}}checked{{end}}><span class="slider"></span></label></div><br>
    <input type="number" name="start" min="0" max="23" placeholder="start hour">
    <input type="number" name="end" min="0" max="23" placeholder="end hour">
<button type="submit">apply</button>
</form>
</div>
</div>

<div class="card">
<div class="card-header"><span class="material-icons">event</span><span class="title">Set Map Schedule</span><button class="minimize">&#8211;</button></div>
<div class="card-content">
<form method="POST" action="/action" class="cmd-form">
    <input type="hidden" name="token" value="{{.Token}}">
    <input type="hidden" name="cmd" value="set-schedule">
    <input type="number" name="days" placeholder="days" min="0">
    <input type="number" name="hours" placeholder="hours" min="0">
    <input type="text" name="date" placeholder="YYYY-MM-DD HH-MM-SS">
    <button type="submit">apply</button>
</form>
<form method="POST" action="/action" class="cmd-form">
    <input type="hidden" name="token" value="{{.Token}}">
    <input type="hidden" name="cmd" value="disable-schedule">
<button type="submit">disable</button>
</form>
</div>
</div>

<div class="card">
<div class="card-header"><span class="material-icons">person</span><span class="title">Set Player Level</span><button class="minimize">&#8211;</button></div>
<div class="card-content">
<form method="POST" action="/action" class="cmd-form">
    <input type="hidden" name="token" value="{{.Token}}">
    <input type="hidden" name="cmd" value="player-level">
    <input type="text" name="name" placeholder="player name">
    <input type="number" name="level" placeholder="level">
    <input type="text" name="reason" placeholder="reason">
<button type="submit">apply</button>
</form>
</div>
</div>

<div class="card">
<div class="card-header"><span class="material-icons">bolt</span><span class="title">Discord Commands</span><button class="minimize">&#8211;</button></div>
<div class="card-content">
<div class="cmd-grid">
{{range .Commands}}
<form method="POST" action="/action" class="cmd-form">
    <input type="hidden" name="token" value="{{$.Token}}">
    <input type="hidden" name="cmd" value="discord">
    <input type="hidden" name="arg" value="{{.Name}}">
    <button type="submit" title="{{.Description}}">{{.Name}}</button>
</form>
{{end}}
</div>
</div>
</div>
</div>
</div>

<div class="area section" id="config-area">
<div class="section-header"><span class="material-icons">settings</span><span class="title">Config</span><button class="minimize">&#8211;</button></div>
<div class="section-content">
<div class="card">
<div class="card-header"><span class="material-icons">build</span><span class="title">Local Configuration</span><button class="minimize">&#8211;</button></div>
<div class="card-content">
<pre>{{.LocalCfg}}</pre>
</div>
</div>
<div class="card">
<div class="card-header"><span class="material-icons">public</span><span class="title">Global Configuration</span><button class="minimize">&#8211;</button></div>
<div class="card-content">
<pre>{{.GlobalCfg}}</pre>
</div>
</div>
</div>
</div>
<script>
document.querySelectorAll('.cmd-form').forEach(f=>{
f.addEventListener('submit',async e=>{
e.preventDefault();
const data=new FormData(f);
const r=await fetch('/action',{method:'POST',body:data});
const t=await r.text();
showResponse(t);
});
});
function showResponse(m){
const c=document.createElement('div');
c.className='card response-card';
c.innerHTML='<div class="card-header"><span class="title">Response</span><button class="close">&times;</button></div><div class="card-content">'+m+'</div>';
document.body.appendChild(c);
const t=setTimeout(()=>c.remove(),15000);
c.querySelector('.close').addEventListener('click',()=>{clearTimeout(t);c.remove();});
}
function makeValueNode(v){
  if(/^https?:/.test(v)||v.startsWith('steam://')){
    const a=document.createElement('a');
    a.href=v;
    a.target='_blank';
    a.className='value-box';
    a.textContent=(v.includes('gosteam')||v.startsWith('steam://'))?'connect':v;
    return a;
  }
  const i=document.createElement('input');
  i.type='text';
  i.readOnly=true;
  i.className='value-box';
  i.value=v;
  return i;
}
function formatCfg(pre){
const lines=pre.textContent.split('\n');
pre.innerHTML='';
const root=document.createElement('div');
root.className='cfg-content';
pre.appendChild(root);
let group=null,content=root;
lines.forEach(l=>{
 if(!l.trim())return;
 if(!l.startsWith(' ')){
  const idx=l.indexOf(':');
  if(idx>-1 && l.slice(idx+1).trim()!==''){
    const k=l.slice(0,idx).trim();
    const v=l.slice(idx+1).trim();
    const div=document.createElement('div');
    div.className='kv-display';
    div.appendChild(document.createElement('span')).textContent=k;
    div.appendChild(makeValueNode(v));
    root.appendChild(div);
    group=null; content=root;
  }else{
    group=document.createElement('div');
    group.className='cfg-group';
    const t=document.createElement('div');
    t.className='cfg-title';
    t.textContent=l.trim().replace(/:$/, '');
    group.appendChild(t);
    content=document.createElement('div');
    content.className='cfg-content';
    group.appendChild(content);
    pre.appendChild(group);
  }
 }else if(content){
  const idx=l.indexOf(':');
  if(idx>-1){
    const k=l.slice(0,idx).trim();
    const v=l.slice(idx+1).trim();
    if(v==='true' || v==='false'){
      const bd=document.createElement('div');
      bd.className='bool-display';
      bd.innerHTML='<span>'+k+'</span><label class="switch"><input type="checkbox" disabled '+(v==='true'?'checked':'')+'><span class="slider"></span></label>';
      content.appendChild(bd);
    }else{
      const div=document.createElement('div');
      div.className='kv-display';
      div.appendChild(document.createElement('span')).textContent=k;
      div.appendChild(makeValueNode(v));
      content.appendChild(div);
    }
  }else{
    const p=document.createElement('p');
    p.textContent=l.trim();
    content.appendChild(p);
  }
 }
});
if(!root.children.length){pre.removeChild(root);}
}
document.querySelectorAll('.minimize').forEach(b=>{
    b.addEventListener('click',e=>{
        const box=b.closest('.card, .section');
        box.classList.toggle('collapsed');
    });
});
document.querySelectorAll('#config-area pre, #info-area pre').forEach(formatCfg);
</script>
</body></html>`

type panelCmd struct {
	Cmd   string
	Label string
	Icon  string
}

type panelSave struct {
	Name string
	Age  string
}

type panelCommand struct {
	Name        string
	Description string
}

type panelCmdGroup struct {
	Name string
	Cmds []panelCmd
}

var modCmdGroups = []panelCmdGroup{
	{
		Name: "ChatWire",
		Cmds: []panelCmd{
			{Cmd: "reboot-chatwire", Label: "Reboot ChatWire", Icon: "restart_alt"},
			{Cmd: "queue-reboot", Label: "Queue Reboot", Icon: "schedule"},
			{Cmd: "force-reboot", Label: "Force Reboot", Icon: "restart_alt"},
			{Cmd: "queue-fact-reboot", Label: "Queue Fact Reboot", Icon: "schedule"},
			{Cmd: "reload-config", Label: "Reload Config", Icon: "refresh"},
		},
	},
	{
		Name: "Factorio",
		Cmds: []panelCmd{
			{Cmd: "start-factorio", Label: "Start Factorio", Icon: "play_arrow"},
			{Cmd: "stop-factorio", Label: "Stop Factorio", Icon: "stop"},
			{Cmd: "install-factorio", Label: "Install Factorio", Icon: "download"},
			{Cmd: "update-factorio", Label: "Update Factorio", Icon: "update"},
			{Cmd: "new-map", Label: "New Map", Icon: "create_new_folder"},
			{Cmd: "archive-map", Label: "Archive Map", Icon: "archive"},
			{Cmd: "map-reset", Label: "Map Reset", Icon: "map"},
		},
	},
	{
		Name: "Mods",
		Cmds: []panelCmd{
			{Cmd: "update-mods", Label: "Update Mods", Icon: "system_update_alt"},
			{Cmd: "sync-mods", Label: "Sync Mods", Icon: "sync"},
		},
	},
}

type panelData struct {
	CWVersion     string
	Factorio      string
	SoftMod       string
	Players       int
	Gametime      string
	SaveName      string
	UPS10         string
	UPS30         string
	UPS60         string
	CWUp          string
	FactUp        string
	NextReset     string
	TimeTill      string
	ResetInterval string
	Total         int
	Mods          int
	Banned        int
	Mem           int
	Reg           int
	Vet           int
	PlayHours     string
	HoursEnabled  bool
	Paused        bool
	Token         string
	CmdGroups     []panelCmdGroup
	Saves         []panelSave
	Commands      []panelCommand
	Info          string
	LocalCfg      string
	GlobalCfg     string
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

func tokenValid(tok string) bool {
	glob.PanelTokenLock.RLock()
	_, ok := glob.PanelTokens[tok]
	glob.PanelTokenLock.RUnlock()
	return ok
}

func handlePanel(w http.ResponseWriter, r *http.Request) {
	tok := r.URL.Query().Get("token")
	if tok == "" || !tokenValid(tok) {
		if !*glob.LocalTestMode {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}
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
	var saves []panelSave
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
			info, _ := f.Info()
			modAge := time.Since(info.ModTime()).Round(time.Second)
			units, _ := durafmt.DefaultUnitsCoder.Decode("y:y,w:w,d:d,h:h,m:m,s:s,ms:ms,us:us")
			ageStr := durafmt.Parse(modAge).LimitFirstN(2).Format(units) + " ago"
			saves = append(saves, panelSave{Name: strings.TrimSuffix(f.Name(), ".zip"), Age: ageStr})
		}
	}
	if len(saves) > 16 {
		saves = saves[:16]
	}

	var cmdList []panelCommand
	for _, c := range commands.ListAllCommands() {
		cmdList = append(cmdList, panelCommand{Name: "/" + c.AppCmd.Name, Description: c.AppCmd.Description})
	}
	sort.Slice(cmdList, func(i, j int) bool { return cmdList[i].Name < cmdList[j].Name })

	groups := make([]panelCmdGroup, len(modCmdGroups))
	copy(groups, modCmdGroups)
	for idx := range groups {
		sort.Slice(groups[idx].Cmds, func(i, j int) bool { return groups[idx].Cmds[i].Label < groups[idx].Cmds[j].Label })
	}
	pd := panelData{CWVersion: constants.Version, Factorio: fact.FactorioVersion, SoftMod: softMod,
		Players: fact.NumPlayers, Gametime: fact.GametimeString, SaveName: fact.LastSaveName,
		CWUp: cwUptime, FactUp: factUptime,
		NextReset: nextReset, TimeTill: timeTill, ResetInterval: resetInterval,
		Total: total, Mods: mods, Banned: ban, PlayHours: playHours, Paused: paused,
		Token: tok, CmdGroups: groups, Saves: saves, Commands: cmdList, Info: buildInfoString(),
		LocalCfg: buildCfgString(cfg.Local), GlobalCfg: buildCfgString(cfg.Global)}
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
		if !*glob.LocalTestMode {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}
		userInfo = &glob.PanelTokenData{Name: "local"}
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
		cmdStr = strings.TrimPrefix(cmdStr, "/")
		cmdStr = "/" + cmdStr
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
	case "config-hours":
		if r.FormValue("enabled") != "" {
			cfg.Local.Options.PlayHourEnable = true
		} else {
			cfg.Local.Options.PlayHourEnable = false
		}
		if v := r.FormValue("start"); v != "" {
			if val, err := strconv.Atoi(v); err == nil && val >= 0 && val < 24 {
				cfg.Local.Options.PlayStartHour = val
			}
		}
		if v := r.FormValue("end"); v != "" {
			if val, err := strconv.Atoi(v); err == nil && val >= 0 && val < 24 {
				cfg.Local.Options.PlayEndHour = val
			}
		}
		cfg.WriteLCfg()
	case "set-schedule":
		n := cfg.ResetInterval{}
		if v := r.FormValue("days"); v != "" {
			if val, err := strconv.Atoi(v); err == nil {
				n.Days = val
			}
		}
		if v := r.FormValue("hours"); v != "" {
			if val, err := strconv.Atoi(v); err == nil {
				n.Hours = val
			}
		}
		cfg.Local.Options.ResetInterval = n
		fact.SetResetDate()
		if v := r.FormValue("date"); v != "" {
			layout := "2006-01-02 15-04-05"
			if t, err := time.Parse(layout, v); err == nil && t.After(time.Now().UTC()) {
				cfg.Local.Options.NextReset = t
			}
		}
		if cfg.Local.Options.NextReset.UTC().Sub(time.Now().UTC()) > (time.Hour*24*90 + time.Hour*24) {
			cfg.Local.Options.NextReset = time.Time{}
		}
		cfg.WriteLCfg()
		support.ConfigSoftMod()
	case "disable-schedule":
		cfg.Local.Options.ResetInterval = cfg.ResetInterval{}
		cfg.Local.Options.NextReset = time.Time{}
		cfg.WriteLCfg()
		support.ConfigSoftMod()
	case "discord":
		arg := strings.TrimPrefix(r.FormValue("arg"), "/")
		for _, c := range commands.ListAllCommands() {
			if strings.EqualFold(c.AppCmd.Name, arg) {
				c.Function(&c, fakeInteraction(userInfo))
				fmt.Fprintf(w, "discord command '%s' executed", arg)
				cwlog.DoLogAudit("%v: discord %s", userInfo.Name, arg)
				return
			}
		}
		fmt.Fprint(w, "command not found")
		return
	case "player-level":
		name := strings.ToLower(r.FormValue("name"))
		if name == "" {
			fmt.Fprint(w, "name required")
			return
		}
		lvlStr := r.FormValue("level")
		lvl, err := strconv.Atoi(lvlStr)
		if err != nil {
			fmt.Fprint(w, "invalid level")
			return
		}
		reason := r.FormValue("reason")
		if reason == "" {
			reason = "No reason given"
		}
		old := fact.PlayerLevelGet(name, false)
		glob.PlayerListLock.RLock()
		pl := glob.PlayerList[name]
		glob.PlayerListLock.RUnlock()
		if pl == nil {
			fmt.Fprint(w, "player not found")
			return
		}
		if lvl >= 0 && old == -1 {
			fact.WriteUnban(name)
		}
		if lvl == -1 && old != -1 {
			reasonString := fmt.Sprintf("%v -- Panel", reason)
			fact.WriteBan(name, reasonString)
			pl.BanReason = reasonString
		}
		fact.AutoPromote(name, false, false)
		fact.PlayerLevelSet(pl.Name, lvl, true)
		fact.SetPlayerListDirty()
		fmt.Fprintf(w, "level set to %v", lvl)
		cwlog.DoLogAudit("%v: player-level %s %d", userInfo.Name, pl.Name, lvl)
		return
	default:
		fmt.Fprintf(w, "Command '%s' not supported via panel", cmd)
		return
	}

	fmt.Fprintf(w, "Command '%s' executed", cmd)
	cwlog.DoLogAudit("%v: ran %s", userInfo.Name, cmd)

	userInfo.Time = time.Now().Unix()
}

func buildInfoString() string {
	var lines []string
	add := func(k, v string) {
		if v == "" || v == "0" || v == constants.Unknown || v == "(not configured)" {
			return
		}
		lines = append(lines, fmt.Sprintf("%s: %s", k, v))
	}

	add(constants.ProgName+" version", constants.Version)
	add("SoftMod version", glob.SoftModVersion)
	add("Factorio version", fact.FactorioVersion)

	now := time.Now().Round(time.Second)
	add("ChatWire up-time", now.Sub(glob.Uptime.Round(time.Second)).Round(time.Second).String())
	if !fact.FactorioBootedAt.IsZero() && fact.FactorioBooted {
		add("Factorio up-time", now.Sub(fact.FactorioBootedAt.Round(time.Second)).Round(time.Second).String())
	}

	if cfg.Local.Options.PlayHourEnable {
		add("Time restrictions", fmt.Sprintf("%d - %d GMT", cfg.Local.Options.PlayStartHour, cfg.Local.Options.PlayEndHour))
	}
	add("Save name", fact.LastSaveName)
	add("Map time", fact.GametimeString)
	if fact.NumPlayers > 0 {
		add("Players online", fmt.Sprintf("%d", fact.NumPlayers))
	}

	if fact.HasResetTime() {
		add("Next map reset", cfg.Local.Options.NextReset.Local().Format("Jan 02 2006 03:04PM"))
		add("Time till reset", fact.TimeTillReset())
		if fact.HasResetInterval() {
			add("Interval", fact.FormatResetInterval())
		}
	}

	ten, thirty, hour := fact.GetFactUPS()
	if hour > 0 {
		add("UPS Average", fmt.Sprintf("10m: %.2f, 30m: %.2f, 1h: %.2f", ten, thirty, hour))
	} else if thirty > 0 {
		add("UPS Average", fmt.Sprintf("10m: %.2f, 30m: %.2f", ten, thirty))
	} else if ten > 0 {
		add("UPS Average", fmt.Sprintf("10m: %.2f", ten))
	}

	glob.PlayerListLock.RLock()
	var mem, reg, vet, mod, ban int
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
			mod++
		}
	}
	bCount := len(banlist.BanList)
	ban += bCount
	glob.PlayerListLock.RUnlock()
	total := ban + mem + reg + vet + mod
	add("Members/Regulars/Veterans", fmt.Sprintf("%d | %d | %d", mem, reg, vet))
	add("Moderators/Banned", fmt.Sprintf("%d | %d", mod, ban))
	add("Total players", fmt.Sprintf("%d", total))

	if fact.PausedTicks > 4 {
		lines = append(lines, "Server is paused")
	}

	if url, ok := fact.MakeSteamURL(); ok {
		add("Steam connect", url)
	}

	return strings.Join(lines, "\n")
}

func cfgLines(v reflect.Value, prefix string) []string {
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	t := v.Type()
	var items []struct {
		name  string
		lines []string
	}
	for i := 0; i < v.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" || f.Tag.Get("json") == "-" {
			continue
		}
		if tag := f.Tag.Get("form"); tag == "-" || tag == "hidden" {
			continue
		}
		name := f.Tag.Get("web")
		if name == "" {
			name = f.Name
		}
		fv := v.Field(i)
		if fv.Kind() == reflect.Struct {
			sub := cfgLines(fv, prefix+"  ")
			items = append(items, struct {
				name  string
				lines []string
			}{name: name, lines: sub})
		} else {
			items = append(items, struct {
				name  string
				lines []string
			}{name: name, lines: []string{fmt.Sprintf("%v", fv.Interface())}})
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].name < items[j].name })
	var out []string
	for _, it := range items {
		if len(it.lines) == 1 {
			out = append(out, fmt.Sprintf("%s%s: %s", prefix, it.name, strings.TrimSpace(it.lines[0])))
		} else {
			out = append(out, fmt.Sprintf("%s%s:", prefix, it.name))
			for _, l := range it.lines {
				out = append(out, prefix+"  "+l)
			}
		}
	}
	return out
}

func buildCfgString(i interface{}) string {
	lines := cfgLines(reflect.ValueOf(i), "")
	return strings.Join(lines, "\n")
}

func fakeInteraction(u *glob.PanelTokenData) *discordgo.InteractionCreate {
	return &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type:      discordgo.InteractionApplicationCommand,
			Member:    &discordgo.Member{User: &discordgo.User{ID: u.DiscID, Username: u.Name}},
			ChannelID: cfg.Local.Channel.ChatChannel,
			GuildID:   cfg.Global.Discord.Guild,
		},
	}
}

func generateCert() (tls.Certificate, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}
	dnsName := cfg.Global.Paths.URLs.Domain
	if glob.LocalTestMode != nil && *glob.LocalTestMode {
		dnsName = "127.0.0.1"
	}
	tpl := x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{Organization: []string{"ChatWire"}},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(24 * time.Hour), DNSNames: []string{dnsName}, BasicConstraintsValid: true}
	der, err := x509.CreateCertificate(rand.Reader, &tpl, &tpl, &priv.PublicKey, priv)
	if err != nil {
		return tls.Certificate{}, err
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	return tls.X509KeyPair(certPEM, keyPEM)
}
