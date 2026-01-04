package panel

import (
	"html/template"
	"sync"
	"time"

	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/glob"
	"ChatWire/watcher"
)

var (
	panelTmpl     *template.Template
	panelTmplLock sync.RWMutex
)

func loadTemplate() {
	tmpl, err := template.ParseFiles(constants.PanelTemplateFile)
	if err != nil {
		cwlog.DoLogCW("panel: template load error: %v", err)
		return
	}
	panelTmplLock.Lock()
	panelTmpl = tmpl
	panelTmplLock.Unlock()
}

// WatchTemplate monitors the panel template and reloads it when changed.
func WatchTemplate() {
	time.Sleep(time.Second)
	loadTemplate()
	watcher.Watch(constants.PanelTemplateFile, 5*time.Second, &glob.ServerRunning, loadTemplate)
}
