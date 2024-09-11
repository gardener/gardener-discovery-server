// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package options

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/pflag"

	"github.com/gardener/gardener-discovery-server/internal/utils"
)

// Options contain the server options.
type Options struct {
	ResyncOptions  ResyncOptions
	ServingOptions ServingOptions
	GardenOptions  WorkloadIdentityOptions
}

// ServingOptions are options applied to the discovery server.
type ServingOptions struct {
	TLSCertFile string
	TLSKeyFile  string

	Address string
	Port    uint
}

// AddFlags adds server options to flagset
func (o *ServingOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.TLSCertFile, "tls-cert-file", o.TLSCertFile, "File containing the x509 Certificate for HTTPS.")
	fs.StringVar(&o.TLSKeyFile, "tls-private-key-file", o.TLSKeyFile, "File containing the x509 private key matching --tls-cert-file.")

	fs.StringVar(&o.Address, "address", "", "The IP address that the server will listen on. If unspecified all interfaces will be used.")
	fs.UintVar(&o.Port, "port", 10443, "The port that the server will listen on.")
}

// Validate checks if options are valid.
func (o *ServingOptions) Validate() []error {
	errs := []error{}
	if strings.TrimSpace(o.TLSCertFile) == "" {
		errs = append(errs, errors.New("--tls-cert-file is required"))
	}

	if strings.TrimSpace(o.TLSKeyFile) == "" {
		errs = append(errs, errors.New("--tls-private-key-file is required"))
	}

	return errs
}

// ApplyTo applies the options to the configuration.
func (o *ServingOptions) ApplyTo(c *ServingConfig) error {
	c.Address = net.JoinHostPort(o.Address, strconv.Itoa(int(o.Port)))

	c.TLSCertFile = o.TLSCertFile
	c.TLSKeyFile = o.TLSKeyFile
	return nil
}

// WorkloadIdentityOptions holds the options for the workload identity OIDC discovery documents.
type WorkloadIdentityOptions struct {
	// OpenIDConfigFile is the path to the file containing the openid configuration.
	OpenIDConfigFile string
	// JWKSFile is the path to the file containing the JWKS.
	JWKSFile string
}

// WorkloadIdentityConfig holds the configuration regarding the workload identity OIDC discovery documents.
type WorkloadIdentityConfig struct {
	// OpenIDConfig is the workload identity openid configuration.
	OpenIDConfig []byte
	// JWKS is the workload identity JWKS.
	JWKS []byte
	// OpenIDConfigPath is the http path the discovery server makes available the workload identity openid configuration document.
	OpenIDConfigPath string
	// JWKSPath is the http path the discovery server makes available the workload identity JWKS document.
	JWKSPath string
	// Enabled indicate whether the discovery server should serve workload identity discovery documents or not.
	Enabled bool
}

// AddFlags adds workload identity options to  flagset
func (o *WorkloadIdentityOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.OpenIDConfigFile, "workload-identity-openid-config-file", o.OpenIDConfigFile, "Path to garden workload identity openid configuration file.")
	fs.StringVar(&o.JWKSFile, "workload-identity-jwks-file", o.JWKSFile, "Path to garden workload identity JWKS file.")
}

// Validate checks if workload identity options are valid.
func (o *WorkloadIdentityOptions) Validate() []error {
	errs := []error{}

	if strings.TrimSpace(o.OpenIDConfigFile) == "" && strings.TrimSpace(o.JWKSFile) == "" {
		return nil
	}

	if strings.TrimSpace(o.OpenIDConfigFile) == "" {
		errs = append(errs, errors.New(`flag "workload-identity-openid-config-file" must be set when "workload-identity-jwks-file" is set`))
	}
	if strings.TrimSpace(o.JWKSFile) == "" {
		errs = append(errs, errors.New(`flag "workload-identity-jwks-file" must be set when "workload-identity-openid-config-file" is set`))
	}

	return errs
}

// ApplyTo applies the options to the configuration.
func (o *WorkloadIdentityOptions) ApplyTo(c *WorkloadIdentityConfig) error {
	if strings.TrimSpace(o.OpenIDConfigFile) == "" {
		// Serving workload identity discovery documents is optional,
		// if the flags are not set, this feature is considered disabled
		c.Enabled = false
		return nil
	}
	c.Enabled = true

	var err error
	c.OpenIDConfig, err = os.ReadFile(o.OpenIDConfigFile)
	if err != nil {
		return err
	}

	c.JWKS, err = os.ReadFile(o.JWKSFile)
	if err != nil {
		return err
	}

	conf, err := utils.LoadOpenIDConfig(c.OpenIDConfig)
	if err != nil {
		return err
	}

	issuerURL, err := url.Parse(conf.Issuer)
	if err != nil {
		return fmt.Errorf("failed to parse the issuer URL, %w", err)
	}
	c.OpenIDConfigPath = issuerURL.EscapedPath() + "/.well-known/openid-configuration"

	jwksURI, err := url.Parse(conf.JWKSURI)
	if err != nil {
		return fmt.Errorf("failed to parse JWKS URI, %w", err)
	}
	c.JWKSPath = jwksURI.EscapedPath()

	return nil
}

// ResyncOptions holds options regarding the resync interval between reconciliations.
type ResyncOptions struct {
	Duration time.Duration
}

// AddFlags adds the [ResyncOptions] flags to the flagset.
func (o *ResyncOptions) AddFlags(fs *pflag.FlagSet) {
	fs.DurationVar(&o.Duration, "resync-period", time.Minute*30, "The period between reconciliations of cluster discovery information.")
}

// Validate checks if options are valid.
func (o *ResyncOptions) Validate() []error {
	var errs []error
	if o.Duration <= 0 {
		errs = append(errs, errors.New("--resync-period must be positive"))
	}
	return errs
}

// ApplyTo applies the options to the configuration.
func (o *ResyncOptions) ApplyTo(c *ResyncConfig) error {
	c.Duration = o.Duration
	return nil
}

// ResyncConfig holds configurations regarding the resync interval between reconciliations.
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
	o.GardenOptions.AddFlags(fs)
}

// ApplyTo applies the options to the configuration.
func (o *Options) ApplyTo(server *Config) error {
	if err := o.ResyncOptions.ApplyTo(&server.Resync); err != nil {
		return err
	}

	if err := o.GardenOptions.ApplyTo(&server.WorkloadIdentity); err != nil {
		return err
	}

	return o.ServingOptions.ApplyTo(&server.Serving)
}

// Validate checks if options are valid.
func (o *Options) Validate() []error {
	return slices.Concat(
		o.ResyncOptions.Validate(),
		o.ServingOptions.Validate(),
		o.GardenOptions.Validate(),
	)
}

// Config has all the context to run the discovery server.
type Config struct {
	Resync           ResyncConfig
	Serving          ServingConfig
	WorkloadIdentity WorkloadIdentityConfig
}

// ServingConfig has the context to run an http server.
type ServingConfig struct {
	Address string

	TLSCertFile string
	TLSKeyFile  string
}
