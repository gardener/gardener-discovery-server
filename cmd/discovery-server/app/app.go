// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gardener/gardener/pkg/client/kubernetes"
	gardenerhealthz "github.com/gardener/gardener/pkg/healthz"
	"github.com/gardener/gardener/pkg/logger"
	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	corev1 "k8s.io/api/core/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/component-base/version"
	"k8s.io/component-base/version/verflag"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	controllerconfig "sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/gardener/gardener-discovery-server/cmd/discovery-server/app/options"
	"github.com/gardener/gardener-discovery-server/internal/dynamiccert"
	oidhandler "github.com/gardener/gardener-discovery-server/internal/handler/openidmeta"
	"github.com/gardener/gardener-discovery-server/internal/metrics"
	oidreconciler "github.com/gardener/gardener-discovery-server/internal/reconciler/openidmeta"
	store "github.com/gardener/gardener-discovery-server/internal/store/openidmeta"
)

// AppName is the name of the application.
const AppName = "gardener-discovery-server"

// NewCommand is the root command for Gardener discovery server.
func NewCommand() *cobra.Command {
	opt := options.NewOptions()
	conf := &options.Config{}

	cmd := &cobra.Command{
		Use: AppName,
		RunE: func(cmd *cobra.Command, _ []string) error {
			logLevel, logFormat := "info", "json" // TODO make this configurable
			log, err := logger.NewZapLogger(logLevel, logFormat)
			if err != nil {
				return fmt.Errorf("error instantiating zap logger: %w", err)
			}
			logf.SetLogger(log)

			log.Info("Starting application", "app", AppName, "version", version.Get())
			cmd.Flags().VisitAll(func(flag *pflag.Flag) {
				log.Info("Flag", "name", flag.Name, "value", flag.Value, "default", flag.DefValue)
			})

			if err := opt.ApplyTo(conf); err != nil {
				return fmt.Errorf("cannot apply options: %w", err)
			}

			return run(cmd.Context(), log, conf)
		},
		PreRunE: func(_ *cobra.Command, _ []string) error {
			verflag.PrintAndExitIfRequested()
			return utilerrors.NewAggregate(opt.Validate())
		},
	}

	fs := cmd.Flags()
	verflag.AddFlags(fs)
	opt.AddFlags(fs)
	fs.AddGoFlagSet(flag.CommandLine)

	return cmd
}

func run(ctx context.Context, log logr.Logger, opts *options.Config) error {
	cfg, err := ctrl.GetConfig()
	if err != nil {
		return err
	}

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Logger: log.WithName("manager"),
		Scheme: kubernetes.GardenScheme,
		Metrics: metricsserver.Options{
			BindAddress: net.JoinHostPort("", "8080"),
			ExtraHandlers: map[string]http.Handler{
				"/metrics/discovery-server": promhttp.Handler(),
			},
		},
		GracefulShutdownTimeout: ptr.To(5 * time.Second),
		LeaderElection:          false,
		PprofBindAddress:        "",
		HealthProbeBindAddress:  net.JoinHostPort("", "8081"),
		Cache: cache.Options{
			ByObject: map[client.Object]cache.ByObject{
				&corev1.Secret{}: {
					Namespaces: map[string]cache.Config{
						"gardener-system-shoot-issuer": {},
					},
				},
			},
		},
		Controller: controllerconfig.Controller{
			RecoverPanic: ptr.To(true),
		},
	})
	if err != nil {
		return fmt.Errorf("unable to create manager: %w", err)
	}

	if err := mgr.AddHealthzCheck("ping", healthz.Ping); err != nil {
		return err
	}
	if err := mgr.AddReadyzCheck("informer-sync", gardenerhealthz.NewCacheSyncHealthz(mgr.GetCache())); err != nil {
		return err
	}

	store := store.NewStore()
	if err := (&oidreconciler.Reconciler{
		ResyncPeriod: opts.Resync.Duration,
		Store:        store,
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create controller: %w", err)
	}

	h := oidhandler.New(store, log.WithName("oid-meta-handler"))

	mux := http.NewServeMux()
	const (
		oidConfigPath = "/projects/{projectName}/shoots/{shootUID}/issuer/.well-known/openid-configuration"
		jwksPath      = "/projects/{projectName}/shoots/{shootUID}/issuer/jwks"
	)
	mux.Handle(
		"GET "+oidConfigPath,
		metrics.InstrumentHandler(oidConfigPath, http.HandlerFunc(h.HandleWellKnown)),
	)
	mux.Handle(
		"GET "+jwksPath,
		metrics.InstrumentHandler(jwksPath, http.HandlerFunc(h.HandleJWKS)),
	)

	cert, err := dynamiccert.New(
		opts.Serving.TLSCertFile,
		opts.Serving.TLSKeyFile,
		dynamiccert.WithLogger(log.WithName("dynamic-cert")),
		dynamiccert.WithRefreshInterval(5*time.Minute),
	)
	if err != nil {
		return fmt.Errorf("failed to parse discovery server certificates: %w", err)
	}

	srv := &http.Server{
		Addr:    opts.Serving.Address,
		Handler: mux,
		TLSConfig: &tls.Config{
			GetCertificate: cert.GetCertificate,
		},
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	srvCh := make(chan error)
	serverCtx, cancelSrv := context.WithCancel(ctx)

	mgrCh := make(chan error)
	mgrCtx, cancelMgr := context.WithCancel(ctx)

	go func() {
		defer cancelSrv()
		mgrCh <- mgr.Start(mgrCtx)
	}()

	go func() {
		defer cancelMgr()
		srvCh <- runServer(serverCtx, log, srv)
	}()

	select {
	case err := <-mgrCh:
		return errors.Join(err, <-srvCh)
	case err := <-srvCh:
		return errors.Join(err, <-mgrCh)
	}
}

// runServer starts the discovery server. It returns if the context is canceled or the server cannot start initially.
func runServer(ctx context.Context, log logr.Logger, srv *http.Server) error {
	log = log.WithName("discovery-server")
	errCh := make(chan error)
	go func(errCh chan<- error) {
		log.Info("Starts listening", "address", srv.Addr)
		defer close(errCh)
		if err := srv.ListenAndServeTLS("", ""); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("failed serving content: %w", err)
		} else {
			log.Info("Server stopped listening")
		}
	}(errCh)

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		log.Info("Shutting down")
		cancelCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		err := srv.Shutdown(cancelCtx)
		if err != nil {
			return fmt.Errorf("discovery server failed graceful shutdown: %w", err)
		}
		log.Info("Shutdown successful")
		return nil
	}
}
