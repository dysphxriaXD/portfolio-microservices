// Package payment содержит E2E-тесты PaymentService.
//
// Тесты поднимают payment как реальный контейнер (testcontainers) и общаются с
// ним по настоящему gRPC. Нужен запущенный Docker.
package payment

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/dysphxriaXD/e2e/testutil"
	paymentv1 "github.com/dysphxriaXD/shared/pkg/proto/payment/v1"
)

var paymentClient paymentv1.PaymentServiceClient

// TestMain поднимает payment в контейнере перед тестами и гасит после.
func TestMain(m *testing.M) {
	ctx := context.Background()

	client, cleanup, err := testutil.StartPayment(ctx)
	if err != nil {
		slog.Error("не удалось поднять payment", "error", err)
		os.Exit(1)
	}

	paymentClient = client

	code := m.Run()

	cleanup()
	os.Exit(code)
}
