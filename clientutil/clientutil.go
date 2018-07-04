package clientutil

import (
	"context"
	"io"
	"net"
	"net/http"
	"path"
	"strings"
	"time"
)

type Client struct {
	*http.Client
	socketPath string
}

// url converts the given
func (c *Client) url(dest string) string {
	return "http://unix" + path.Join(c.socketPath, dest)
}

func (c *Client) Get(dest string) (*http.Response, error) {
	return c.Client.Get(c.url(dest))
}

func (c *Client) PostString(dest string, body string) (*http.Response, error) {
	return c.Client.Post(c.url(dest), "application/json", strings.NewReader(body))
}

func (c *Client) Post(dest string, body io.Reader) (*http.Response, error) {
	return c.Client.Post(c.url(dest), "application/json", body)
}

func GetClient(socketPath string) *Client {
	// Create the HTTP client and return it
	return &Client{
		Client: &http.Client{
			Transport: &http.Transport{
				DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
					dialer := &net.Dialer{Timeout: 5 * time.Second}
					return dialer.Dial("unix", socketPath)
				},
			},
		},
		socketPath: socketPath,
	}
}
