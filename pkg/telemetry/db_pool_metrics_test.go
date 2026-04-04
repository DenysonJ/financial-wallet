package telemetry

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric/noop"
)

func TestRegisterDBPoolMetrics(t *testing.T) {
	otel.SetMeterProvider(noop.NewMeterProvider())
	t.Cleanup(func() { otel.SetMeterProvider(nil) })

	tests := []struct {
		name     string
		nilDB    bool
		poolName string
		wantErr  bool
	}{
		{
			name:     "sucesso com DB válido",
			poolName: "writer",
		},
		{
			name:     "nil DB retorna nil sem erro",
			nilDB:    true,
			poolName: "writer",
		},
		{
			name:     "pool name alternativo",
			poolName: "reader",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.nilDB {
				regErr := RegisterDBPoolMetrics(context.Background(), "test-svc", nil, tt.poolName)
				assert.NoError(t, regErr)
				return
			}

			db, _, mockErr := sqlmock.New()
			require.NoError(t, mockErr)
			defer func() {
				err := db.Close()
				if err != nil {
					t.Errorf("failed to close mock DB connection: %v", err)
				}
			}()

			regErr := RegisterDBPoolMetrics(context.Background(), "test-svc", db, tt.poolName)

			if tt.wantErr {
				assert.Error(t, regErr)
			} else {
				assert.NoError(t, regErr)
			}
		})
	}
}
