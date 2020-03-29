package fetcher

import "net/http"

var HttpClient *http.Client = &http.Client{
	Transport: uaWrapper{
		UserAgent: "Mozilla/5.0",
		Transport: http.DefaultTransport,
	},
}

type uaWrapper struct {
	UserAgent string
	Transport http.RoundTripper
}

func (u uaWrapper) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", u.UserAgent)
	return u.Transport.RoundTrip(req)
}
