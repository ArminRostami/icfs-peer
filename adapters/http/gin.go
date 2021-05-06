package httpa

import (
	"icfs-client/adapters/ipfs"
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

type RevProxy struct {
	Url   *url.URL
	Proxy *httputil.ReverseProxy
}

func NewProxy(rawUrl string) (*RevProxy, error) {
	u, err := url.Parse(rawUrl)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse url")
	}
	return &RevProxy{Url: u, Proxy: httputil.NewSingleHostReverseProxy(u)}, nil
}

type Handler struct {
	IS     *ipfs.IpfsService
	RProxy *RevProxy
}

func (h *Handler) Serve() error {
	r := gin.Default()
	r.Any("/*proxyPath", h.proxy)
	return r.Run(":5200")
}

func (h *Handler) proxy(c *gin.Context) {
	c.Request.URL.Host = h.RProxy.Url.Host
	c.Request.URL.Scheme = h.RProxy.Url.Scheme
	c.Request.Header.Set("X-Forwarded-Host", c.GetHeader("Host"))
	c.Request.Host = h.RProxy.Url.Host

	h.RProxy.Proxy.ServeHTTP(c.Writer, c.Request)
}
