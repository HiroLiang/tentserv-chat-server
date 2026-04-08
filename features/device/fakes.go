package device

import (
	"context"
	"sync"
	"time"

	deviceUseCase "github.com/HiroLiang/tentserv-chat-server/internal/application/device/usecase"
	domaindevice "github.com/HiroLiang/tentserv-chat-server/internal/domain/device"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/transaction"
)

type Deps struct {
	deviceRepo *bddDeviceRepo
}

func NewDeps() *Deps {
	deps := &Deps{deviceRepo: newBDDDeviceRepo()}
	deps.Reset()
	return deps
}

func (d *Deps) Reset() {
	d.deviceRepo.reset()
}

func (d *Deps) RegisterUseCases(
	uow transaction.UnitOfWork,
) (*deviceUseCase.RegisterUseCase, *deviceUseCase.UpdateDeviceUseCase) {
	registerUseCase := deviceUseCase.NewRegisterUseCase(uow, d.deviceRepo)
	updateUseCase := deviceUseCase.NewUpdateDeviceUseCase(d.deviceRepo)
	return registerUseCase, updateUseCase
}

type bddDeviceRepo struct {
	mu          sync.Mutex
	devices     map[string]*domaindevice.Device
	createCalls int
	findCalls   int
	updateCalls int
	updatedIDs  []string
	insertedAt  time.Time
	updatedAt   time.Time
}

func newBDDDeviceRepo() *bddDeviceRepo {
	return &bddDeviceRepo{
		insertedAt: time.Date(2026, 4, 8, 1, 2, 3, 0, time.UTC),
		updatedAt:  time.Date(2026, 4, 8, 9, 10, 11, 0, time.UTC),
	}
}

func (r *bddDeviceRepo) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.devices = map[string]*domaindevice.Device{}
	r.createCalls = 0
	r.findCalls = 0
	r.updateCalls = 0
	r.updatedIDs = nil
}

func (r *bddDeviceRepo) seed(deviceID, name, platform string) error {
	parsedID, err := shared.ParseDeviceID(deviceID)
	if err != nil {
		return err
	}
	parsedPlatform, err := domaindevice.ParsePlatform(platform)
	if err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	createdAt := time.Date(2026, 4, 7, 1, 2, 3, 0, time.UTC)
	r.devices[deviceID] = &domaindevice.Device{
		ID:        parsedID,
		Platform:  parsedPlatform,
		Name:      name,
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}
	return nil
}

func (r *bddDeviceRepo) FindByID(_ context.Context, deviceID shared.DeviceID) (*domaindevice.Device, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.findCalls++
	d, ok := r.devices[deviceID.String()]
	if !ok {
		return nil, domaindevice.ErrDeviceNotFound
	}
	return cloneBDDDevice(d), nil
}

func (r *bddDeviceRepo) FindAllByAccountID(context.Context, shared.AccountID) ([]*domaindevice.Device, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	devices := make([]*domaindevice.Device, 0, len(r.devices))
	for _, d := range r.devices {
		devices = append(devices, cloneBDDDevice(d))
	}
	return devices, nil
}

func (r *bddDeviceRepo) Create(_ context.Context, d *domaindevice.Device) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.createCalls++
	if _, ok := r.devices[d.ID.String()]; ok {
		return domaindevice.ErrDeviceAlreadyExists
	}

	created := cloneBDDDevice(d)
	created.CreatedAt = r.insertedAt
	created.UpdatedAt = r.insertedAt
	r.devices[d.ID.String()] = created
	return nil
}

func (r *bddDeviceRepo) Update(_ context.Context, d *domaindevice.Device) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.updateCalls++
	r.updatedIDs = append(r.updatedIDs, d.ID.String())
	if _, ok := r.devices[d.ID.String()]; !ok {
		return domaindevice.ErrDeviceNotFound
	}

	updated := cloneBDDDevice(d)
	updated.UpdatedAt = r.updatedAt
	r.devices[d.ID.String()] = updated
	return nil
}

func (r *bddDeviceRepo) BindAccount(context.Context, shared.DeviceID, shared.AccountID) error {
	return nil
}

func (r *bddDeviceRepo) DeleteByAccount(context.Context, shared.DeviceID, shared.AccountID) error {
	return nil
}

func cloneBDDDevice(d *domaindevice.Device) *domaindevice.Device {
	if d == nil {
		return nil
	}
	return &domaindevice.Device{
		ID:        d.ID,
		Platform:  d.Platform,
		Name:      d.Name,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
}

var _ domaindevice.Repository = (*bddDeviceRepo)(nil)
