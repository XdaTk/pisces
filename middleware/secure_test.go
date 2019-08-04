package middleware

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"pisces"
	"testing"
)

func TestSecure(t *testing.T) {
	e := pisces.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	h := func(c pisces.Context) error {
		return c.String(http.StatusOK, "test")
	}

	// Default
	_ = Secure()(h)(c)
	assert.Equal(t, "1; mode=block", rec.Header().Get(pisces.HeaderXXSSProtection))
	assert.Equal(t, "nosniff", rec.Header().Get(pisces.HeaderXContentTypeOptions))
	assert.Equal(t, "SAMEORIGIN", rec.Header().Get(pisces.HeaderXFrameOptions))
	assert.Equal(t, "", rec.Header().Get(pisces.HeaderStrictTransportSecurity))
	assert.Equal(t, "", rec.Header().Get(pisces.HeaderContentSecurityPolicy))

	// Custom
	req.Header.Set(pisces.HeaderXForwardedProto, "https")
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	_ = SecureWithConfig(SecureConfig{
		XSSProtection:         "",
		ContentTypeNosniff:    "",
		XFrameOptions:         "",
		HSTSMaxAge:            3600,
		ContentSecurityPolicy: "default-src 'self'",
	})(h)(c)
	assert.Equal(t, "", rec.Header().Get(pisces.HeaderXXSSProtection))
	assert.Equal(t, "", rec.Header().Get(pisces.HeaderXContentTypeOptions))
	assert.Equal(t, "", rec.Header().Get(pisces.HeaderXFrameOptions))
	assert.Equal(t, "max-age=3600; includeSubdomains", rec.Header().Get(pisces.HeaderStrictTransportSecurity))
	assert.Equal(t, "default-src 'self'", rec.Header().Get(pisces.HeaderContentSecurityPolicy))
	assert.Equal(t, "", rec.Header().Get(pisces.HeaderContentSecurityPolicyReportOnly))

	// Custom with CSPReportOnly flag
	req.Header.Set(pisces.HeaderXForwardedProto, "https")
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	_ = SecureWithConfig(SecureConfig{
		XSSProtection:         "",
		ContentTypeNosniff:    "",
		XFrameOptions:         "",
		HSTSMaxAge:            3600,
		ContentSecurityPolicy: "default-src 'self'",
		CSPReportOnly:         true,
	})(h)(c)
	assert.Equal(t, "", rec.Header().Get(pisces.HeaderXXSSProtection))
	assert.Equal(t, "", rec.Header().Get(pisces.HeaderXContentTypeOptions))
	assert.Equal(t, "", rec.Header().Get(pisces.HeaderXFrameOptions))
	assert.Equal(t, "max-age=3600; includeSubdomains", rec.Header().Get(pisces.HeaderStrictTransportSecurity))
	assert.Equal(t, "default-src 'self'", rec.Header().Get(pisces.HeaderContentSecurityPolicyReportOnly))
	assert.Equal(t, "", rec.Header().Get(pisces.HeaderContentSecurityPolicy))

	// Custom, with preload option enabled
	req.Header.Set(pisces.HeaderXForwardedProto, "https")
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	_ = SecureWithConfig(SecureConfig{
		HSTSMaxAge:         3600,
		HSTSPreloadEnabled: true,
	})(h)(c)
	assert.Equal(t, "max-age=3600; includeSubdomains; preload", rec.Header().Get(pisces.HeaderStrictTransportSecurity))

	// Custom, with preload option enabled and subdomains excluded
	req.Header.Set(pisces.HeaderXForwardedProto, "https")
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	_ = SecureWithConfig(SecureConfig{
		HSTSMaxAge:            3600,
		HSTSPreloadEnabled:    true,
		HSTSExcludeSubdomains: true,
	})(h)(c)
	assert.Equal(t, "max-age=3600; preload", rec.Header().Get(pisces.HeaderStrictTransportSecurity))
}
