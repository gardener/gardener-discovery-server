// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package options

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/pflag"
)

// Options contain the server options.
type Options struct {
	ResyncOptions  ResyncOptions
	ServingOptions ServingOptions
}

// ServingOptions are options applied to the discovery server.
type ServingOptions struct {
	TLSCertFile string
	TLSKeyFile  string

	Address string
	Port    uint
}

// AddFlags adds server options to flagset
func (s *ServingOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&s.TLSCertFile, "tls-cert-file", s.TLSCertFile, "File containing the x509 Certificate for HTTPS.")
	fs.StringVar(&s.TLSKeyFile, "tls-private-key-file", s.TLSKeyFile, "File containing the x509 private key matching --tls-cert-file.")

	fs.StringVar(&s.Address, "address", "", "The IP address that the server will listen on. If unspecified all interfaces will be used.")
	fs.UintVar(&s.Port, "port", 10443, "The port that the server will listen on.")
}

func (s *ServingOptions) Validate() []error {
	errs := []error{}
	if strings.TrimSpace(s.TLSCertFile) == "" {
		errs = append(errs, errors.New("--tls-cert-file is required"))
	}

	if strings.TrimSpace(s.TLSKeyFile) == "" {
		errs = append(errs, errors.New("--tls-private-key-file is required"))
	}

	return errs
}

func (s *ServingOptions) ApplyTo(c *ServingConfig) error {
	c.Address = net.JoinHostPort(s.Address, strconv.Itoa(int(s.Port)))
	serverCert, err := tls.LoadX509KeyPair(s.TLSCertFile, s.TLSKeyFile)
	if err != nil {
		return fmt.Errorf("failed to parse discovery server certificates: %w", err)
	}

	c.TLSConfig = &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		MinVersion:   tls.VersionTLS12,
	}

	return nil
}

type ResyncOptions struct {
	Duration time.Duration
}

func (o *ResyncOptions) AddFlags(fs *pflag.FlagSet) {
	fs.DurationVar(&o.Duration, "resync-period", time.Minute*10, "The period between reconciliations of cluster discovery information.")
}

func (o *ResyncOptions) Validate() []error {
	var errs []error
	if o.Duration <= 0 {
		errs = append(errs, errors.New("--resync-period must be positive"))
	}
	return errs
}

func (s *ResyncOptions) ApplyTo(c *ResyncConfig) error {
	c.Duration = s.Duration
	return nil
}

type ResyncConfig struct {
	Duration time.Duration
}

// NewOptions return options with default values.
func NewOptions() *Options {
	opts := &Options{
		ResyncOptions: ResyncOptions{},
	}
	return opts
}

// AddFlags adds server options to flagset
func (o *Options) AddFlags(fs *pflag.FlagSet) {
	o.ServingOptions.AddFlags(fs)
	o.ResyncOptions.AddFlags(fs)
}

// ApplyTo applies the options to the configuration.
func (o *Options) ApplyTo(server *Config) error {
	if err := o.ResyncOptions.ApplyTo(&server.Resync); err != nil {
		return err
	}

	return o.ServingOptions.ApplyTo(&server.Serving)
}

// Validate checks if options are valid
func (o *Options) Validate() []error {
	return append(
		o.ResyncOptions.Validate(),
		o.ServingOptions.Validate()...,
	)
}

// Config has all the context to run the discovery server
type Config struct {
	Resync  ResyncConfig
	Serving ServingConfig
}

type ServingConfig struct {
	TLSConfig *tls.Config
	Address   string
}
