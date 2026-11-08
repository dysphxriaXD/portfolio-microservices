package main

import (
	"cmp"
	"log/slog"
	"net/http"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	orderHandler "github.com/dysphxriaXD/order/internal/handler"
	inventoryv1 "github.com/dysphxriaXD/shared/pkg/proto/inventory/v1"
	paymentv1 "github.com/dysphxriaXD/shared/pkg/proto/payment/v1"
)

func main() {
	// Адреса соседей приходят из окружения: E2E-тесты поднимают сервисы в
	// Docker-сети и передают INVENTORY_ADDRESS и PAYMENT_ADDRESS. Без переменных
	// остаётся localhost для запуска на своей машине.
	inventoryServiceAddress := cmp.Or(os.Getenv("INVENTORY_ADDRESS"), "localhost:50051")
	paymentServiceAddress := cmp.Or(os.Getenv("PAYMENT_ADDRESS"), "localhost:50052")

	// TODO: Настроить gRPC клиент с параметрами keepalive
	// Подумайте, какие параметры стоит задать для gRPC клиента
	// См. examples/week_1/GRPC_CONNECTIONS.md

	// Создать gRPC соединение с InventoryService
	inventoryConn, err := grpc.NewClient(inventoryServiceAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("не удалось подключиться к InventoryService", "error", err)
		os.Exit(1)
	}
	defer inventoryConn.Close()

	// TODO: Создать gRPC клиент PaymentService по адресу paymentServiceAddress

	// Создаём хранилище и обработчик
	store := orderHandler.NewOrderStore()
	h := orderHandler.NewHandler(
		inventoryv1.NewInventoryServiceClient(inventoryConn),
		paymentv1.NewPaymentServiceClient(paymentConn),
		store,
	)

	// TODO: Сгенерировать код ogen из OpenAPI спецификации
	// Команда: task ogen:gen

	// Создать OpenAPI сервер
	orderServer, err := orderHandler.SetupServer(h)
	if err != nil {
		slog.Error("ошибка создания сервера OpenAPI", "error", err)
		os.Exit(1)
	}

	// TODO: Настроить HTTP сервер с таймаутами
	// Создайте &http.Server{...} с явными таймаутами вместо http.ListenAndServe(...)
	// Минимальный набор: ReadHeaderTimeout (защита от Slowloris), ReadTimeout, WriteTimeout, IdleTimeout
	// Без ReadHeaderTimeout сервер уязвим к атаке Slowloris (медленная отправка заголовков)
	// См. examples/week_1/HTTP_SERVER.md

	// TODO: Реализовать graceful shutdown для HTTP сервера
	// При получении сигнала SIGINT/SIGTERM сервер должен:
	// 1. Перестать принимать новые соединения
	// 2. Дождаться завершения текущих запросов (с таймаутом)
	// 3. Закрыть gRPC соединения
	// 4. Корректно завершить работу
	// Подсказка: используйте signal.NotifyContext и httpServer.Shutdown(ctx)

	// Строку "HTTP сервер запущен" ждут E2E-тесты: по ней они понимают, что сервис
	// готов. Адрес с пустым хостом слушает все интерфейсы, иначе в контейнер не достучаться.
	slog.Info("HTTP сервер запущен", "address", ":8080")

	err = http.ListenAndServe(":8080", orderServer)
	if err != nil {
		slog.Error("ошибка запуска сервера", "error", err)
		os.Exit(1)
	}
}
