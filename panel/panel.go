package panel

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"html/template"
	"math/big"
	"net/http"
	"time"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/fact"
	"ChatWire/glob"
)

var panelHTML = `<!DOCTYPE html>
<html><head><title>ChatWire Status</title></head>
<body>
<h2>ChatWire Status</h2>
<p>Factorio version: {{.Factorio}}</p>
<p>Players online: {{.Players}}</p>
<p>Game time: {{.Gametime}}</p>
</body></html>`

type panelData struct {
	Factorio string
	Players  int
	Gametime string
}

// Start runs the HTTPS status panel server.
func Start() {
	http.HandleFunc("/panel", handlePanel)
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
	glob.PanelTokenLock.Lock()
	_, ok := glob.PanelTokens[tok]
	if ok {
		delete(glob.PanelTokens, tok)
	}
	glob.PanelTokenLock.Unlock()
	if !ok {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}
	t := template.Must(template.New("panel").Parse(panelHTML))
	pd := panelData{Factorio: fact.FactorioVersion, Players: fact.NumPlayers, Gametime: fact.GametimeString}
	_ = t.Execute(w, pd)
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
