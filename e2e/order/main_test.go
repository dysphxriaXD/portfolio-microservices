// Package order содержит E2E-тесты OrderService.
//
// Тесты поднимают весь стек (order + его зависимости inventory и payment) как
// реальные контейнеры (testcontainers) и общаются с order по HTTP. order внутри
// docker-сети ходит к соседям сам. Нужен запущенный Docker.
package order

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/dysphxriaXD/e2e/testutil"
)

var (
	httpClient = &http.Client{Timeout: 5 * time.Second}
	baseURL    string
)

// orderBaseURL возвращает базовый URL order-сервиса для HTTP-тестов.
func orderBaseURL() string {
	return baseURL
}

// TestMain поднимает стек order'а в контейнерах перед тестами и гасит после.
func TestMain(m *testing.M) {
	ctx := context.Background()

	url, cleanup, err := testutil.StartOrderStack(ctx)
	if err != nil {
		slog.Error("не удалось поднять стек order", "error", err)
		os.Exit(1)
	}

	baseURL = url

	code := m.Run()

	cleanup()
	os.Exit(code)
}
