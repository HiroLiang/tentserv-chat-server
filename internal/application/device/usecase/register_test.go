package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/device"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/transaction"
)

const registerDeviceID = "550e8400-e29b-41d4-a716-446655440000"

type registerTxStub struct {
	commitCalls   int
	rollbackCalls int
	commitErr     error
}

func (t *registerTxStub) Commit() error {
	t.commitCalls++
	return t.commitErr
}

func (t *registerTxStub) Rollback() error {
	t.rollbackCalls++
	return nil
}

type registerUOWStub struct {
	tx       *registerTxStub
	beginErr error
}

func (u *registerUOWStub) Begin(ctx context.Context) (context.Context, transaction.Transaction, error) {
	if u.beginErr != nil {
		return ctx, nil, u.beginErr
	}
	if u.tx == nil {
		u.tx = &registerTxStub{}
	}
	return ctx, u.tx, nil
}

type registerDeviceRepoStub struct {
	devices     map[string]*device.Device
	createErr   error
	findErr     error
	updateErr   error
	createCalls int
	findCalls   int
	updateCalls int
}

func newRegisterDeviceRepoStub() *registerDeviceRepoStub {
	return &registerDeviceRepoStub{devices: map[string]*device.Device{}}
}

func (s *registerDeviceRepoStub) FindByID(_ context.Context, deviceID shared.DeviceID) (*device.Device, error) {
	s.findCalls++
	if s.findErr != nil {
		return nil, s.findErr
	}
	d, ok := s.devices[deviceID.String()]
	if !ok {
		return nil, device.ErrDeviceNotFound
	}
	return cloneRegisterDevice(d), nil
}

func (s *registerDeviceRepoStub) FindAllByAccountID(context.Context, shared.AccountID) ([]*device.Device, error) {
	return nil, nil
}

func (s *registerDeviceRepoStub) Create(_ context.Context, d *device.Device) error {
	s.createCalls++
	if s.createErr != nil {
		return s.createErr
	}
	if _, exists := s.devices[d.ID.String()]; exists {
		return device.ErrDeviceAlreadyExists
	}
	created := cloneRegisterDevice(d)
	created.CreatedAt = time.Date(2026, 4, 8, 1, 2, 3, 0, time.UTC)
	created.UpdatedAt = created.CreatedAt
	s.devices[d.ID.String()] = created
	return nil
}

func (s *registerDeviceRepoStub) Update(_ context.Context, d *device.Device) error {
	s.updateCalls++
	if s.updateErr != nil {
		return s.updateErr
	}
	if _, exists := s.devices[d.ID.String()]; !exists {
		return device.ErrDeviceNotFound
	}
	updated := cloneRegisterDevice(d)
	updated.UpdatedAt = time.Date(2026, 4, 8, 2, 3, 4, 0, time.UTC)
	s.devices[d.ID.String()] = updated
	return nil
}

func (s *registerDeviceRepoStub) BindAccount(context.Context, shared.DeviceID, shared.AccountID) error {
	return nil
}

func (s *registerDeviceRepoStub) DeleteByAccount(context.Context, shared.DeviceID, shared.AccountID) error {
	return nil
}

func cloneRegisterDevice(d *device.Device) *device.Device {
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

func registerUseCaseInput(deviceID, platform, name string) appShared.UseCaseInput[RegisterInput] {
	return appShared.UseCaseInput[RegisterInput]{
		Data: RegisterInput{
			DeviceID: deviceID,
			Platform: platform,
			Name:     name,
		},
	}
}

func seedRegisterDevice(t *testing.T, repo *registerDeviceRepoStub) {
	t.Helper()
	deviceID, err := shared.ParseDeviceID(registerDeviceID)
	if err != nil {
		t.Fatal(err)
	}
	createdAt := time.Date(2026, 4, 7, 1, 2, 3, 0, time.UTC)
	repo.devices[registerDeviceID] = &device.Device{
		ID:        deviceID,
		Platform:  device.Linux,
		Name:      "Old Workstation",
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}
}

func TestRegisterUseCase_BeginFailureHasStructuredLog(t *testing.T) {
	start := time.Now()
	repo := newRegisterDeviceRepoStub()
	uow := &registerUOWStub{beginErr: errors.New("begin failed")}
	uc := NewRegisterUseCase(uow, repo)
	input := registerUseCaseInput(registerDeviceID, "macos", "Hiro MacBook")

	t.Log("Given: unit of work cannot start a transaction")
	t.Logf("Input: device_id=%s platform=%s name=%q", input.Data.DeviceID, input.Data.Platform, input.Data.Name)
	t.Log("Action: execute register usecase transaction begin failure path")

	out, err := uc.Execute(context.Background(), input)

	t.Logf("Output: out=%+v err=%v", out, err)
	t.Logf("Mutation: create_calls=%d update_calls=%d find_calls=%d rows=%d", repo.createCalls, repo.updateCalls, repo.findCalls, len(repo.devices))
	t.Logf("Duration: %s", time.Since(start))

	if !errors.Is(err, ErrRegisterFailed) {
		t.Fatalf("expected ErrRegisterFailed, got %v", err)
	}
	if repo.createCalls != 0 || repo.updateCalls != 0 || repo.findCalls != 0 {
		t.Fatalf("expected no repository calls, got create=%d update=%d find=%d", repo.createCalls, repo.updateCalls, repo.findCalls)
	}
}

func TestRegisterUseCase_CreateFailureHasStructuredLog(t *testing.T) {
	start := time.Now()
	repo := newRegisterDeviceRepoStub()
	repo.createErr = errors.New("database unavailable")
	uow := &registerUOWStub{}
	uc := NewRegisterUseCase(uow, repo)
	input := registerUseCaseInput(registerDeviceID, "macos", "Hiro MacBook")

	t.Log("Given: repository create operation returns a database error")
	t.Logf("Input: device_id=%s platform=%s name=%q", input.Data.DeviceID, input.Data.Platform, input.Data.Name)
	t.Log("Action: execute register usecase create failure path")

	out, err := uc.Execute(context.Background(), input)

	t.Logf("Output: out=%+v err=%v", out, err)
	t.Logf("Mutation: create_calls=%d update_calls=%d find_calls=%d commit_calls=%d rollback_calls=%d rows=%d",
		repo.createCalls, repo.updateCalls, repo.findCalls, uow.tx.commitCalls, uow.tx.rollbackCalls, len(repo.devices))
	t.Logf("Duration: %s", time.Since(start))

	if !errors.Is(err, ErrRegisterFailed) {
		t.Fatalf("expected ErrRegisterFailed, got %v", err)
	}
	if repo.createCalls != 1 || repo.updateCalls != 0 || repo.findCalls != 0 || uow.tx.commitCalls != 0 {
		t.Fatalf("expected create=1 update=0 find=0 commit=0, got create=%d update=%d find=%d commit=%d",
			repo.createCalls, repo.updateCalls, repo.findCalls, uow.tx.commitCalls)
	}
}

func TestRegisterUseCase_UpdateExistingDeviceFailureHasStructuredLog(t *testing.T) {
	start := time.Now()
	repo := newRegisterDeviceRepoStub()
	seedRegisterDevice(t, repo)
	repo.updateErr = errors.New("update failed")
	uow := &registerUOWStub{}
	uc := NewRegisterUseCase(uow, repo)
	input := registerUseCaseInput(registerDeviceID, "macos", "Renamed MacBook")

	t.Log("Given: repository update fails after existing device is loaded")
	t.Logf("Input: device_id=%s platform=%s name=%q", input.Data.DeviceID, input.Data.Platform, input.Data.Name)
	t.Log("Action: execute register usecase existing-device update failure path")

	out, err := uc.Execute(context.Background(), input)

	t.Logf("Output: out=%+v err=%v", out, err)
	t.Logf("Mutation: create_calls=%d update_calls=%d find_calls=%d commit_calls=%d rollback_calls=%d persisted_name=%q",
		repo.createCalls, repo.updateCalls, repo.findCalls, uow.tx.commitCalls, uow.tx.rollbackCalls, repo.devices[registerDeviceID].Name)
	t.Logf("Duration: %s", time.Since(start))

	if !errors.Is(err, ErrRegisterFailed) {
		t.Fatalf("expected ErrRegisterFailed, got %v", err)
	}
	if repo.createCalls != 1 || repo.updateCalls != 1 || repo.findCalls != 1 || uow.tx.commitCalls != 0 {
		t.Fatalf("expected create=1 update=1 find=1 commit=0, got create=%d update=%d find=%d commit=%d",
			repo.createCalls, repo.updateCalls, repo.findCalls, uow.tx.commitCalls)
	}
}

func TestRegisterUseCase_CommitFailureHasStructuredLog(t *testing.T) {
	start := time.Now()
	commitErr := errors.New("commit failed")
	repo := newRegisterDeviceRepoStub()
	uow := &registerUOWStub{tx: &registerTxStub{commitErr: commitErr}}
	uc := NewRegisterUseCase(uow, repo)
	input := registerUseCaseInput(registerDeviceID, "macos", "Hiro MacBook")

	t.Log("Given: transaction commit returns an error after device persistence succeeds")
	t.Logf("Input: device_id=%s platform=%s name=%q", input.Data.DeviceID, input.Data.Platform, input.Data.Name)
	t.Log("Action: execute register usecase commit failure path")

	out, err := uc.Execute(context.Background(), input)

	t.Logf("Output: out=%+v err=%v", out, err)
	t.Logf("Mutation: create_calls=%d update_calls=%d find_calls=%d commit_calls=%d rollback_calls=%d rows=%d",
		repo.createCalls, repo.updateCalls, repo.findCalls, uow.tx.commitCalls, uow.tx.rollbackCalls, len(repo.devices))
	t.Logf("Duration: %s", time.Since(start))

	if !errors.Is(err, commitErr) {
		t.Fatalf("expected commit error, got %v", err)
	}
	if repo.createCalls != 1 || repo.findCalls != 1 || uow.tx.commitCalls != 1 {
		t.Fatalf("expected create=1 find=1 commit=1, got create=%d find=%d commit=%d",
			repo.createCalls, repo.findCalls, uow.tx.commitCalls)
	}
}
