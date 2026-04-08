package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/device"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

const updateDeviceID = "660e8400-e29b-41d4-a716-446655440000"

type updateDeviceRepoStub struct {
	devices     map[string]*device.Device
	findErr     error
	findErrCall int
	updateErr   error
	findCalls   int
	updateCalls int
	updatedIDs  []string
}

func newUpdateDeviceRepoStub() *updateDeviceRepoStub {
	return &updateDeviceRepoStub{devices: map[string]*device.Device{}}
}

func (s *updateDeviceRepoStub) FindByID(_ context.Context, deviceID shared.DeviceID) (*device.Device, error) {
	s.findCalls++
	if s.findErr != nil && (s.findErrCall == 0 || s.findCalls == s.findErrCall) {
		return nil, s.findErr
	}
	d, ok := s.devices[deviceID.String()]
	if !ok {
		return nil, device.ErrDeviceNotFound
	}
	return cloneUpdateDevice(d), nil
}

func (s *updateDeviceRepoStub) FindAllByAccountID(context.Context, shared.AccountID) ([]*device.Device, error) {
	return nil, nil
}

func (s *updateDeviceRepoStub) Create(context.Context, *device.Device) error {
	return nil
}

func (s *updateDeviceRepoStub) Update(_ context.Context, d *device.Device) error {
	s.updateCalls++
	s.updatedIDs = append(s.updatedIDs, d.ID.String())
	if s.updateErr != nil {
		return s.updateErr
	}
	if _, exists := s.devices[d.ID.String()]; !exists {
		return device.ErrDeviceNotFound
	}
	updated := cloneUpdateDevice(d)
	updated.UpdatedAt = time.Date(2026, 4, 8, 9, 10, 11, 0, time.UTC)
	s.devices[d.ID.String()] = updated
	return nil
}

func (s *updateDeviceRepoStub) BindAccount(context.Context, shared.DeviceID, shared.AccountID) error {
	return nil
}

func (s *updateDeviceRepoStub) DeleteByAccount(context.Context, shared.DeviceID, shared.AccountID) error {
	return nil
}

func cloneUpdateDevice(d *device.Device) *device.Device {
	if d == nil {
		return nil
	}
	return &device.Device{
		ID:        d.ID,
		Platform:  d.Platform,
		Name:      d.Name,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
}

func updateUseCaseInput(deviceID, platform, name string) appShared.UseCaseInput[UpdateDeviceInput] {
	return appShared.UseCaseInput[UpdateDeviceInput]{
		Data: UpdateDeviceInput{
			DeviceID: deviceID,
			Platform: platform,
			Name:     name,
		},
	}
}

func seedUpdateDevice(t *testing.T, repo *updateDeviceRepoStub) {
	t.Helper()
	deviceID, err := shared.ParseDeviceID(updateDeviceID)
	if err != nil {
		t.Fatal(err)
	}
	createdAt := time.Date(2026, 4, 7, 1, 2, 3, 0, time.UTC)
	repo.devices[updateDeviceID] = &device.Device{
		ID:        deviceID,
		Platform:  device.Linux,
		Name:      "Old Device",
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}
}

func TestUpdateDeviceUseCase_UpdateFailureHasStructuredLog(t *testing.T) {
	start := time.Now()
	repo := newUpdateDeviceRepoStub()
	seedUpdateDevice(t, repo)
	repo.updateErr = device.ErrDeviceNotFound
	uc := NewUpdateDeviceUseCase(repo)
	input := updateUseCaseInput(updateDeviceID, "macos", "Renamed Device")

	t.Log("Given: repository update returns an error after the device is loaded")
	t.Logf("Input: path_device_id=%s payload_platform=%s payload_device_name=%q",
		input.Data.DeviceID, input.Data.Platform, input.Data.Name)
	t.Log("Action: execute update device usecase update failure path")

	out, err := uc.Execute(context.Background(), input)

	t.Logf("Output: out=%+v err=%v", out, err)
	t.Logf("Mutation: find_calls=%d update_calls=%d updated_ids=%v persisted_name=%q",
		repo.findCalls, repo.updateCalls, repo.updatedIDs, repo.devices[updateDeviceID].Name)
	t.Logf("Duration: %s", time.Since(start))

	if !errors.Is(err, ErrUpdateFailed) {
		t.Fatalf("expected ErrUpdateFailed, got %v", err)
	}
	if repo.findCalls != 1 || repo.updateCalls != 1 {
		t.Fatalf("expected find=1 update=1, got find=%d update=%d", repo.findCalls, repo.updateCalls)
	}
}

func TestUpdateDeviceUseCase_PostUpdateReadFailureHasStructuredLog(t *testing.T) {
	start := time.Now()
	repo := newUpdateDeviceRepoStub()
	seedUpdateDevice(t, repo)
	uc := NewUpdateDeviceUseCase(repo)
	input := updateUseCaseInput(updateDeviceID, "macos", "Renamed Device")

	t.Log("Given: repository can update the device, but the post-update read fails")
	t.Logf("Input: path_device_id=%s payload_platform=%s payload_device_name=%q",
		input.Data.DeviceID, input.Data.Platform, input.Data.Name)
	t.Log("Action: execute update device usecase with second read failure")

	repo.findErr = errors.New("post-update read failed")
	repo.findErrCall = 2
	out, err := uc.Execute(context.Background(), input)

	t.Logf("Output: out=%+v err=%v", out, err)
	t.Logf("Mutation: find_calls=%d update_calls=%d updated_ids=%v persisted_name=%q",
		repo.findCalls, repo.updateCalls, repo.updatedIDs, repo.devices[updateDeviceID].Name)
	t.Logf("Duration: %s", time.Since(start))

	if !errors.Is(err, ErrUpdateFailed) {
		t.Fatalf("expected ErrUpdateFailed, got %v", err)
	}
	if repo.findCalls != 2 || repo.updateCalls != 1 {
		t.Fatalf("expected find=2 update=1, got find=%d update=%d", repo.findCalls, repo.updateCalls)
	}
}
