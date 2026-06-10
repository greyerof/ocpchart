package commands

import (
	"fmt"

	"github.com/greyerof/ocpchart/internal/auth"
	"github.com/greyerof/ocpchart/internal/config"
	"github.com/greyerof/ocpchart/internal/thanos"
	"github.com/spf13/cobra"
)

// version and commit are set at build time via ldflags in the Makefile.
var (
	version = "dev"
	commit  = "none"
)

var (
	flagKubeconfig  string
	flagThanosURL   string
	flagToken       string
	flagInsecureTLS bool
)

// NewRootCmd builds the CLI command tree with all subcommands and flags registered.
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "ocpchart",
		Short: "Live ASCII Prometheus charts for OpenShift",
		Long: `ocpchart queries Prometheus/Thanos on OpenShift clusters and renders
interactive ASCII charts in the terminal. Supports pan/zoom navigation,
live auto-refresh, and metric discovery.`,
		SilenceUsage: true,
	}

	pf := rootCmd.PersistentFlags()
	pf.StringVar(&flagKubeconfig, "kubeconfig", "", "path to kubeconfig (defaults to $KUBECONFIG)")
	pf.StringVar(&flagThanosURL, "thanos-url", "", "Thanos querier URL (requires --token)")
	pf.StringVar(&flagToken, "token", "", "bearer token for Thanos (requires --thanos-url)")
	pf.BoolVar(&flagInsecureTLS, "insecure-tls", false, "skip TLS certificate verification")

	rootCmd.AddCommand(
		newPlotCmd(),
		newMetricsCmd(),
		newVersionCmd(),
	)

	return rootCmd
}

// resolveCredentials validates the auth flag combination and returns the
// appropriate credentials (kubeconfig auto-discovery or manual token+URL).
func resolveCredentials() (*auth.Credentials, error) {
	if err := config.ValidateAuthFlags(flagKubeconfig, flagThanosURL, flagToken); err != nil {
		return nil, err
	}

	if flagThanosURL != "" && flagToken != "" {
		return auth.FromManual(flagThanosURL, flagToken, flagInsecureTLS), nil
	}

	creds, err := auth.FromKubeconfig(flagKubeconfig, flagInsecureTLS)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Discovered Thanos URL: %s\n", creds.ThanosURL)

	return creds, nil
}

// resolveClient resolves auth credentials and creates a Thanos API client.
func resolveClient() (*thanos.Client, error) {
	creds, err := resolveCredentials()
	if err != nil {
		return nil, err
	}

	return thanos.NewClient(creds)
}
