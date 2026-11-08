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

// Тесты OrderService (HTTP): оплата заказа

func TestOrder_Pay_Success_Card(t *testing.T) {
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
	payResult, resp := payOrder(t, createResult.OrderUUID, payReq)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.NotNil(t, payResult)
	assert.NotEmpty(t, payResult.TransactionUUID)
}

func TestOrder_Pay_VerifyStatusChange(t *testing.T) {
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

	// Получаем и проверяем смену статуса на PAID
	order, getResp := getOrder(t, createResult.OrderUUID)
	defer getResp.Body.Close()

	require.Equal(t, http.StatusOK, getResp.StatusCode)
	assert.Equal(t, "PAID", order.Status)
	assert.NotNil(t, order.TransactionUUID)
	require.NotNil(t, order.PaymentMethod, "после оплаты payment_method должен быть заполнен")
	assert.Equal(t, "CARD", *order.PaymentMethod)
}

func TestOrder_Pay_NotFound(t *testing.T) {
	payReq := &PayOrderRequest{PaymentMethod: "CARD"}
	_, resp := payOrder(t, uuid.New().String(), payReq)
	defer resp.Body.Close()

	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestOrder_Pay_AlreadyPaid(t *testing.T) {
	// Создаём заказ
	createReq := &CreateOrderRequest{
		HullUUID:   testutil.HullAluminumUUID,
		EngineUUID: testutil.EngineIonCUUID,
	}
	createResult, createResp := createOrder(t, createReq)
	defer createResp.Body.Close()
	require.NotNil(t, createResult)

	// Оплачиваем заказ в первый раз
	payReq := &PayOrderRequest{PaymentMethod: "CARD"}
	_, payResp1 := payOrder(t, createResult.OrderUUID, payReq)
	defer payResp1.Body.Close()

	// Пытаемся оплатить повторно (должна быть ошибка конфликта)
	_, payResp2 := payOrder(t, createResult.OrderUUID, payReq)
	defer payResp2.Body.Close()

	require.Equal(t, http.StatusConflict, payResp2.StatusCode)
}

func TestOrder_Pay_AlreadyCancelled(t *testing.T) {
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

	// Пытаемся оплатить отменённый заказ (должна быть ошибка конфликта)
	payReq := &PayOrderRequest{PaymentMethod: "CARD"}
	_, payResp := payOrder(t, createResult.OrderUUID, payReq)
	defer payResp.Body.Close()

	require.Equal(t, http.StatusConflict, payResp.StatusCode)
}

func TestOrder_Pay_AllMethods(t *testing.T) {
	t.Parallel()

	methods := []string{"CARD", "SBP", "CREDIT_CARD", "INVESTOR_MONEY"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			t.Parallel()

			// Создаём заказ
			createReq := &CreateOrderRequest{
				HullUUID:   testutil.HullAluminumUUID,
				EngineUUID: testutil.EngineIonCUUID,
			}
			createResult, createResp := createOrder(t, createReq)
			defer createResp.Body.Close()
			require.NotNil(t, createResult)

			// Оплачиваем этим методом
			payReq := &PayOrderRequest{PaymentMethod: method}
			payResult, resp := payOrder(t, createResult.OrderUUID, payReq)
			defer resp.Body.Close()

			require.Equal(t, http.StatusOK, resp.StatusCode)
			require.NotNil(t, payResult)
			assert.NotEmpty(t, payResult.TransactionUUID)

			// Проверяем, что метод оплаты сохранён
			order, getResp := getOrder(t, createResult.OrderUUID)
			defer getResp.Body.Close()
			require.NotNil(t, order, "заказ не получен: GET /api/v1/orders/{order_uuid} вернул %d вместо 200", getResp.StatusCode)
			require.NotNil(t, order.PaymentMethod)
			assert.Equal(t, method, *order.PaymentMethod)
		})
	}
}

func TestOrder_Pay_InvalidUUIDInPath(t *testing.T) {
	body := `{"payment_method": "CARD"}`
	httpReq, err := http.NewRequest(http.MethodPost, orderBaseURL()+"/api/v1/orders/not-a-uuid/pay", bytes.NewReader([]byte(body)))
	require.NoError(t, err)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestOrder_Pay_InvalidPaymentMethod(t *testing.T) {
	// Создаём заказ
	createReq := &CreateOrderRequest{
		HullUUID:   testutil.HullAluminumUUID,
		EngineUUID: testutil.EngineIonCUUID,
	}
	createResult, createResp := createOrder(t, createReq)
	defer createResp.Body.Close()
	require.NotNil(t, createResult)

	// Пытаемся оплатить невалидным методом (ogen отклонит)
	body := `{"payment_method": "BITCOIN"}`
	httpReq, err := http.NewRequest(http.MethodPost,
		orderBaseURL()+"/api/v1/orders/"+createResult.OrderUUID+"/pay",
		bytes.NewReader([]byte(body)))
	require.NoError(t, err)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestOrder_Pay_MissingPaymentMethod(t *testing.T) {
	// Создаём заказ
	createReq := &CreateOrderRequest{
		HullUUID:   testutil.HullAluminumUUID,
		EngineUUID: testutil.EngineIonCUUID,
	}
	createResult, createResp := createOrder(t, createReq)
	defer createResp.Body.Close()
	require.NotNil(t, createResult)

	// Пытаемся оплатить без payment_method
	body := `{}`
	httpReq, err := http.NewRequest(http.MethodPost,
		orderBaseURL()+"/api/v1/orders/"+createResult.OrderUUID+"/pay",
		bytes.NewReader([]byte(body)))
	require.NoError(t, err)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestOrder_Pay_EmptyBody(t *testing.T) {
	// Создаём заказ
	createReq := &CreateOrderRequest{
		HullUUID:   testutil.HullAluminumUUID,
		EngineUUID: testutil.EngineIonCUUID,
	}
	createResult, createResp := createOrder(t, createReq)
	defer createResp.Body.Close()
	require.NotNil(t, createResult)

	httpReq, err := http.NewRequest(http.MethodPost,
		orderBaseURL()+"/api/v1/orders/"+createResult.OrderUUID+"/pay",
		bytes.NewReader([]byte("")))
	require.NoError(t, err)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
