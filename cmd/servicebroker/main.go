package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path"
	"strconv"
	"syscall"

	prom "github.com/prometheus/client_golang/prometheus"

	"github.com/gogolok/osb-broker-lib/pkg/metrics"
	"github.com/gogolok/osb-broker-lib/pkg/rest"
	"github.com/gogolok/osb-broker-lib/pkg/server"
	"github.com/gogolok/osb-dummy-app/pkg/broker"
)

var options struct {
	broker.Options

	Port        int
	Insecure    bool
	TLSCert     string
	TLSKey      string
	TLSCertFile string
	TLSKeyFile  string
}

func init() {
	flag.IntVar(&options.Port, "port", 8443, "use '--port' option to specify the port for broker to listen on")
	flag.BoolVar(&options.Insecure, "insecure", false, "use --insecure to use HTTP vs HTTPS.")
	flag.StringVar(&options.TLSCertFile, "tls-cert-file", "", "File containing the default x509 Certificate for HTTPS. (CA cert, if any, concatenated after server cert).")
	flag.StringVar(&options.TLSKeyFile, "tls-private-key-file", "", "File containing the default x509 private key matching --tls-cert-file.")
	flag.StringVar(&options.TLSCert, "tlsCert", "", "base-64 encoded PEM block to use as the certificate for TLS. If '--tlsCert' is used, then '--tlsKey' must also be used.")
	flag.StringVar(&options.TLSKey, "tlsKey", "", "base-64 encoded PEM block to use as the private key matching the TLS certificate.")
	broker.AddFlags(&options.Options)
	flag.Parse()
}

func main() {
	if err := run(); err != nil && err != context.Canceled && err != context.DeadlineExceeded {
		slog.Error("error", "err", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	go cancelOnInterrupt(ctx, cancelFunc)

	return runWithContext(ctx)
}

func runWithContext(ctx context.Context) error {
	if flag.Arg(0) == "version" {
		fmt.Printf("%s/%s\n", path.Base(os.Args[0]), "0.1.0")
		return nil
	}
	if (options.TLSCert != "" || options.TLSKey != "") &&
		(options.TLSCert == "" || options.TLSKey == "") {
		fmt.Println("To use TLS with specified cert or key data, both --tlsCert and --tlsKey must be used")
		return nil
	}

	addr := ":" + strconv.Itoa(options.Port)

	businessLogic, err := broker.NewBusinessLogic(options.Options)
	if err != nil {
		return err
	}

	// Prom. metrics
	reg := prom.NewRegistry()
	osbMetrics := metrics.New()
	reg.MustRegister(osbMetrics)

	api, err := rest.NewAPISurface(businessLogic, osbMetrics)
	if err != nil {
		return err
	}

	s := server.New(api, reg)

	slog.Info("Starting broker!")

	if options.Insecure {
		err = s.Run(ctx, addr)
	} else {
		if options.TLSCert != "" && options.TLSKey != "" {
			slog.Info("Starting secure broker with TLS cert and key data")
			err = s.RunTLS(ctx, addr, options.TLSCert, options.TLSKey)
		} else {
			if options.TLSCertFile == "" || options.TLSKeyFile == "" {
				slog.Error("unable to run securely without TLS Certificate and Key. Please review options and if running with TLS, specify --tls-cert-file and --tls-private-key-file or --tlsCert and --tlsKey.")
				return nil
			}
			slog.Info("Starting secure broker with file based TLS cert and key")
			err = s.RunTLSWithTLSFiles(ctx, addr, options.TLSCertFile, options.TLSKeyFile)
		}
	}
	return err
}

func cancelOnInterrupt(ctx context.Context, f context.CancelFunc) {
	term := make(chan os.Signal)
	signal.Notify(term, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case <-term:
			slog.Info("Received SIGTERM, exiting gracefully...")
			f()
			os.Exit(0)
		case <-ctx.Done():
			os.Exit(0)
		}
	}
}
