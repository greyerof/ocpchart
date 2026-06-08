package auth

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	authv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	monitoringNamespace = "openshift-monitoring"
	serviceAccountName  = "prometheus-k8s"
	thanosRoutePath     = "/apis/route.openshift.io/v1/namespaces/openshift-monitoring/routes/thanos-querier"
	tokenDuration       = 10 * time.Minute
	routeRequestTimeout = 30 * time.Second
)

type Credentials struct {
	ThanosURL   string
	Token       string
	InsecureTLS bool
}

// FromManual creates credentials from user-provided token and URL.
func FromManual(thanosURL, token string, insecureTLS bool) *Credentials {
	return &Credentials{
		ThanosURL:   thanosURL,
		Token:       token,
		InsecureTLS: insecureTLS,
	}
}

// FromKubeconfig discovers the Thanos URL and creates a short-lived SA token.
func FromKubeconfig(kubeconfigPath string, insecureTLS bool) (*Credentials, error) {
	if kubeconfigPath == "" {
		kubeconfigPath = os.Getenv("KUBECONFIG")
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("loading kubeconfig: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("creating kubernetes client: %w", err)
	}

	thanosHost, err := discoverThanosURL(config)
	if err != nil {
		return nil, fmt.Errorf("discovering thanos URL: %w", err)
	}

	token, err := createSAToken(clientset)
	if err != nil {
		return nil, fmt.Errorf("creating service account token: %w", err)
	}

	return &Credentials{
		ThanosURL:   "https://" + thanosHost,
		Token:       token,
		InsecureTLS: insecureTLS,
	}, nil
}

// discoverThanosURL resolves the Thanos route host from the OpenShift Route API.
func discoverThanosURL(config *rest.Config) (string, error) {
	transport, err := rest.TransportFor(config)
	if err != nil {
		return "", fmt.Errorf("creating transport: %w", err)
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   routeRequestTimeout,
	}
	url := config.Host + thanosRoutePath

	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("fetching thanos route: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("thanos route returned status %d", resp.StatusCode)
	}

	var route struct {
		Spec struct {
			Host string `json:"host"`
		} `json:"spec"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&route); err != nil {
		return "", fmt.Errorf("decoding route: %w", err)
	}

	if route.Spec.Host == "" {
		return "", fmt.Errorf("thanos route has empty host")
	}

	return route.Spec.Host, nil
}

// createSAToken issues a short-lived token for the monitoring service account.
func createSAToken(clientset kubernetes.Interface) (string, error) {
	expSeconds := int64(tokenDuration.Seconds())
	tokenReq := &authv1.TokenRequest{
		Spec: authv1.TokenRequestSpec{
			ExpirationSeconds: &expSeconds,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := clientset.CoreV1().ServiceAccounts(monitoringNamespace).CreateToken(
		ctx, serviceAccountName, tokenReq, metav1.CreateOptions{},
	)
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}

	return result.Status.Token, nil
}

// HTTPClient returns an http.Client that injects the bearer token.
func (c *Credentials) HTTPClient() *http.Client {
	return &http.Client{
		Transport: &tokenTransport{
			token: c.Token,
			base: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: c.InsecureTLS, //nolint:gosec // user-controlled flag
				},
			},
		},
	}
}

type tokenTransport struct {
	token string
	base  http.RoundTripper
}

// RoundTrip clones a request and injects the bearer token header.
func (t *tokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req2 := req.Clone(req.Context())
	req2.Header.Set("Authorization", "Bearer "+t.token)
	return t.base.RoundTrip(req2)
}
