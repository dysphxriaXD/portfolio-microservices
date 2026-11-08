# Образ нужен только для интеграционных тестов (E2E на testcontainers).
# Контейнеризацию подробно разбираем на неделе 8 — здесь можно не вникать.
#
# Один Dockerfile на все сервисы: build-arg SERVICE выбирает, какой cmd собрать
# (inventory / payment / order). Контекст сборки = корень проекта
# (нужны go.work и все модули).
#
# Builder на alpine: минимальная база (на порядок меньше CVE, чем у debian-golang),
# а на статический бинарь (CGO_ENABLED=0) musl всё равно не влияет. Эти CVE и так
# остаются в builder-стадии и в финальный distroless-образ не попадают.
FROM golang:1.27.2-alpine AS builder
WORKDIR /src

# Сначала только манифесты модулей: слой со скачанными зависимостями
# переиспользуется, пока не менялись go.mod/go.sum (правки кода сервисов его
# не инвалидируют). Этот слой общий для всех значений SERVICE — модули
# скачиваются один раз на все три образа.
COPY go.work go.work.sum ./
COPY order/go.mod order/go.sum ./order/
COPY inventory/go.mod inventory/go.sum ./inventory/
COPY payment/go.mod payment/go.sum ./payment/
COPY shared/go.mod shared/go.sum ./shared/
COPY e2e/go.mod e2e/go.sum ./e2e/
RUN go mod download

# Теперь исходники и сборка выбранного сервиса.
# -ldflags="-s -w" срезает debug-таблицы → меньше бинарь.
COPY . .
ARG SERVICE
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/app ./${SERVICE}/cmd

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /bin/app /bin/app
ENTRYPOINT ["/bin/app"]
