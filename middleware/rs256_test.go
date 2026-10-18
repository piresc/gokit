package middleware_test

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/piresc/gokit/auth"
	"github.com/piresc/gokit/config"
	"github.com/piresc/gokit/middleware"
	"github.com/piresc/gokit/redis"
	"github.com/piresc/gokit/testkit"
)

type countingChecker struct {
	calls atomic.Int32
}

func (c *countingChecker) IsRevoked(context.Context, string) (bool, error) {
	c.calls.Add(1)
	return false, nil
}

type hangingChecker struct{}

func (hangingChecker) IsRevoked(ctx context.Context, _ string) (bool, error) {
	<-ctx.Done()
	return false, ctx.Err()
}

func newRS256Redis(t *testing.T) (*auth.RedisRevocationStore, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	client, err := redis.NewClient(config.RedisConfig{
		Host: mr.Host(),
		Port: int(mr.Server().Addr().Port),
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })
	return auth.NewRedisRevocationStore(client), mr
}

func signRS256(t *testing.T, key *rsa.PrivateKey, claims auth.RSAClaims) string {
	t.Helper()
	tok, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(key)
	require.NoError(t, err)
	return tok
}

func errField(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	s, _ := body["error"].(string)
	return s
}

func TestRS256Auth(t *testing.T) {
	priv, _, pubPEM := testkit.NewRSAKeyPair(t)
	otherPriv, _, _ := testkit.NewRSAKeyPair(t)
	pub, err := auth.LoadRSAPublicKey(string(pubPEM), "")
	require.NoError(t, err)
	verifier := auth.NewRSAVerifier(pub)
	signer := auth.NewRSASigner(priv, "test", time.Hour)
	otherSigner := auth.NewRSASigner(otherPriv, "test", time.Hour)
	expiredSigner := auth.NewRSASigner(priv, "test", -time.Hour)

	valid, validJTI, _, err := signer.Sign("u-1", "a@b.c", "investor")
	require.NoError(t, err)
	expired, _, _, err := expiredSigner.Sign("u-1", "a@b.c", "investor")
	require.NoError(t, err)
	wrongKey, _, _, err := otherSigner.Sign("u-1", "a@b.c", "investor")
	require.NoError(t, err)
	hs, err := jwt.NewWithClaims(jwt.SigningMethodHS256, auth.RSAClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "u-1",
			ID:        "hs-jti",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}).SignedString(pubPEM)
	require.NoError(t, err)
	noJTI := signRS256(t, priv, auth.RSAClaims{RegisteredClaims: jwt.RegisteredClaims{
		Subject:   "u-1",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}})

	tests := []struct {
		name       string
		header     string
		setup      func(t *testing.T) (auth.RevocationChecker, middleware.RS256Options)
		wantStatus int
		wantErr    string
		wantCalled bool
		wantRetry  string
	}{
		{name: "no header", wantStatus: 401, wantErr: "authentication required"},
		{name: "basic scheme", header: "Basic abc", wantStatus: 401, wantErr: "authentication required"},
		{name: "empty bearer", header: "Bearer ", wantStatus: 401, wantErr: "authentication required"},
		{name: "valid", header: "Bearer " + valid, wantStatus: 200, wantCalled: true},
		{name: "expired", header: "Bearer " + expired, wantStatus: 401, wantErr: "token expired"},
		{name: "wrong key", header: "Bearer " + wrongKey, wantStatus: 401, wantErr: "authentication required"},
		{name: "hs256 alg confusion", header: "Bearer " + hs, wantStatus: 401, wantErr: "authentication required"},
		{
			name: "revoked", header: "Bearer " + valid, wantStatus: 401, wantErr: "token revoked",
			setup: func(t *testing.T) (auth.RevocationChecker, middleware.RS256Options) {
				store, _ := newRS256Redis(t)
				require.NoError(t, store.Revoke(context.Background(), validJTI, time.Minute))
				return store, middleware.RS256Options{}
			},
		},
		{
			name: "redis closed fail closed", header: "Bearer " + valid, wantStatus: 503,
			wantErr: "service unavailable", wantRetry: "5",
			setup: func(t *testing.T) (auth.RevocationChecker, middleware.RS256Options) {
				store, mr := newRS256Redis(t)
				mr.Close()
				return store, middleware.RS256Options{RevocationTimeout: 200 * time.Millisecond}
			},
		},
		{
			name: "redis closed fail open", header: "Bearer " + valid, wantStatus: 200, wantCalled: true,
			setup: func(t *testing.T) (auth.RevocationChecker, middleware.RS256Options) {
				store, mr := newRS256Redis(t)
				mr.Close()
				return store, middleware.RS256Options{FailOpen: true, RevocationTimeout: 200 * time.Millisecond}
			},
		},
		{name: "revocation nil", header: "Bearer " + valid, wantStatus: 200, wantCalled: true},
		{
			name: "empty jti skips lookup", header: "Bearer " + noJTI, wantStatus: 200, wantCalled: true,
			setup: func(t *testing.T) (auth.RevocationChecker, middleware.RS256Options) {
				store, mr := newRS256Redis(t)
				if err := mr.Set(auth.RevocationKeyPrefix, "1"); err != nil {
					t.Fatal(err)
				}
				return store, middleware.RS256Options{}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var opts middleware.RS256Options
			if tt.setup != nil {
				var rc auth.RevocationChecker
				rc, opts = tt.setup(t)
				opts.Revocation = rc
			}
			called := false
			h := chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				if tt.wantStatus == 200 {
					assert.Equal(t, "u-1", middleware.GetUserID(r))
					if tt.header != "Bearer "+noJTI {
						assert.Equal(t, "investor", middleware.GetRole(r))
					}
					c, ok := middleware.GetClaims(r)
					assert.True(t, ok)
					assert.Equal(t, "u-1", c.Subject)
				}
				w.WriteHeader(http.StatusOK)
			}), newMiddleware().RS256Auth(verifier, opts))

			headers := map[string]string{}
			if tt.header != "" {
				headers["Authorization"] = tt.header
			}
			w := performRequest(h, http.MethodGet, "/", headers)
			assert.Equal(t, tt.wantStatus, w.Code)
			assert.Equal(t, tt.wantCalled, called)
			if tt.wantErr != "" {
				assert.Equal(t, tt.wantErr, errField(t, w))
			}
			assert.Equal(t, tt.wantRetry, w.Header().Get("Retry-After"))
		})
	}
}

func TestRS256Auth_ValidClaims(t *testing.T) {
	priv, _, pubPEM := testkit.NewRSAKeyPair(t)
	pub, err := auth.LoadRSAPublicKey(string(pubPEM), "")
	require.NoError(t, err)
	tok, _, _, err := auth.NewRSASigner(priv, "test", time.Hour).Sign("u-1", "a@b.c", "investor")
	require.NoError(t, err)

	called := false
	h := newMiddleware().RS256Auth(auth.NewRSAVerifier(pub), middleware.RS256Options{})(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			assert.Equal(t, "u-1", middleware.GetUserID(r))
			assert.Equal(t, "investor", middleware.GetRole(r))
			c, ok := middleware.GetClaims(r)
			require.True(t, ok)
			assert.Equal(t, "a@b.c", c.Email)
		}))
	w := performRequest(h, http.MethodGet, "/", map[string]string{"Authorization": "Bearer " + tok})
	assert.Equal(t, 200, w.Code)
	assert.True(t, called)
}

func TestRS256Auth_EmptyJTISkipsChecker(t *testing.T) {
	priv, _, pubPEM := testkit.NewRSAKeyPair(t)
	pub, err := auth.LoadRSAPublicKey(string(pubPEM), "")
	require.NoError(t, err)
	tok := signRS256(t, priv, auth.RSAClaims{RegisteredClaims: jwt.RegisteredClaims{
		Subject:   "u-1",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}})

	cc := &countingChecker{}
	h := newMiddleware().RS256Auth(auth.NewRSAVerifier(pub), middleware.RS256Options{Revocation: cc})(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "u-1", middleware.GetUserID(r))
			assert.Equal(t, "", middleware.GetRole(r))
		}))
	w := performRequest(h, http.MethodGet, "/", map[string]string{"Authorization": "Bearer " + tok})
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, int32(0), cc.calls.Load())
}

func TestRS256Auth_HungRevocation(t *testing.T) {
	priv, _, pubPEM := testkit.NewRSAKeyPair(t)
	pub, err := auth.LoadRSAPublicKey(string(pubPEM), "")
	require.NoError(t, err)
	tok, _, _, err := auth.NewRSASigner(priv, "test", time.Hour).Sign("u-1", "a@b.c", "investor")
	require.NoError(t, err)

	for _, failOpen := range []bool{false, true} {
		opts := middleware.RS256Options{
			Revocation:        hangingChecker{},
			FailOpen:          failOpen,
			RevocationTimeout: 100 * time.Millisecond,
		}
		h := newMiddleware().RS256Auth(auth.NewRSAVerifier(pub), opts)(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		start := time.Now()
		w := performRequest(h, http.MethodGet, "/", map[string]string{"Authorization": "Bearer " + tok})
		assert.Less(t, time.Since(start), 500*time.Millisecond)
		if failOpen {
			assert.Equal(t, 200, w.Code)
		} else {
			assert.Equal(t, 503, w.Code)
			assert.Equal(t, "5", w.Header().Get("Retry-After"))
		}
	}
}

func TestOptionalRS256Auth(t *testing.T) {
	priv, _, pubPEM := testkit.NewRSAKeyPair(t)
	pub, err := auth.LoadRSAPublicKey(string(pubPEM), "")
	require.NoError(t, err)
	verifier := auth.NewRSAVerifier(pub)
	valid, _, _, err := auth.NewRSASigner(priv, "test", time.Hour).Sign("u-1", "a@b.c", "investor")
	require.NoError(t, err)
	expired, _, _, err := auth.NewRSASigner(priv, "test", -time.Hour).Sign("u-1", "a@b.c", "investor")
	require.NoError(t, err)

	run := func(header string) (*httptest.ResponseRecorder, *auth.RSAClaims, bool) {
		var got *auth.RSAClaims
		var ok bool
		h := newMiddleware().OptionalRS256Auth(verifier, middleware.RS256Options{})(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got, ok = middleware.GetClaims(r)
			}))
		headers := map[string]string{}
		if header != "" {
			headers["Authorization"] = header
		}
		return performRequest(h, http.MethodGet, "/", headers), got, ok
	}

	w, c, ok := run("")
	assert.Equal(t, 200, w.Code)
	assert.Nil(t, c)
	assert.False(t, ok)

	w, c, ok = run("Bearer " + valid)
	assert.Equal(t, 200, w.Code)
	require.True(t, ok)
	assert.Equal(t, "a@b.c", c.Email)

	w, _, _ = run("Bearer " + expired)
	assert.Equal(t, 401, w.Code)
	assert.Equal(t, "token expired", errField(t, w))

	w, _, _ = run("Basic abc")
	assert.Equal(t, 401, w.Code)
	assert.Equal(t, "authentication required", errField(t, w))
}

func TestRS256Auth_WithRequireRole(t *testing.T) {
	priv, _, pubPEM := testkit.NewRSAKeyPair(t)
	pub, err := auth.LoadRSAPublicKey(string(pubPEM), "")
	require.NoError(t, err)
	tok, _, _, err := auth.NewRSASigner(priv, "test", time.Hour).Sign("u-1", "a@b.c", "investor")
	require.NoError(t, err)

	m := newMiddleware()
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	hdr := map[string]string{"Authorization": "Bearer " + tok}

	h := chain(ok, m.RS256Auth(auth.NewRSAVerifier(pub), middleware.RS256Options{}), m.RequireRole("admin"))
	assert.Equal(t, 403, performRequest(h, http.MethodGet, "/", hdr).Code)

	h = chain(ok, m.RS256Auth(auth.NewRSAVerifier(pub), middleware.RS256Options{}), m.RequireRole("investor"))
	assert.Equal(t, 200, performRequest(h, http.MethodGet, "/", hdr).Code)
}
