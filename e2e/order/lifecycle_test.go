package order

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dysphxriaXD/e2e/testutil"
)

// Тесты полного жизненного цикла

func TestOrder_FullLifecycle_CreatePayGet(t *testing.T) {
	// 1. Создаём заказ
	createReq := &CreateOrderRequest{
		HullUUID:   testutil.HullTitaniumUUID,
		EngineUUID: testutil.EngineIonBUUID,
		ShieldUUID: new(testutil.ShieldEnergyUUID),
	}
	createResult, createResp := createOrder(t, createReq)
	defer createResp.Body.Close()
	require.NotNil(t, createResult)
	assert.NotEmpty(t, createResult.OrderUUID)

	expectedTotal := int64(testutil.HullTitaniumPrice + testutil.EngineIonBPrice + testutil.ShieldEnergyPrice)
	assert.Equal(t, expectedTotal, createResult.TotalPrice)

	// 2. Получаем заказ (проверяем PENDING_PAYMENT)
	order1, getResp1 := getOrder(t, createResult.OrderUUID)
	defer getResp1.Body.Close()
	require.NotNil(t, order1, "заказ не получен: GET /api/v1/orders/{order_uuid} вернул %d вместо 200", getResp1.StatusCode)
	assert.Equal(t, "PENDING_PAYMENT", order1.Status)
	assert.Nil(t, order1.TransactionUUID)

	// 3. Оплачиваем заказ
	payReq := &PayOrderRequest{PaymentMethod: "SBP"}
	payResult, payResp := payOrder(t, createResult.OrderUUID, payReq)
	defer payResp.Body.Close()
	require.NotNil(t, payResult)
	assert.NotEmpty(t, payResult.TransactionUUID)

	// 4. Получаем заказ (проверяем PAID)
	order2, getResp2 := getOrder(t, createResult.OrderUUID)
	defer getResp2.Body.Close()
	require.NotNil(t, order2, "заказ не получен: GET /api/v1/orders/{order_uuid} вернул %d вместо 200", getResp2.StatusCode)

	assert.Equal(t, "PAID", order2.Status)
	require.NotNil(t, order2.TransactionUUID)
	assert.Equal(t, payResult.TransactionUUID, *order2.TransactionUUID)
	require.NotNil(t, order2.PaymentMethod)
	assert.Equal(t, "SBP", *order2.PaymentMethod)
}

func TestOrder_FullLifecycle_CreateCancelGet(t *testing.T) {
	// 1. Создаём заказ
	createReq := &CreateOrderRequest{
		HullUUID:   testutil.HullAluminumUUID,
		EngineUUID: testutil.EngineIonCUUID,
	}
	createResult, createResp := createOrder(t, createReq)
	defer createResp.Body.Close()
	require.NotNil(t, createResult)

	// 2. Получаем заказ (проверяем PENDING_PAYMENT)
	order1, getResp1 := getOrder(t, createResult.OrderUUID)
	defer getResp1.Body.Close()
	require.NotNil(t, order1, "заказ не получен: GET /api/v1/orders/{order_uuid} вернул %d вместо 200", getResp1.StatusCode)
	assert.Equal(t, "PENDING_PAYMENT", order1.Status)

	// 3. Отменяем заказ
	_, cancelResp := cancelOrder(t, createResult.OrderUUID)
	defer cancelResp.Body.Close()

	// 4. Получаем заказ (проверяем CANCELLED)
	order2, getResp2 := getOrder(t, createResult.OrderUUID)
	defer getResp2.Body.Close()
	require.NotNil(t, order2, "заказ не получен: GET /api/v1/orders/{order_uuid} вернул %d вместо 200", getResp2.StatusCode)

	assert.Equal(t, "CANCELLED", order2.Status)
	assert.Nil(t, order2.TransactionUUID)
}

func TestOrder_FullLifecycle_AllPartsPayGet(t *testing.T) {
	// Полный жизненный цикл со всеми 4 деталями: hull + engine + shield + weapon
	shieldUUID := testutil.ShieldEnergyUUID
	weaponUUID := testutil.WeaponLaserUUID
	createReq := &CreateOrderRequest{
		HullUUID:   testutil.HullTitaniumUUID,
		EngineUUID: testutil.EngineIonBUUID,
		ShieldUUID: &shieldUUID,
		WeaponUUID: &weaponUUID,
	}

	// 1. Создаём заказ
	createResult, createResp := createOrder(t, createReq)
	defer createResp.Body.Close()
	require.NotNil(t, createResult)

	expectedTotal := int64(testutil.HullTitaniumPrice + testutil.EngineIonBPrice + testutil.ShieldEnergyPrice + testutil.WeaponLaserPrice)
	assert.Equal(t, expectedTotal, createResult.TotalPrice)

	// 2. Проверяем все детали в GET ответе
	order1, getResp1 := getOrder(t, createResult.OrderUUID)
	defer getResp1.Body.Close()
	require.NotNil(t, order1, "заказ не получен: GET /api/v1/orders/{order_uuid} вернул %d вместо 200", getResp1.StatusCode)
	assert.Equal(t, testutil.HullTitaniumUUID, order1.HullUUID)
	assert.Equal(t, testutil.EngineIonBUUID, order1.EngineUUID)
	require.NotNil(t, order1.ShieldUUID)
	assert.Equal(t, shieldUUID, *order1.ShieldUUID)
	require.NotNil(t, order1.WeaponUUID)
	assert.Equal(t, weaponUUID, *order1.WeaponUUID)

	// 3. Оплачиваем заказ
	payReq := &PayOrderRequest{PaymentMethod: "CREDIT_CARD"}
	payResult, payResp := payOrder(t, createResult.OrderUUID, payReq)
	defer payResp.Body.Close()
	require.NotNil(t, payResult)

	// 4. Проверяем финальное состояние
	order2, getResp2 := getOrder(t, createResult.OrderUUID)
	defer getResp2.Body.Close()
	require.NotNil(t, order2, "заказ не получен: GET /api/v1/orders/{order_uuid} вернул %d вместо 200", getResp2.StatusCode)

	assert.Equal(t, "PAID", order2.Status)
	require.NotNil(t, order2.PaymentMethod)
	assert.Equal(t, "CREDIT_CARD", *order2.PaymentMethod)
}
