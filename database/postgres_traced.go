package database

import (
	"github.com/jmoiron/sqlx"

	"github.com/piresc/gokit/tracing"
)

type TracedPostgresClient struct {
	*PostgresClient
	tracer tracing.Tracer
}

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
