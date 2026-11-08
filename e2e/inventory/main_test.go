// Package inventory содержит E2E-тесты InventoryService.
//
// Тесты поднимают inventory как реальный контейнер (testcontainers) и общаются
// с ним по настоящему gRPC. Нужен запущенный Docker.
package inventory

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/dysphxriaXD/e2e/testutil"
	inventoryv1 "github.com/dysphxriaXD/shared/pkg/proto/inventory/v1"
)

var inventoryClient inventoryv1.InventoryServiceClient

// TestMain поднимает inventory в контейнере перед тестами и гасит после.
func TestMain(m *testing.M) {
	ctx := context.Background()

	client, cleanup, err := testutil.StartInventory(ctx)
	if err != nil {
		slog.Error("не удалось поднять inventory", "error", err)
		os.Exit(1)
	}

	inventoryClient = client

	code := m.Run()

	cleanup()
	os.Exit(code)
}
