package tunnel

import (
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
				originStr := r.In.Header.Get("Origin")
				if origin, err := url.Parse(originStr); err == nil {
					origin.Scheme = target.Scheme
					origin.Host = target.Host

					r.Out.Header.Set("Origin", origin.String())
				}
			}
		},
	}
}
