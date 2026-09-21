// chatwire-web serves moderator web controls for registered local instances.
package main

import (
	"ChatWire/webapi"
	"ChatWire/webcontrol"
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
func run() error {
	path := flag.String("config", "cw-web-config.json", "Path to the web service configuration")
	check := flag.Bool("check-config", false, "Validate configuration and credentials without opening listeners or changing state")
	flag.Parse()
	var c webapi.Config
	if e := webcontrol.LoadConfig(*path, &c); e != nil {
		return e
	}
	if e := c.Validate(); e != nil {
		return e
	}
	if *check {
		if _, e := webcontrol.Credential(c.BrokerCredentialFile); e != nil {
			return e
		}
		for _, ep := range c.Instances {
			if ep.Enabled {
				if _, e := webcontrol.Credential(ep.CredentialFile); e != nil {
					return e
				}
			}
		}
		log.Print("Web configuration is valid.")
		return nil
	}
	// Own listeners before recovering persistent job state. A second process or a
	// config-check command must never reconcile jobs belonging to a live service.
	broker, e := webcontrol.Listen(c.BrokerSocket)
	if e != nil {
		return e
	}
	defer broker.Close()
	listener, e := net.Listen("tcp", c.Listen)
	if e != nil {
		return e
	}
	defer listener.Close()
	app, e := webapi.New(c)
	if e != nil {
		return e
	}
	app.SetConfigPath(*path)
	internal := &http.Server{Handler: app.Broker(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16384}
	public := &http.Server{Handler: app.Handler(), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16384}
	errs := make(chan error, 2)
	go func() { errs <- internal.Serve(broker) }()
	go func() { errs <- public.Serve(listener) }()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	var serverErr error
	select {
	case <-ctx.Done():
	case e := <-errs:
		if e != http.ErrServerClosed {
			serverErr = e
		}
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = public.Shutdown(shutdown)
	_ = internal.Shutdown(shutdown)
	return serverErr
}
