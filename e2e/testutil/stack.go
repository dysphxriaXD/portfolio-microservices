package testutil

import (
	"context"
	"fmt"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	inventoryv1 "github.com/dysphxriaXD/shared/pkg/proto/inventory/v1"
	paymentv1 "github.com/dysphxriaXD/shared/pkg/proto/payment/v1"
)

// buildContext: корень проекта относительно каталога тест-пакета
// (e2e/<service>/), откуда go test запускает тесты: рабочая директория = каталог
// пакета, поэтому до корня проекта два уровня вверх.
const buildContext = "../.."

// dockerfilePath: единый Dockerfile на все сервисы (выбор через build-arg SERVICE),
// путь относительно buildContext.
const dockerfilePath = "deploy/docker/Dockerfile"

const startupTimeout = 120 * time.Second

// serviceSpec описывает один сервис для подъёма в контейнере.
type serviceSpec struct {
	name     string            // имя = сетевой алиас + build-arg SERVICE (inventory/payment/order)
	port     string            // внутренний порт контейнера, напр. "50051/tcp"
	readyLog string            // подстрока в логах, означающая готовность
	env      map[string]string // переменные окружения (для order: адреса соседей)
}

var (
	inventorySpec = serviceSpec{
		name:     "inventory",
		port:     "50051/tcp",
		readyLog: "gRPC сервер запущен",
	}
	paymentSpec = serviceSpec{
		name:     "payment",
		port:     "50052/tcp",
		readyLog: "gRPC сервер запущен",
	}
	orderSpec = serviceSpec{
		name:     "order",
		port:     "8080/tcp",
		readyLog: "HTTP сервер запущен",
		env: map[string]string{
			"INVENTORY_ADDRESS": "inventory:50051",
			"PAYMENT_ADDRESS":   "payment:50052",
		},
	}
)

// StartInventory поднимает только inventory и возвращает gRPC-клиент к нему.
// Возвращаемая функция гасит контейнер и сеть (вызывать после прогона тестов).
func StartInventory(ctx context.Context) (inventoryv1.InventoryServiceClient, func(), error) {
	td := &teardown{}

	nw, err := newNetwork(ctx, td)
	if err != nil {
		return nil, nil, err
	}

	conn, err := startGRPCService(ctx, td, nw.Name, inventorySpec)
	if err != nil {
		td.run()
		return nil, nil, err
	}

	return inventoryv1.NewInventoryServiceClient(conn), td.run, nil
}

// StartPayment поднимает только payment и возвращает gRPC-клиент к нему.
func StartPayment(ctx context.Context) (paymentv1.PaymentServiceClient, func(), error) {
	td := &teardown{}

	nw, err := newNetwork(ctx, td)
	if err != nil {
		return nil, nil, err
	}

	conn, err := startGRPCService(ctx, td, nw.Name, paymentSpec)
	if err != nil {
		td.run()
		return nil, nil, err
	}

	return paymentv1.NewPaymentServiceClient(conn), td.run, nil
}

// StartOrderStack поднимает order вместе с его зависимостями (inventory, payment)
// в одной сети и возвращает базовый HTTP-URL order'а. Зависимости тестам напрямую
// не нужны: order ходит к ним сам по сети, поэтому наружу отдаётся только URL.
func StartOrderStack(ctx context.Context) (string, func(), error) {
	td := &teardown{}

	nw, err := newNetwork(ctx, td)
	if err != nil {
		return "", nil, err
	}

	if err = startContainer(ctx, td, nw.Name, inventorySpec); err != nil {
		td.run()
		return "", nil, err
	}

	if err = startContainer(ctx, td, nw.Name, paymentSpec); err != nil {
		td.run()
		return "", nil, err
	}

	orderC, err := startService(ctx, nw.Name, orderSpec)
	if err != nil {
		td.run()
		return "", nil, fmt.Errorf("start order: %w", err)
	}
	td.add(func() { _ = orderC.Terminate(context.Background()) })

	url, err := httpBaseURL(ctx, orderC, orderSpec.port)
	if err != nil {
		td.run()
		return "", nil, fmt.Errorf("order url: %w", err)
	}

	return url, td.run, nil
}

// teardown копит функции очистки и выполняет их в обратном порядке.
// Использует context.Background(): переданный в Start* ctx может быть уже отменён
// к моменту очистки, а контейнеры/сеть всё равно надо погасить.
type teardown struct {
	funcs []func()
}

func (t *teardown) add(f func()) { t.funcs = append(t.funcs, f) }

func (t *teardown) run() {
	for i := len(t.funcs) - 1; i >= 0; i-- {
		t.funcs[i]()
	}
}

func newNetwork(ctx context.Context, td *teardown) (*testcontainers.DockerNetwork, error) {
	nw, err := network.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("create network: %w", err)
	}
	td.add(func() { _ = nw.Remove(context.Background()) })

	return nw, nil
}

// startContainer поднимает контейнер сервиса в сети и регистрирует его остановку.
func startContainer(ctx context.Context, td *teardown, networkName string, spec serviceSpec) error {
	c, err := startService(ctx, networkName, spec)
	if err != nil {
		return fmt.Errorf("start %s: %w", spec.name, err)
	}
	td.add(func() { _ = c.Terminate(context.Background()) })

	return nil
}

// startGRPCService поднимает gRPC-сервис в сети и возвращает соединение к нему.
func startGRPCService(ctx context.Context, td *teardown, networkName string, spec serviceSpec) (*grpc.ClientConn, error) {
	c, err := startService(ctx, networkName, spec)
	if err != nil {
		return nil, fmt.Errorf("start %s: %w", spec.name, err)
	}
	td.add(func() { _ = c.Terminate(context.Background()) })

	conn, err := grpcConn(ctx, c, spec.port)
	if err != nil {
		return nil, fmt.Errorf("%s conn: %w", spec.name, err)
	}
	td.add(func() { _ = conn.Close() })

	return conn, nil
}

func startService(ctx context.Context, networkName string, spec serviceSpec) (testcontainers.Container, error) {
	service := spec.name
	req := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context:    buildContext,
			Dockerfile: dockerfilePath,
			BuildArgs:  map[string]*string{"SERVICE": &service},
			// Стабильный тег: без Repo/Tag testcontainers генерит случайный тег на
			// каждый билд, а KeepImage копит их орфанами (диск забивается за десяток
			// прогонов). Со стабильным тегом образ переиспользуется и не копится.
			Repo:      "week1-e2e-" + service,
			Tag:       "latest",
			KeepImage: true,
		},
		ExposedPorts:   []string{spec.port},
		Networks:       []string{networkName},
		NetworkAliases: map[string][]string{networkName: {spec.name}},
		Env:            spec.env,
		WaitingFor:     wait.ForLog(spec.readyLog).WithStartupTimeout(startupTimeout),
	}

	return testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
}

func grpcConn(ctx context.Context, c testcontainers.Container, port string) (*grpc.ClientConn, error) {
	host, err := c.Host(ctx)
	if err != nil {
		return nil, err
	}

	mapped, err := c.MappedPort(ctx, port)
	if err != nil {
		return nil, err
	}

	return grpc.NewClient(
		fmt.Sprintf("%s:%s", host, mapped.Port()),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
}

func httpBaseURL(ctx context.Context, c testcontainers.Container, port string) (string, error) {
	host, err := c.Host(ctx)
	if err != nil {
		return "", err
	}

	mapped, err := c.MappedPort(ctx, port)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("http://%s:%s", host, mapped.Port()), nil
}
