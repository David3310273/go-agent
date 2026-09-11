// auto-added: test cases for core.Storage interface and related functions
package test

import (
	"testing"

	"github.com/David3310273/go-agent/core"
	testmock "github.com/David3310273/go-agent/test/mock"
	"go.uber.org/mock/gomock"
)

// compile-time check: MockStorage satisfies core.Storage
var _ core.Storage = (*testmock.MockStorage)(nil)

// =============================================================================
// Storage interface tests
// =============================================================================

func TestMockStorage_SetConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := testmock.NewMockStorage(ctrl)
	configPath := "/path/to/config.json"

	mockStorage.EXPECT().SetConfig(configPath)

	mockStorage.SetConfig(configPath)
}

func TestMockStorage_Stop(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := testmock.NewMockStorage(ctrl)

	mockStorage.EXPECT().Stop()

	mockStorage.Stop()
}

// =============================================================================
// InitStorageClient function tests
// =============================================================================

func TestInitStorageClient(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := testmock.NewMockStorage(ctrl)
	configPath := "config.json"

	mockStorage.EXPECT().SetConfig(configPath)

	core.InitStorageClient(mockStorage, configPath)
}

func TestInitStorageClient_WithAbsolutePath(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := testmock.NewMockStorage(ctrl)
	configPath := "/absolute/path/to/config.json"

	mockStorage.EXPECT().SetConfig(configPath)

	core.InitStorageClient(mockStorage, configPath)
}

// =============================================================================
// StopStorageClient function tests
// =============================================================================

func TestStopStorageClient(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := testmock.NewMockStorage(ctrl)

	mockStorage.EXPECT().Stop()

	core.StopStorageClient(mockStorage)
}

// =============================================================================
// Storage lifecycle tests
// =============================================================================

func TestStorageLifecycle_InitAndStop(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := testmock.NewMockStorage(ctrl)
	configPath := "config.json"

	gomock.InOrder(
		mockStorage.EXPECT().SetConfig(configPath),
		mockStorage.EXPECT().Stop(),
	)

	core.InitStorageClient(mockStorage, configPath)
	core.StopStorageClient(mockStorage)
}

func TestStorageLifecycle_MultipleInitCalls(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := testmock.NewMockStorage(ctrl)

	mockStorage.EXPECT().SetConfig("config1.json")
	mockStorage.EXPECT().SetConfig("config2.json")

	core.InitStorageClient(mockStorage, "config1.json")
	core.InitStorageClient(mockStorage, "config2.json")
}
