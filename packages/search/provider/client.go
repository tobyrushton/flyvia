package provider

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"net/url"

	utls "github.com/refraction-networking/utls"
	"golang.org/x/net/http2"
)

var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
}

type BrowserClientOptions struct {
	ProxyURL string
}

type BrowserClient struct {
	proxyURL *url.URL
}

func NewBrowserClient(opts BrowserClientOptions) (*BrowserClient, error) {
	var proxyURL *url.URL
	if opts.ProxyURL != "" {
		var err error
		proxyURL, err = url.Parse(opts.ProxyURL)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy URL: %w", err)
		}
	}

	return &BrowserClient{
		proxyURL: proxyURL,
	}, nil
}

func newUTLSConn(addr string, proxyURL *url.URL) (net.Conn, error) {
	var conn net.Conn
	var err error

	if proxyURL != nil {
		switch proxyURL.Scheme {
		case "", "http":
			conn, err = net.Dial("tcp", proxyURL.Host)
		case "https":
			proxyHost := proxyURL.Hostname()
			conn, err = tls.Dial("tcp", proxyURL.Host, &tls.Config{ServerName: proxyHost})
		default:
			return nil, fmt.Errorf("unsupported proxy scheme: %s", proxyURL.Scheme)
		}
		if err != nil {
			return nil, fmt.Errorf("proxy dial error: %w", err)
		}

		// build CONNECT request with proxy auth credentials
		connectReq := fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\n", addr, addr)
		if proxyURL.User != nil {
			password, _ := proxyURL.User.Password()
			credentials := base64.StdEncoding.EncodeToString(
				[]byte(proxyURL.User.Username() + ":" + password),
			)
			connectReq += fmt.Sprintf("Proxy-Authorization: Basic %s\r\n", credentials)
		}
		connectReq += "\r\n"

		if _, err := fmt.Fprint(conn, connectReq); err != nil {
			return nil, fmt.Errorf("proxy connect error: %w", err)
		}

		reader := bufio.NewReader(conn)
		resp, err := http.ReadResponse(reader, &http.Request{Method: http.MethodConnect})
		if err != nil {
			return nil, fmt.Errorf("proxy connect error: %w", err)
		}
		if resp.Body != nil {
			resp.Body.Close()
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("proxy tunnel failed: %s", resp.Status)
		}
	} else {
		conn, err = net.Dial("tcp", addr)
		if err != nil {
			return nil, fmt.Errorf("dial error: %w", err)
		}
	}

	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("split host error: %w", err)
	}

	uconn := utls.UClient(conn, &utls.Config{
		ServerName: host,
	}, utls.HelloChrome_Auto)

	if err := uconn.Handshake(); err != nil {
		return nil, fmt.Errorf("utls handshake error: %w", err)
	}

	return uconn, nil
}

type browserTransport struct {
	proxyURL *url.URL
}

func (t *browserTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	addr := req.URL.Host
	if req.URL.Port() == "" {
		addr = net.JoinHostPort(addr, "443")
	}

	conn, err := newUTLSConn(addr, t.proxyURL)
	if err != nil {
		return nil, err
	}

	tr := &http2.Transport{
		DialTLSContext: func(ctx context.Context, network, addr string, cfg *tls.Config) (net.Conn, error) {
			return conn, nil
		},
	}

	return tr.RoundTrip(req)
}

func (c *BrowserClient) Do(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", userAgents[rand.Intn(len(userAgents))])
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	req.Header.Set("Sec-Ch-Ua", `"Not_A Brand";v="8", "Chromium";v="120", "Google Chrome";v="120"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"Windows"`)
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")

	transport := &browserTransport{proxyURL: c.proxyURL}
	return (&http.Client{Transport: transport}).Do(req)
}
