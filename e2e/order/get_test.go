package order

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dysphxriaXD/e2e/testutil"
)

// Тесты OrderService (HTTP): получение заказа

func TestOrder_Get_Success(t *testing.T) {
	// Сначала создаём заказ
	createReq := &CreateOrderRequest{
		HullUUID:   testutil.HullAluminumUUID,
		EngineUUID: testutil.EngineIonCUUID,
	}
	createResult, createResp := createOrder(t, createReq)
	defer createResp.Body.Close()
	require.NotNil(t, createResult)

	// Получаем заказ
	order, resp := getOrder(t, createResult.OrderUUID)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.NotNil(t, order)
	assert.Equal(t, createResult.OrderUUID, order.OrderUUID)
	assert.Equal(t, testutil.HullAluminumUUID, order.HullUUID)
	assert.Equal(t, testutil.EngineIonCUUID, order.EngineUUID)
	assert.Equal(t, createResult.TotalPrice, order.TotalPrice)
}

func TestOrder_Get_VerifyStatus_PendingPayment(t *testing.T) {
	// Создаём заказ
	createReq := &CreateOrderRequest{
		HullUUID:   testutil.HullAluminumUUID,
		EngineUUID: testutil.EngineIonCUUID,
	}
	createResult, createResp := createOrder(t, createReq)
	defer createResp.Body.Close()
	require.NotNil(t, createResult)

	// Получаем и проверяем статус
	order, resp := getOrder(t, createResult.OrderUUID)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "PENDING_PAYMENT", order.Status)
}

func TestOrder_Get_NotFound(t *testing.T) {
	_, resp := getOrder(t, uuid.New().String())
	defer resp.Body.Close()

	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestOrder_Get_WithOptionalParts(t *testing.T) {
	shieldUUID := testutil.ShieldEnergyUUID
	weaponUUID := testutil.WeaponLaserUUID
	req := &CreateOrderRequest{
		HullUUID:   testutil.HullAluminumUUID,
		EngineUUID: testutil.EngineIonCUUID,
		ShieldUUID: &shieldUUID,
		WeaponUUID: &weaponUUID,
	}

	createResult, createResp := createOrder(t, req)
	defer createResp.Body.Close()
	require.NotNil(t, createResult)

	// Получаем заказ и проверяем, что опциональные детали сохранены
	order, resp := getOrder(t, createResult.OrderUUID)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.NotNil(t, order.ShieldUUID)
	require.NotNil(t, order.WeaponUUID)
	assert.Equal(t, shieldUUID, *order.ShieldUUID)
	assert.Equal(t, weaponUUID, *order.WeaponUUID)
}

func TestOrder_Get_InvalidUUIDInPath(t *testing.T) {
	resp, err := httpClient.Get(orderBaseURL() + "/api/v1/orders/not-a-uuid")
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestOrder_Get_NewOrder_OptionalFieldsEmpty(t *testing.T) {
	// Создаём заказ без опциональных деталей
	createReq := &CreateOrderRequest{
		HullUUID:   testutil.HullAluminumUUID,
		EngineUUID: testutil.EngineIonCUUID,
	}
	createResult, createResp := createOrder(t, createReq)
	defer createResp.Body.Close()
	require.NotNil(t, createResult)

	// До оплаты и без щита с вооружением эти поля пусты
	order, resp := getOrder(t, createResult.OrderUUID)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Nil(t, order.ShieldUUID, "shield_uuid не передавали при создании")
	assert.Nil(t, order.WeaponUUID, "weapon_uuid не передавали при создании")
	assert.Nil(t, order.TransactionUUID, "transaction_uuid появляется только после оплаты")
	assert.Nil(t, order.PaymentMethod, "payment_method появляется только после оплаты")
}

func TestOrder_Get_VerifyCreatedAt(t *testing.T) {
	// Создаём заказ
	createReq := &CreateOrderRequest{
		HullUUID:   testutil.HullAluminumUUID,
		EngineUUID: testutil.EngineIonCUUID,
	}
	createResult, createResp := createOrder(t, createReq)
	defer createResp.Body.Close()
	require.NotNil(t, createResult)

	order, resp := getOrder(t, createResult.OrderUUID)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.NotEmpty(t, order.CreatedAt, "created_at должен быть заполнен")

	// Парсим время: строка должна быть валидной датой, а время не нулевым
	createdAt, err := time.Parse(time.RFC3339Nano, order.CreatedAt)
	require.NoError(t, err, "не удалось распарсить created_at: %s", order.CreatedAt)
	assert.False(t, createdAt.IsZero(), "created_at не должен быть нулевым")
}
