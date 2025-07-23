package middleware

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"go-slim.dev/slim"
	"go-slim.dev/slim/nego"
)

// CORSConfig defines the config for CORS middleware.
type CORSConfig struct {
	// AllowOrigin defines a list of origins that may access the resource.
	// Optional. Default value []string{"*"}.
	AllowOrigins []string

	// AllowOriginFunc is a custom function to validate the origin. It takes the
	// origin as an argument and returns true if allowed or false otherwise. If
	// an error is returned, it is returned by the handler. If this option is
	// set, AllowOrigins is ignored.
	// Optional.
	AllowOriginFunc func(origin string) (bool, error)

	// AllowMethods defines a list methods allowed when accessing the resource.
	// This is used in response to a preflight request.
	// Optional. Default value DefaultCORSConfig.AllowMethods.
	AllowMethods []string

	// AllowHeaders defines a list of request headers that can be used when
	// making the actual request. This is in response to a preflight request.
	// Optional. Default value []string{}.
	AllowHeaders []string

	// AllowCredentials indicates whether or not the response to the request
	// can be exposed when the credential flag is true. When used as part of
	// a response to a preflight request, this indicates whether or not the
	// actual request can be made using credentials.
	// Optional. Default value is false.
	AllowCredentials bool

	// ExposeHeaders defines the whitelist headers that clients are allowed to
	// access.
	// Optional. Default value []string{}.
	ExposeHeaders []string

	// MaxAge indicates how long (in seconds) the results of a preflight request
	// can be cached.
	// Optional. Default value 0.
	MaxAge int
}

func CORS() slim.MiddlewareFunc {
	return CORSWithConfig(CORSConfig{})
}

func CORSWithConfig(config CORSConfig) slim.MiddlewareFunc {
	if len(config.AllowOrigins) == 0 {
		config.AllowOrigins = []string{"*"}
	}
	if len(config.AllowMethods) == 0 {
		config.AllowMethods = []string{
			http.MethodGet,
			http.MethodHead,
			http.MethodPut,
			http.MethodPatch,
			http.MethodPost,
			http.MethodDelete,
		}
	}

	var allowOriginPatterns []string
	for _, origin := range config.AllowOrigins {
		pattern := regexp.QuoteMeta(origin)
		pattern = strings.ReplaceAll(pattern, "\\*", ".*")
		pattern = strings.ReplaceAll(pattern, "\\?", ".")
		pattern = "^" + pattern + "$"
		allowOriginPatterns = append(allowOriginPatterns, pattern)
	}

	allowMethods := strings.Join(config.AllowMethods, ",")
	allowHeaders := strings.Join(config.AllowHeaders, ",")
	exposeHeaders := strings.Join(config.ExposeHeaders, ",")
	maxAge := strconv.Itoa(config.MaxAge)

	return func(c slim.Context, next slim.HandlerFunc) error {
		req := c.Request()
		res := c.Response()
		origin := req.Header.Get(nego.HeaderOrigin)
		allowOrigin := ""

		preflight := req.Method == http.MethodOptions
		res.Header().Add(nego.HeaderVary, nego.HeaderOrigin)

		// No Origin provided
		if origin == "" {
			if !preflight {
				return next(c)
			}
			return c.NoContent(http.StatusNoContent)
		}

		if config.AllowOriginFunc != nil {
			allowed, err := config.AllowOriginFunc(origin)
			if err != nil {
				return err
			}
			if allowed {
				allowOrigin = origin
			}
		} else {
			// Check allowed origins
			for _, o := range config.AllowOrigins {
				if o == "*" && config.AllowCredentials {
					allowOrigin = origin
					break
				}
				if o == "*" || o == origin {
					allowOrigin = o
					break
				}
				if matchSubdomain(origin, o) {
					allowOrigin = origin
					break
				}
			}

			// Check allowed origin patterns
			for _, re := range allowOriginPatterns {
				if allowOrigin == "" {
					didx := strings.Index(origin, "://")
					if didx == -1 {
						continue
					}
					domAuth := origin[didx+3:]
					// to avoid regex cost by invalid long domain
					if len(domAuth) > 253 {
						break
					}

					if match, _ := regexp.MatchString(re, origin); match {
						allowOrigin = origin
						break
					}
				}
			}
		}

		// Origin isn't allowed
		if allowOrigin == "" {
			if !preflight {
				return next(c)
			}
			return c.NoContent(http.StatusNoContent)
		}

		// Simple request
		if !preflight {
			res.Header().Set(nego.HeaderAccessControlAllowOrigin, allowOrigin)
			if config.AllowCredentials {
				res.Header().Set(nego.HeaderAccessControlAllowCredentials, "true")
			}
			if exposeHeaders != "" {
				res.Header().Set(nego.HeaderAccessControlExposeHeaders, exposeHeaders)
			}
			return next(c)
		}

		// Preflight request
		res.Header().Add(nego.HeaderVary, nego.HeaderAccessControlRequestMethod)
		res.Header().Add(nego.HeaderVary, nego.HeaderAccessControlRequestHeaders)
		res.Header().Set(nego.HeaderAccessControlAllowOrigin, allowOrigin)
		res.Header().Set(nego.HeaderAccessControlAllowMethods, allowMethods)
		if config.AllowCredentials {
			res.Header().Set(nego.HeaderAccessControlAllowCredentials, "true")
		}
		if allowHeaders != "" {
			res.Header().Set(nego.HeaderAccessControlAllowHeaders, allowHeaders)
		} else {
			h := req.Header.Get(nego.HeaderAccessControlRequestHeaders)
			if h != "" {
				res.Header().Set(nego.HeaderAccessControlAllowHeaders, h)
			}
		}
		if config.MaxAge > 0 {
			res.Header().Set(nego.HeaderAccessControlMaxAge, maxAge)
		}
		return c.NoContent(http.StatusNoContent)
	}
}
