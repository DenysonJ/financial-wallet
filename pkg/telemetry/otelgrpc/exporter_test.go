package otelgrpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExporters_ReturnsOptions(t *testing.T) {
	opts, exportErr := Exporters(context.Background(), Config{
		CollectorURL: "localhost:4317",
		Insecure:     true,
	})
	require.NoError(t, exportErr)
	assert.Len(t, opts, 2)
}

func TestExporters_Insecure(t *testing.T) {
	opts, exportErr := Exporters(context.Background(), Config{
		CollectorURL: "collector.example.com:4317",
		Insecure:     true,
	})
	require.NoError(t, exportErr)
	assert.Len(t, opts, 2)
}

func TestExporters_Secure(t *testing.T) {
	opts, exportErr := Exporters(context.Background(), Config{
		CollectorURL: "collector.example.com:4317",
		Insecure:     false,
	})
	require.NoError(t, exportErr)
	assert.Len(t, opts, 2)
}
