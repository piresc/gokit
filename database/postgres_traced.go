package database

import (
	"github.com/jmoiron/sqlx"

	"github.com/piresc/gokit/tracing"
)

// TracedPostgresClient wraps PostgresClient.
//
// Deprecated: since v0.5.0 the clients from NewPostgresClient / NewClient create spans themselves.
type TracedPostgresClient struct {
	*PostgresClient
	tracer tracing.Tracer
}

// NewTracedPostgresClient wraps client.
//
// Deprecated: since v0.5.0 the clients from NewPostgresClient / NewClient create spans themselves.
func NewTracedPostgresClient(client *PostgresClient, tracer tracing.Tracer) *TracedPostgresClient {
	return &TracedPostgresClient{
		PostgresClient: client,
		tracer:         tracer,
	}
}

func (t *TracedPostgresClient) GetDB() *sqlx.DB {
	return t.PostgresClient.GetDB()
}

func (t *TracedPostgresClient) Close() error {
	return t.PostgresClient.Close()
}
