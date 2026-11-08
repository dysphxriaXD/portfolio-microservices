package order

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dysphxriaXD/e2e/testutil"
)

// Тесты OrderService (HTTP): отмена заказа

func TestOrder_Cancel_Success(t *testing.T) {
	// Создаём заказ
	createReq := &CreateOrderRequest{
		HullUUID:   testutil.HullAluminumUUID,
		EngineUUID: testutil.EngineIonCUUID,
	}
	createResult, createResp := createOrder(t, createReq)
	defer createResp.Body.Close()
	require.NotNil(t, createResult)

	// Отменяем заказ
	_, resp := cancelOrder(t, createResult.OrderUUID)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestOrder_Cancel_Success_EmptyJSONBody(t *testing.T) {
	// Создаём заказ
	createReq := &CreateOrderRequest{
		HullUUID:   testutil.HullAluminumUUID,
		EngineUUID: testutil.EngineIonCUUID,
	}
	createResult, createResp := createOrder(t, createReq)
	defer createResp.Body.Close()
	require.NotNil(t, createResult)

	// Отменяем заказ с телом {}: по контракту тело пустое или {}
	httpReq, err := http.NewRequest(http.MethodPost,
		orderBaseURL()+"/api/v1/orders/"+createResult.OrderUUID+"/cancel",
		bytes.NewReader([]byte("{}")))
	require.NoError(t, err)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestOrder_Cancel_VerifyStatusChange(t *testing.T) {
	// Создаём заказ
	createReq := &CreateOrderRequest{
		HullUUID:   testutil.HullAluminumUUID,
		EngineUUID: testutil.EngineIonCUUID,
	}
	createResult, createResp := createOrder(t, createReq)
	defer createResp.Body.Close()
	require.NotNil(t, createResult)

	// Отменяем заказ
	_, cancelResp := cancelOrder(t, createResult.OrderUUID)
	defer cancelResp.Body.Close()

	// Получаем и проверяем смену статуса на CANCELLED
	order, getResp := getOrder(t, createResult.OrderUUID)
	defer getResp.Body.Close()

	require.Equal(t, http.StatusOK, getResp.StatusCode)
	assert.Equal(t, "CANCELLED", order.Status)
}

func TestOrder_Cancel_NotFound(t *testing.T) {
	_, resp := cancelOrder(t, uuid.New().String())
	defer resp.Body.Close()

	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestOrder_Cancel_AlreadyPaid(t *testing.T) {
	// Создаём заказ
	createReq := &CreateOrderRequest{
		HullUUID:   testutil.HullAluminumUUID,
		EngineUUID: testutil.EngineIonCUUID,
	}
	createResult, createResp := createOrder(t, createReq)
	defer createResp.Body.Close()
	require.NotNil(t, createResult)

	// Оплачиваем заказ
	payReq := &PayOrderRequest{PaymentMethod: "CARD"}
	_, payResp := payOrder(t, createResult.OrderUUID, payReq)
	defer payResp.Body.Close()

	// Пытаемся отменить оплаченный заказ (должна быть ошибка конфликта)
	_, cancelResp := cancelOrder(t, createResult.OrderUUID)
	defer cancelResp.Body.Close()

	require.Equal(t, http.StatusConflict, cancelResp.StatusCode)
}

func TestOrder_Cancel_AlreadyCancelled(t *testing.T) {
	// Создаём заказ
	createReq := &CreateOrderRequest{
		HullUUID:   testutil.HullAluminumUUID,
		EngineUUID: testutil.EngineIonCUUID,
	}
	createResult, createResp := createOrder(t, createReq)
	defer createResp.Body.Close()
	require.NotNil(t, createResult)

	// Отменяем заказ первый раз
	_, cancelResp1 := cancelOrder(t, createResult.OrderUUID)
	defer cancelResp1.Body.Close()

	// Пытаемся отменить повторно (должна быть ошибка конфликта)
	_, cancelResp2 := cancelOrder(t, createResult.OrderUUID)
	defer cancelResp2.Body.Close()

	require.Equal(t, http.StatusConflict, cancelResp2.StatusCode)
}

func TestOrder_Cancel_InvalidUUIDInPath(t *testing.T) {
	httpReq, err := http.NewRequest(http.MethodPost, orderBaseURL()+"/api/v1/orders/not-a-uuid/cancel", nil)
	require.NoError(t, err)

	resp, err := httpClient.Do(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
