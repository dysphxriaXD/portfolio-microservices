package inventory

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/dysphxriaXD/e2e/testutil"
	inventoryv1 "github.com/dysphxriaXD/shared/pkg/proto/inventory/v1"
)

// Тесты InventoryService (gRPC)

func TestInventory_GetPart_Success(t *testing.T) {
	resp, err := inventoryClient.GetPart(context.Background(), &inventoryv1.GetPartRequest{
		Uuid: testutil.HullAluminumUUID,
	})
	require.NoError(t, err)

	part := resp.GetPart()
	assert.Equal(t, testutil.HullAluminumUUID, part.GetUuid())
	assert.Equal(t, int64(testutil.HullAluminumPrice), part.GetPrice())
	assert.Equal(t, inventoryv1.PartType_PART_TYPE_HULL, part.GetPartType())
	assert.NotEmpty(t, part.GetName())
	assert.NotNil(t, part.GetCreatedAt())
}

func TestInventory_GetPart_AllTypes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		uuid     string
		price    int64
		partType inventoryv1.PartType
	}{
		{"Hull Aluminum", testutil.HullAluminumUUID, testutil.HullAluminumPrice, inventoryv1.PartType_PART_TYPE_HULL},
		{"Hull Titanium", testutil.HullTitaniumUUID, testutil.HullTitaniumPrice, inventoryv1.PartType_PART_TYPE_HULL},
		{"Engine Ion C", testutil.EngineIonCUUID, testutil.EngineIonCPrice, inventoryv1.PartType_PART_TYPE_ENGINE},
		{"Engine Ion B", testutil.EngineIonBUUID, testutil.EngineIonBPrice, inventoryv1.PartType_PART_TYPE_ENGINE},
		{"Shield Energy", testutil.ShieldEnergyUUID, testutil.ShieldEnergyPrice, inventoryv1.PartType_PART_TYPE_SHIELD},
		{"Weapon Laser", testutil.WeaponLaserUUID, testutil.WeaponLaserPrice, inventoryv1.PartType_PART_TYPE_WEAPON},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			resp, err := inventoryClient.GetPart(context.Background(), &inventoryv1.GetPartRequest{
				Uuid: tc.uuid,
			})
			require.NoError(t, err)

			part := resp.GetPart()
			assert.Equal(t, tc.uuid, part.GetUuid())
			assert.Equal(t, tc.price, part.GetPrice())
			assert.Equal(t, tc.partType, part.GetPartType())
		})
	}
}

func TestInventory_GetPart_NotFound(t *testing.T) {
	_, err := inventoryClient.GetPart(context.Background(), &inventoryv1.GetPartRequest{
		Uuid: uuid.New().String(),
	})
	require.Error(t, err)
	testutil.AssertGRPCStatus(t, err, codes.NotFound)
}

func TestInventory_GetPart_EmptyUUID(t *testing.T) {
	_, err := inventoryClient.GetPart(context.Background(), &inventoryv1.GetPartRequest{
		Uuid: "",
	})
	require.Error(t, err)
	testutil.AssertGRPCStatus(t, err, codes.InvalidArgument)
}

func TestInventory_GetPart_InvalidUUID(t *testing.T) {
	_, err := inventoryClient.GetPart(context.Background(), &inventoryv1.GetPartRequest{
		Uuid: "invalid-uuid-format",
	})
	require.Error(t, err)
	testutil.AssertGRPCStatus(t, err, codes.InvalidArgument)
}

func TestInventory_ListParts_All(t *testing.T) {
	resp, err := inventoryClient.ListParts(context.Background(), &inventoryv1.ListPartsRequest{
		PartType: inventoryv1.PartType_PART_TYPE_UNSPECIFIED,
	})
	require.NoError(t, err)
	assert.Len(t, resp.GetParts(), 7)
}

func TestInventory_ListParts_ByType_Hull(t *testing.T) {
	resp, err := inventoryClient.ListParts(context.Background(), &inventoryv1.ListPartsRequest{
		PartType: inventoryv1.PartType_PART_TYPE_HULL,
	})
	require.NoError(t, err)
	assert.Len(t, resp.GetParts(), 3)

	for _, part := range resp.GetParts() {
		assert.Equal(t, inventoryv1.PartType_PART_TYPE_HULL, part.GetPartType())
	}
}

func TestInventory_ListParts_ByType_Engine(t *testing.T) {
	resp, err := inventoryClient.ListParts(context.Background(), &inventoryv1.ListPartsRequest{
		PartType: inventoryv1.PartType_PART_TYPE_ENGINE,
	})
	require.NoError(t, err)
	assert.Len(t, resp.GetParts(), 2)

	for _, part := range resp.GetParts() {
		assert.Equal(t, inventoryv1.PartType_PART_TYPE_ENGINE, part.GetPartType())
	}
}

func TestInventory_ListParts_ByType_Shield(t *testing.T) {
	resp, err := inventoryClient.ListParts(context.Background(), &inventoryv1.ListPartsRequest{
		PartType: inventoryv1.PartType_PART_TYPE_SHIELD,
	})
	require.NoError(t, err)
	assert.Len(t, resp.GetParts(), 1)
	assert.Equal(t, testutil.ShieldEnergyUUID, resp.GetParts()[0].GetUuid())
}

func TestInventory_ListParts_ByType_Weapon(t *testing.T) {
	resp, err := inventoryClient.ListParts(context.Background(), &inventoryv1.ListPartsRequest{
		PartType: inventoryv1.PartType_PART_TYPE_WEAPON,
	})
	require.NoError(t, err)
	assert.Len(t, resp.GetParts(), 1)
	assert.Equal(t, testutil.WeaponLaserUUID, resp.GetParts()[0].GetUuid())
}

func TestInventory_ListParts_SortedByName(t *testing.T) {
	resp, err := inventoryClient.ListParts(context.Background(), &inventoryv1.ListPartsRequest{
		PartType: inventoryv1.PartType_PART_TYPE_UNSPECIFIED,
	})
	require.NoError(t, err)

	parts := resp.GetParts()
	for i := 1; i < len(parts); i++ {
		assert.LessOrEqual(t, parts[i-1].GetName(), parts[i].GetName(),
			"детали должны быть отсортированы по имени в алфавитном порядке")
	}
}

func TestInventory_ListParts_ByType_SortedByName(t *testing.T) {
	// Корпусов три, поэтому на них видно порядок и при фильтре по типу
	resp, err := inventoryClient.ListParts(context.Background(), &inventoryv1.ListPartsRequest{
		PartType: inventoryv1.PartType_PART_TYPE_HULL,
	})
	require.NoError(t, err)

	parts := resp.GetParts()
	for i := 1; i < len(parts); i++ {
		assert.LessOrEqual(t, parts[i-1].GetName(), parts[i].GetName(),
			"детали одного типа должны быть отсортированы по имени в алфавитном порядке")
	}
}

// Тесты ListParts.uuids

func TestInventory_ListParts_ByUuids_Success(t *testing.T) {
	uuids := []string{testutil.HullAluminumUUID, testutil.EngineIonCUUID, testutil.ShieldEnergyUUID}

	resp, err := inventoryClient.ListParts(context.Background(), &inventoryv1.ListPartsRequest{
		Uuids: uuids,
	})
	require.NoError(t, err)
	assert.Len(t, resp.GetParts(), 3)

	// Проверяем, что вернулись нужные детали
	returnedUUIDs := make([]string, len(resp.GetParts()))
	for i, part := range resp.GetParts() {
		returnedUUIDs[i] = part.GetUuid()
	}
	assert.ElementsMatch(t, uuids, returnedUUIDs)
}

func TestInventory_ListParts_ByUuids_PreservesOrder(t *testing.T) {
	// Запрос в определённом порядке (engine, hull, weapon)
	uuids := []string{testutil.EngineIonCUUID, testutil.HullAluminumUUID, testutil.WeaponLaserUUID}

	resp, err := inventoryClient.ListParts(context.Background(), &inventoryv1.ListPartsRequest{
		Uuids: uuids,
	})
	require.NoError(t, err)
	assert.Len(t, resp.GetParts(), 3)

	// Проверяем, что порядок сохранён как в запросе
	for i, part := range resp.GetParts() {
		assert.Equal(t, uuids[i], part.GetUuid(),
			"деталь с индексом %d должна соответствовать порядку запрошенных UUID", i)
	}
}

func TestInventory_ListParts_ByUuids_IgnoresPartType(t *testing.T) {
	// Запрос с uuids И part_type (part_type должен быть проигнорирован)
	uuids := []string{testutil.HullAluminumUUID, testutil.EngineIonCUUID}

	resp, err := inventoryClient.ListParts(context.Background(), &inventoryv1.ListPartsRequest{
		Uuids:    uuids,
		PartType: inventoryv1.PartType_PART_TYPE_WEAPON, // Должен быть проигнорирован
	})
	require.NoError(t, err)
	assert.Len(t, resp.GetParts(), 2)

	// Проверяем, что получили Hull и Engine, а не Weapons
	assert.Equal(t, testutil.HullAluminumUUID, resp.GetParts()[0].GetUuid())
	assert.Equal(t, testutil.EngineIonCUUID, resp.GetParts()[1].GetUuid())
}

func TestInventory_ListParts_ByUuids_NotFound(t *testing.T) {
	// Включаем один несуществующий UUID
	nonExistentUUID := uuid.New().String()
	uuids := []string{testutil.HullAluminumUUID, nonExistentUUID, testutil.EngineIonCUUID}

	_, err := inventoryClient.ListParts(context.Background(), &inventoryv1.ListPartsRequest{
		Uuids: uuids,
	})
	require.Error(t, err)
	testutil.AssertGRPCStatus(t, err, codes.NotFound)
}

func TestInventory_ListParts_ByUuids_InvalidUUID(t *testing.T) {
	uuids := []string{testutil.HullAluminumUUID, "invalid-uuid-format"}

	_, err := inventoryClient.ListParts(context.Background(), &inventoryv1.ListPartsRequest{
		Uuids: uuids,
	})
	require.Error(t, err)
	testutil.AssertGRPCStatus(t, err, codes.InvalidArgument)
}

func TestInventory_ListParts_ByUuids_SingleUUID(t *testing.T) {
	uuids := []string{testutil.WeaponLaserUUID}

	resp, err := inventoryClient.ListParts(context.Background(), &inventoryv1.ListPartsRequest{
		Uuids: uuids,
	})
	require.NoError(t, err)
	assert.Len(t, resp.GetParts(), 1)
	assert.Equal(t, testutil.WeaponLaserUUID, resp.GetParts()[0].GetUuid())
	assert.Equal(t, int64(testutil.WeaponLaserPrice), resp.GetParts()[0].GetPrice())
}

func TestInventory_ListParts_ByUuids_AllParts(t *testing.T) {
	// Запрашиваем все 7 деталей по UUID
	uuids := []string{
		testutil.HullAluminumUUID, testutil.HullTitaniumUUID,
		testutil.EngineIonCUUID, testutil.EngineIonBUUID,
		testutil.ShieldEnergyUUID, testutil.WeaponLaserUUID,
		testutil.HullOutOfStockUUID,
	}

	resp, err := inventoryClient.ListParts(context.Background(), &inventoryv1.ListPartsRequest{
		Uuids: uuids,
	})
	require.NoError(t, err)
	assert.Len(t, resp.GetParts(), 7)

	// Проверяем, что порядок совпадает с порядком запроса
	for i, part := range resp.GetParts() {
		assert.Equal(t, uuids[i], part.GetUuid())
	}
}

// Тест inventory: деталь с нулевым остатком

func TestInventory_GetPart_OutOfStock(t *testing.T) {
	resp, err := inventoryClient.GetPart(context.Background(), &inventoryv1.GetPartRequest{
		Uuid: testutil.HullOutOfStockUUID,
	})
	require.NoError(t, err)

	part := resp.GetPart()
	assert.Equal(t, testutil.HullOutOfStockUUID, part.GetUuid())
	assert.Equal(t, int64(testutil.HullOutOfStockPrice), part.GetPrice())
	assert.Equal(t, int64(0), part.GetStockQuantity())
	assert.Equal(t, inventoryv1.PartType_PART_TYPE_HULL, part.GetPartType())
}

func TestInventory_ListParts_ByUuids_EmptyList(t *testing.T) {
	// Пустой список UUID (должен вернуть все детали при фильтрации по типу UNSPECIFIED)
	resp, err := inventoryClient.ListParts(context.Background(), &inventoryv1.ListPartsRequest{
		Uuids: []string{},
	})
	require.NoError(t, err)
	assert.Len(t, resp.GetParts(), 7)
}
