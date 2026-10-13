package middleware

import (
	"context"
	"net/http"
)

// contextKey is a package-local type for context keys, preventing collisions
// with keys from other packages.
type contextKey string

const (
	KeyTransactionID contextKey = "transactionid"
	KeyMsisdn        contextKey = "x-msisdn"
	KeyAppVersion    contextKey = "appversion"
	KeyOSVersion     contextKey = "osversion"
	KeyDeviceID      contextKey = "x-device"
)

// SetContextValues extracts common request headers (transaction ID, MSISDN,
// app version, OS version, device ID) and stores them in the request context
// for downstream handlers.
func (m *Middleware) SetContextValues() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			ctx = context.WithValue(ctx, KeyTransactionID, r.Header.Get("transactionid"))
			ctx = context.WithValue(ctx, KeyMsisdn, r.Header.Get("x-msisdn"))

			appVersion := r.Header.Get("appversion")
			if appVersion == "" {
				appVersion = r.Header.Get("mytelkomsel-mobile-app-version")
			}
			if appVersion == "" {
				appVersion = r.Header.Get("mytelkomsel-web-app-version")
			}
			ctx = context.WithValue(ctx, KeyAppVersion, appVersion)

			ctx = context.WithValue(ctx, KeyOSVersion, r.Header.Get("osversion"))
			ctx = context.WithValue(ctx, KeyDeviceID, r.Header.Get("x-device"))

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetTransactionID retrieves the transaction ID stored by SetContextValues.
func GetTransactionID(r *http.Request) string {
	v, _ := r.Context().Value(KeyTransactionID).(string)
	return v
}

// GetMsisdn retrieves the MSISDN stored by SetContextValues.
func GetMsisdn(r *http.Request) string {
	v, _ := r.Context().Value(KeyMsisdn).(string)
	return v
}

// GetAppVersion retrieves the app version stored by SetContextValues.
func GetAppVersion(r *http.Request) string {
	v, _ := r.Context().Value(KeyAppVersion).(string)
	return v
}

// GetOSVersion retrieves the OS version stored by SetContextValues.
func GetOSVersion(r *http.Request) string {
	v, _ := r.Context().Value(KeyOSVersion).(string)
	return v
}

// GetDeviceID retrieves the device ID stored by SetContextValues.
func GetDeviceID(r *http.Request) string {
	v, _ := r.Context().Value(KeyDeviceID).(string)
	return v
}
