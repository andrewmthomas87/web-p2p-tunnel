package tunnel

import (
	"fmt"
	"net/http/httputil"
	"net/url"
)

func newSingleHostReverseProxy(target *url.URL, changeHostHeader, changeOriginHeader bool) *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(target)
			r.SetXForwarded()

			if !changeHostHeader {
				r.Out.Host = r.In.Host
			}

			if changeOriginHeader {
				r.Out.Header.Set("Origin", fmt.Sprintf("%s://%s", target.Scheme, target.Host))
			}
		},
	}
}
