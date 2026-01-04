package panel

import (
	"crypto/tls"
	"fmt"
	"net/http"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
)

// Start runs the HTTPS status panel server.
func Start() {
	http.HandleFunc("/panel", handlePanel)
	http.HandleFunc("/panel-data", handlePanelData)
	http.HandleFunc("/action", handleAction)
	go WatchTemplate()
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
