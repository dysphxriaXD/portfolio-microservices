package order

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dysphxriaXD/e2e/testutil"
)

// Тесты OrderService (HTTP): создание заказа

func TestOrder_Create_Success_MinimalParts(t *testing.T) {
	req := &CreateOrderRequest{
		HullUUID:   testutil.HullAluminumUUID,
		EngineUUID: testutil.EngineIonCUUID,
	}

	result, resp := createOrder(t, req)
	defer resp.Body.Close()

	require.Equal(t, http.StatusCreated, resp.StatusCode)
	require.NotNil(t, result)
	assert.NotEmpty(t, result.OrderUUID)
	assert.Equal(t, int64(testutil.HullAluminumPrice+testutil.EngineIonCPrice), result.TotalPrice)

	// Проверяем, что UUID заказа валиден
	_, err := uuid.Parse(result.OrderUUID)
	assert.NoError(t, err, "order_uuid должен быть в формате UUID: %s", result.OrderUUID)
}

func TestOrder_Create_Success_AllParts(t *testing.T) {
	req := &CreateOrderRequest{
		HullUUID:   testutil.HullTitaniumUUID,
		EngineUUID: testutil.EngineIonBUUID,
		ShieldUUID: new(testutil.ShieldEnergyUUID),
		WeaponUUID: new(testutil.WeaponLaserUUID),
	}

	result, resp := createOrder(t, req)
	defer resp.Body.Close()

	require.Equal(t, http.StatusCreated, resp.StatusCode)
	require.NotNil(t, result)
	assert.NotEmpty(t, result.OrderUUID)

	expectedTotal := int64(testutil.HullTitaniumPrice + testutil.EngineIonBPrice + testutil.ShieldEnergyPrice + testutil.WeaponLaserPrice)
	assert.Equal(t, expectedTotal, result.TotalPrice)
}

func TestOrder_Create_VerifyTotalPrice(t *testing.T) {
	req := &CreateOrderRequest{
		HullUUID:   testutil.HullAluminumUUID, // 500000
		EngineUUID: testutil.EngineIonCUUID,   // 300000
	}

	result, resp := createOrder(t, req)
	defer resp.Body.Close()

	require.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Equal(t, int64(800000), result.TotalPrice, "500000 + 300000 = 800000")
}

func TestOrder_Create_HullNotFound(t *testing.T) {
	req := &CreateOrderRequest{
		HullUUID:   uuid.New().String(),
		EngineUUID: testutil.EngineIonCUUID,
	}

	_, resp := createOrder(t, req)
	defer resp.Body.Close()

	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestOrder_Create_EngineNotFound(t *testing.T) {
	req := &CreateOrderRequest{
		HullUUID:   testutil.HullAluminumUUID,
		EngineUUID: uuid.New().String(),
	}

	_, resp := createOrder(t, req)
	defer resp.Body.Close()

	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestOrder_Create_ShieldNotFound(t *testing.T) {
	req := &CreateOrderRequest{
		HullUUID:   testutil.HullAluminumUUID,
		EngineUUID: testutil.EngineIonCUUID,
		ShieldUUID: new(uuid.New().String()),
	}

	_, resp := createOrder(t, req)
	defer resp.Body.Close()

	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestOrder_Create_WeaponNotFound(t *testing.T) {
	req := &CreateOrderRequest{
		HullUUID:   testutil.HullAluminumUUID,
		EngineUUID: testutil.EngineIonCUUID,
		WeaponUUID: new(uuid.New().String()),
	}

	_, resp := createOrder(t, req)
	defer resp.Body.Close()

	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestOrder_Create_WithWeaponOnly(t *testing.T) {
	req := &CreateOrderRequest{
		HullUUID:   testutil.HullAluminumUUID,
		EngineUUID: testutil.EngineIonCUUID,
		WeaponUUID: new(testutil.WeaponLaserUUID),
	}

	result, resp := createOrder(t, req)
	defer resp.Body.Close()

	require.Equal(t, http.StatusCreated, resp.StatusCode)
	require.NotNil(t, result)
	expectedTotal := int64(testutil.HullAluminumPrice + testutil.EngineIonCPrice + testutil.WeaponLaserPrice)
	assert.Equal(t, expectedTotal, result.TotalPrice)
}

// Тесты отсутствия на складе (StockQuantity <= 0)

func TestOrder_Create_OutOfStock(t *testing.T) {
	req := &CreateOrderRequest{
		HullUUID:   testutil.HullOutOfStockUUID,
		EngineUUID: testutil.EngineIonCUUID,
	}

	_, resp := createOrder(t, req)
	defer resp.Body.Close()

	require.Equal(t, http.StatusConflict, resp.StatusCode)
}

// Тесты ogen-валидации (400 Bad Request)

func TestOrder_Create_InvalidBody_EmptyJSON(t *testing.T) {
	httpReq, err := http.NewRequest(http.MethodPost, orderBaseURL()+"/api/v1/orders", bytes.NewReader([]byte("{}")))
	require.NoError(t, err)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestOrder_Create_InvalidBody_NotJSON(t *testing.T) {
	httpReq, err := http.NewRequest(http.MethodPost, orderBaseURL()+"/api/v1/orders", bytes.NewReader([]byte("not json")))
	require.NoError(t, err)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestOrder_Create_InvalidBody_MissingHullUUID(t *testing.T) {
	body := `{"engine_uuid": "` + testutil.EngineIonCUUID + `"}`
	httpReq, err := http.NewRequest(http.MethodPost, orderBaseURL()+"/api/v1/orders", bytes.NewReader([]byte(body)))
	require.NoError(t, err)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestOrder_Create_InvalidBody_MissingEngineUUID(t *testing.T) {
	body := `{"hull_uuid": "` + testutil.HullAluminumUUID + `"}`
	httpReq, err := http.NewRequest(http.MethodPost, orderBaseURL()+"/api/v1/orders", bytes.NewReader([]byte(body)))
	require.NoError(t, err)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestOrder_Create_InvalidBody_InvalidHullUUID(t *testing.T) {
	body := `{"hull_uuid": "not-a-uuid", "engine_uuid": "` + testutil.EngineIonCUUID + `"}`
	httpReq, err := http.NewRequest(http.MethodPost, orderBaseURL()+"/api/v1/orders", bytes.NewReader([]byte(body)))
	require.NoError(t, err)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestOrder_Create_InvalidBody_InvalidEngineUUID(t *testing.T) {
	body := `{"hull_uuid": "` + testutil.HullAluminumUUID + `", "engine_uuid": "not-a-uuid"}`
	httpReq, err := http.NewRequest(http.MethodPost, orderBaseURL()+"/api/v1/orders", bytes.NewReader([]byte(body)))
	require.NoError(t, err)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestOrder_Create_InvalidBody_InvalidShieldUUID(t *testing.T) {
	body := `{"hull_uuid": "` + testutil.HullAluminumUUID + `", "engine_uuid": "` + testutil.EngineIonCUUID + `", "shield_uuid": "not-a-uuid"}`
	httpReq, err := http.NewRequest(http.MethodPost, orderBaseURL()+"/api/v1/orders", bytes.NewReader([]byte(body)))
	require.NoError(t, err)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestOrder_Create_InvalidBody_InvalidWeaponUUID(t *testing.T) {
	body := `{"hull_uuid": "` + testutil.HullAluminumUUID + `", "engine_uuid": "` + testutil.EngineIonCUUID + `", "weapon_uuid": "not-a-uuid"}`
	httpReq, err := http.NewRequest(http.MethodPost, orderBaseURL()+"/api/v1/orders", bytes.NewReader([]byte(body)))
	require.NoError(t, err)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// Тесты с опциональными деталями

func TestOrder_Create_Success_NullOptionalParts(t *testing.T) {
	// Опциональные детали переданы явным null, как в примере запроса из контракта
	body := `{"hull_uuid": "` + testutil.HullAluminumUUID + `", "engine_uuid": "` + testutil.EngineIonCUUID + `", "shield_uuid": null, "weapon_uuid": null}`
	httpReq, err := http.NewRequest(http.MethodPost, orderBaseURL()+"/api/v1/orders", bytes.NewReader([]byte(body)))
	require.NoError(t, err)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusCreated, resp.StatusCode, "null в shield_uuid и weapon_uuid означает «детали нет»")

	var result CreateOrderResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, int64(testutil.HullAluminumPrice+testutil.EngineIonCPrice), result.TotalPrice)
}

func TestOrder_Create_WithShieldOnly(t *testing.T) {
	req := &CreateOrderRequest{
		HullUUID:   testutil.HullAluminumUUID,
		EngineUUID: testutil.EngineIonCUUID,
		ShieldUUID: new(testutil.ShieldEnergyUUID),
	}

	result, resp := createOrder(t, req)
	defer resp.Body.Close()

	require.Equal(t, http.StatusCreated, resp.StatusCode)
	require.NotNil(t, result)
	expectedTotal := int64(testutil.HullAluminumPrice + testutil.EngineIonCPrice + testutil.ShieldEnergyPrice)
	assert.Equal(t, expectedTotal, result.TotalPrice)
}
