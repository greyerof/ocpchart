package commands

import (
	"fmt"

	"github.com/greyerof/ocpchart/internal/auth"
	"github.com/greyerof/ocpchart/internal/config"
	"github.com/greyerof/ocpchart/internal/thanos"
	"github.com/spf13/cobra"
)

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

var rootCmd = &cobra.Command{
	Use:   "ocpchart",
	Short: "Live ASCII Prometheus charts for OpenShift",
	Long: `ocpchart queries Prometheus/Thanos on OpenShift clusters and renders
live ASCII charts in the terminal. Supports interactive navigation,
auto-refresh, and metric discovery.`,
	SilenceUsage: true,
}

func init() {
	pf := rootCmd.PersistentFlags()
	pf.StringVar(&flagKubeconfig, "kubeconfig", "", "path to kubeconfig (defaults to $KUBECONFIG)")
	pf.StringVar(&flagThanosURL, "thanos-url", "", "Thanos querier URL (requires --token)")
	pf.StringVar(&flagToken, "token", "", "bearer token for Thanos (requires --thanos-url)")
	pf.BoolVar(&flagInsecureTLS, "insecure-tls", false, "skip TLS certificate verification")
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

// resolveCredentials validates auth flags and returns credentials.
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

// resolveClient builds auth + thanos client in one step.
func resolveClient() (*thanos.Client, error) {
	creds, err := resolveCredentials()
	if err != nil {
		return nil, err
	}

	return thanos.NewClient(creds)
}
