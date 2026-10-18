package bootstrap

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/piresc/gokit/config"
)

func TestMigrationURL_EscapesCredentials(t *testing.T) {
	cfg := config.DatabaseConfig{
		Host:     "db.local",
		Port:     5432,
		Username: "us@er",
		Password: "p@ss/w:rd",
		Database: "appdb",
		SSLMode:  "disable",
	}

	u, err := url.Parse(migrationURL(cfg))
	require.NoError(t, err)

	pw, ok := u.User.Password()
	require.True(t, ok)
	assert.Equal(t, "p@ss/w:rd", pw)
	assert.Equal(t, "us@er", u.User.Username())
	assert.Equal(t, "db.local:5432", u.Host)
	assert.Equal(t, "/appdb", u.Path)
	assert.Equal(t, "disable", u.Query().Get("sslmode"))
}
