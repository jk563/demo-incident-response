package middleware

import (
	"net/http"

	"github.com/aws/aws-xray-sdk-go/xray"
)

// XRay wraps each request in an X-Ray segment.
func XRay(name string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return xray.Handler(xray.NewFixedSegmentNamer(name), next)
	}
}
