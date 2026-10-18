package testkit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/piresc/gokit/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGenerateToken_ShouldReturnNonEmptyToken_WhenValidInputProvided
func TestGenerateToken_ShouldReturnNonEmptyToken_WhenValidInputProvided(t *testing.T) {
	// Arrange
	userID := "user-001"
	role := "driver"
	secret := "test-secret-key-for-jwt-testing-purposes-only"

	// Act
	token := GenerateToken(userID, role, secret)

	// Assert
	assert.NotEmpty(t, token, "generated token should not be empty")
}

// TestGenerateToken_ShouldReturnValidatableToken_WhenValidInputProvided
func TestGenerateToken_ShouldReturnValidatableToken_WhenValidInputProvided(t *testing.T) {
	// Arrange
	userID := "user-002"
	role := "admin"
	secret := "test-secret-key-for-jwt-testing-purposes-only"

	// Act
	token := GenerateToken(userID, role, secret)

	// Assert
	claims, err := auth.ValidateToken(token, secret)
	require.NoError(t, err, "token should be valid and parseable")
	assert.NotNil(t, claims, "claims should not be nil")
}

// TestGenerateToken_ShouldContainCorrectClaims_WhenValidInputProvided
func TestGenerateToken_ShouldContainCorrectClaims_WhenValidInputProvided(t *testing.T) {
	// Arrange
	userID := "user-003"
	role := "operator"
	secret := "test-secret-key-for-jwt-testing-purposes-only"

	// Act
	token := GenerateToken(userID, role, secret)

	// Assert
	claims, err := auth.ValidateToken(token, secret)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID, "userID claim should match input")
	assert.Equal(t, role, claims.Role, "role claim should match input")
	assert.Equal(t, "test", claims.Issuer, "issuer should be 'test'")
}

// TestAssertError_ShouldPass_WhenErrorContainsSubstring
func TestAssertError_ShouldPass_WhenErrorContainsSubstring(t *testing.T) {
	t.Run("should not fail when error contains substring", func(t *testing.T) {
		err := assert.AnError // just use a generic error
		AssertError(t, err, "assert")
		// If this test passes, AssertError didn't call t.Errorf
	})
}

// TestAssertNoError_ShouldPass_WhenErrorIsNil
func TestAssertNoError_ShouldPass_WhenErrorIsNil(t *testing.T) {
	AssertNoError(t, nil)
	// If this reaches here, AssertNoError passed
}

// TestNewHTTPClient_ShouldConstructClient
func TestNewHTTPClient_ShouldConstructClient(t *testing.T) {
	// Arrange
	baseURL := "http://localhost:8080"

	// Act
	client := NewHTTPClient(baseURL)

	// Assert
	assert.NotNil(t, client, "client should not be nil")
	assert.Equal(t, baseURL, client.baseURL, "baseURL should be set")
}

// TestHTTPClient_WithToken_ShouldSetToken
func TestHTTPClient_WithToken_ShouldSetToken(t *testing.T) {
	// Arrange
	client := NewHTTPClient("http://localhost:8080")
	token := "test-token-123"

	// Act
	result := client.WithToken(token)

	// Assert
	assert.Equal(t, token, result.token, "token should be set")
	assert.Equal(t, client, result, "should return same client for chaining")
}

// TestHTTPClient_Do_ShouldAddAuthorizationHeader_WhenTokenIsSet
func TestHTTPClient_Do_ShouldAddAuthorizationHeader_WhenTokenIsSet(t *testing.T) {
	// Arrange
	token := "test-jwt-token-abc123"
	var receivedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL).WithToken(token)

	// Act
	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	require.NoError(t, err)
	resp, err := client.Do(req)

	// Assert
	require.NoError(t, err, "request should succeed")
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "Bearer test-jwt-token-abc123", receivedAuth, "Authorization header should contain Bearer token")
}

// TestHTTPClient_AssertStatus_ShouldSucceed_WhenStatusMatches
func TestHTTPClient_AssertStatus_ShouldSucceed_WhenStatusMatches(t *testing.T) {
	// Arrange
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL)

	// Act
	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	require.NoError(t, err)
	resp, err := client.AssertStatus(req, http.StatusOK)

	// Assert
	require.NoError(t, err, "should succeed when status matches")
	_ = resp.Body.Close()
}

// TestHTTPClient_AssertStatus_ShouldFail_WhenStatusDoesNotMatch
func TestHTTPClient_AssertStatus_ShouldFail_WhenStatusDoesNotMatch(t *testing.T) {
	// Arrange
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL)

	// Act
	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	require.NoError(t, err)
	resp, err := client.AssertStatus(req, http.StatusOK)

	// Assert
	assert.Error(t, err, "should return error when status does not match")
	assert.Nil(t, resp, "response should be nil on error")
	assert.Contains(t, err.Error(), "expected status 200", "error should mention expected status")
}

// TestHTTPClient_ParseJSON_ShouldUnmarshalJSON
func TestHTTPClient_ParseJSON_ShouldUnmarshalJSON(t *testing.T) {
	// Arrange
	type TestData struct {
		Message string `json:"message"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":"hello"}`))
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL)

	// Act
	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)

	var data TestData
	err = client.ParseJSON(resp, &data)

	// Assert
	require.NoError(t, err, "should unmarshal JSON without error")
	assert.Equal(t, "hello", data.Message, "should parse message field correctly")
}

// TestWaitForHealth_ShouldReturnNil_WhenServerReturns200
func TestWaitForHealth_ShouldReturnNil_WhenServerReturns200(t *testing.T) {
	// Arrange
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Act
	err := WaitForHealth(server.URL, 5*time.Second)

	// Assert
	assert.NoError(t, err, "should succeed when server returns 200")
}

// TestWaitForHealth_ShouldReturnError_WhenServerIsUnavailable
func TestWaitForHealth_ShouldReturnError_WhenServerIsUnavailable(t *testing.T) {
	// Arrange — use a URL that nothing is listening on
	url := "http://127.0.0.1:19999"

	// Act
	err := WaitForHealth(url, 2*time.Second)

	// Assert
	assert.Error(t, err, "should return error when server is unreachable")
	assert.Contains(t, err.Error(), "timed out", "error should indicate timeout")
}

// TestNewRSAKeyPair_ShouldReturnParseableKeys
func TestNewRSAKeyPair_ShouldReturnParseableKeys(t *testing.T) {
	priv, privPEM, pubPEM := NewRSAKeyPair(t)

	gotPriv, err := auth.LoadRSAPrivateKey(string(privPEM), "")
	require.NoError(t, err)
	assert.True(t, gotPriv.Equal(priv))

	gotPub, err := auth.LoadRSAPublicKey(string(pubPEM), "")
	require.NoError(t, err)
	assert.True(t, gotPub.Equal(&priv.PublicKey))
	assert.Equal(t, 2048, priv.N.BitLen())
}
