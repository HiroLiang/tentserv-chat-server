package device

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	bddsupport "github.com/HiroLiang/tentserv-chat-server/features/support"
	"github.com/cucumber/godog"
)

type steps struct {
	*bddsupport.APITestContext
	deps  *Deps
	start time.Time
}

func RegisterSteps(ctx *godog.ScenarioContext, apiCtx *bddsupport.APITestContext, deps *Deps) {
	s := &steps{APITestContext: apiCtx, deps: deps}

	ctx.Step(`^device lifecycle state is clean$`, s.deviceLifecycleStateIsClean)
	ctx.Step(`^a device exists with id "([^"]*)", name "([^"]*)", and platform "([^"]*)"$`, s.aDeviceExists)
	ctx.Step(`^I register a device with id "([^"]*)", name "([^"]*)", and platform "([^"]*)"$`, s.iRegisterADevice)
	ctx.Step(`^I update device "([^"]*)" with name "([^"]*)" and platform "([^"]*)"$`, s.iUpdateADevice)
	ctx.Step(`^the device register mutation should create one device without update$`, s.theDeviceRegisterMutationShouldCreateOneDeviceWithoutUpdate)
	ctx.Step(`^the device register mutation should update an existing device$`, s.theDeviceRegisterMutationShouldUpdateAnExistingDevice)
	ctx.Step(`^the device register mutation should not create or update a device$`, s.theDeviceRegisterMutationShouldNotCreateOrUpdateADevice)
	ctx.Step(`^the device update mutation should update one device$`, s.theDeviceUpdateMutationShouldUpdateOneDevice)
	ctx.Step(`^the device update mutation should not touch repository$`, s.theDeviceUpdateMutationShouldNotTouchRepository)
	ctx.Step(`^the device update mutation should find once without update$`, s.theDeviceUpdateMutationShouldFindOnceWithoutUpdate)
	ctx.Step(`^the device "([^"]*)" should remain name "([^"]*)" and platform "([^"]*)"$`, s.theDeviceShouldRemain)
	ctx.Step(`^the device response should include id "([^"]*)", name "([^"]*)", platform "([^"]*)"$`, s.theDeviceResponseShouldInclude)
	ctx.Step(`^the device response should include created_at "([^"]*)"$`, s.theDeviceResponseShouldIncludeCreatedAt)
	ctx.Step(`^the device response should include updated_at "([^"]*)"$`, s.theDeviceResponseShouldIncludeUpdatedAt)
}

func (a *steps) deviceLifecycleStateIsClean() error {
	a.start = time.Now()
	fmt.Println("Given: device lifecycle state is clean")
	fmt.Printf("Input: existing_devices=%d\n", len(a.deps.deviceRepo.devices))
	fmt.Println("Action: reset in-memory device dependencies")
	fmt.Println("Output: clean device lifecycle state")
	fmt.Printf("Mutation: devices=%d create_calls=%d update_calls=%d find_calls=%d\n",
		len(a.deps.deviceRepo.devices), a.deps.deviceRepo.createCalls, a.deps.deviceRepo.updateCalls, a.deps.deviceRepo.findCalls)
	fmt.Printf("Duration: %s\n", time.Since(a.start))
	return nil
}

func (a *steps) aDeviceExists(deviceID, name, platform string) error {
	a.start = time.Now()
	fmt.Println("Given: an existing device row is seeded")
	fmt.Printf("Input: device_id=%s device_name=%q platform=%s\n", deviceID, name, platform)

	if err := a.deps.deviceRepo.seed(deviceID, name, platform); err != nil {
		return err
	}

	fmt.Println("Action: seed device repository")
	fmt.Printf("Output: existing_devices=%d\n", len(a.deps.deviceRepo.devices))
	fmt.Printf("Mutation: devices=%d\n", len(a.deps.deviceRepo.devices))
	fmt.Printf("Duration: %s\n", time.Since(a.start))
	return nil
}

func (a *steps) iRegisterADevice(deviceID, name, platform string) error {
	a.start = time.Now()
	fmt.Println("Given: device registration HTTP endpoint is available")
	fmt.Printf("Input: device_id=%s device_name=%q platform=%s\n", deviceID, name, platform)
	fmt.Println("Action: POST /api/device/register")

	payload := map[string]string{
		"device_id":   deviceID,
		"device_name": name,
		"platform":    platform,
	}
	if err := a.DoJSONRequest(http.MethodPost, "/api/device/register", payload); err != nil {
		return err
	}

	fmt.Printf("Output: status=%d body=%s\n", a.Response.StatusCode, string(a.ResponseBody))
	fmt.Printf("Mutation: device_create_calls=%d device_update_calls=%d device_find_calls=%d devices=%d\n",
		a.deps.deviceRepo.createCalls, a.deps.deviceRepo.updateCalls, a.deps.deviceRepo.findCalls, len(a.deps.deviceRepo.devices))
	fmt.Printf("Duration: %s\n", time.Since(a.start))
	return nil
}

func (a *steps) iUpdateADevice(deviceID, name, platform string) error {
	a.start = time.Now()
	fmt.Println("Given: device update HTTP endpoint is available")
	fmt.Printf("Input: path_device_id=%s payload_device_name=%q payload_platform=%s\n", deviceID, name, platform)
	fmt.Printf("Action: PATCH /api/device/%s\n", deviceID)

	payload := map[string]string{
		"device_name": name,
		"platform":    platform,
	}
	if err := a.DoJSONRequest(http.MethodPatch, "/api/device/"+deviceID, payload); err != nil {
		return err
	}

	fmt.Printf("Output: status=%d body=%s\n", a.Response.StatusCode, string(a.ResponseBody))
	fmt.Printf("Mutation: device_create_calls=%d device_update_calls=%d device_find_calls=%d updated_ids=%v devices=%d\n",
		a.deps.deviceRepo.createCalls, a.deps.deviceRepo.updateCalls, a.deps.deviceRepo.findCalls, a.deps.deviceRepo.updatedIDs, len(a.deps.deviceRepo.devices))
	fmt.Printf("Duration: %s\n", time.Since(a.start))
	return nil
}

func (a *steps) theDeviceRegisterMutationShouldCreateOneDeviceWithoutUpdate() error {
	start := time.Now()
	fmt.Println("Given: device registration should insert a missing device")
	fmt.Println("Input: expecting create=1 update=0 find=1")
	fmt.Println("Action: inspect in-memory device counters")

	matches := a.deps.deviceRepo.createCalls == 1 &&
		a.deps.deviceRepo.updateCalls == 0 &&
		a.deps.deviceRepo.findCalls == 1 &&
		len(a.deps.deviceRepo.devices) == 1

	fmt.Printf("Output: mutation_match=%t\n", matches)
	fmt.Printf("Mutation: device_create_calls=%d device_update_calls=%d device_find_calls=%d devices=%d\n",
		a.deps.deviceRepo.createCalls, a.deps.deviceRepo.updateCalls, a.deps.deviceRepo.findCalls, len(a.deps.deviceRepo.devices))
	fmt.Printf("Duration: %s\n", time.Since(start))

	if !matches {
		return fmt.Errorf("expected device insert mutation create=1 update=0 find=1 devices=1")
	}
	return nil
}

func (a *steps) theDeviceRegisterMutationShouldUpdateAnExistingDevice() error {
	start := time.Now()
	fmt.Println("Given: device registration should upsert an existing device")
	fmt.Println("Input: expecting create=1 update=1 find=2")
	fmt.Println("Action: inspect in-memory device counters")

	matches := a.deps.deviceRepo.createCalls == 1 &&
		a.deps.deviceRepo.updateCalls == 1 &&
		a.deps.deviceRepo.findCalls == 2 &&
		len(a.deps.deviceRepo.devices) == 1

	fmt.Printf("Output: mutation_match=%t\n", matches)
	fmt.Printf("Mutation: device_create_calls=%d device_update_calls=%d device_find_calls=%d updated_ids=%v devices=%d\n",
		a.deps.deviceRepo.createCalls, a.deps.deviceRepo.updateCalls, a.deps.deviceRepo.findCalls, a.deps.deviceRepo.updatedIDs, len(a.deps.deviceRepo.devices))
	fmt.Printf("Duration: %s\n", time.Since(start))

	if !matches {
		return fmt.Errorf("expected device upsert mutation create=1 update=1 find=2 devices=1")
	}
	return nil
}

func (a *steps) theDeviceRegisterMutationShouldNotCreateOrUpdateADevice() error {
	start := time.Now()
	fmt.Println("Given: device registration should fail before repository mutation")
	fmt.Println("Input: expecting create=0 update=0 find=0")
	fmt.Println("Action: inspect in-memory device counters")

	noMutation := a.deps.deviceRepo.createCalls == 0 &&
		a.deps.deviceRepo.updateCalls == 0 &&
		a.deps.deviceRepo.findCalls == 0

	fmt.Printf("Output: no_mutation=%t\n", noMutation)
	fmt.Printf("Mutation: device_create_calls=%d device_update_calls=%d device_find_calls=%d devices=%d\n",
		a.deps.deviceRepo.createCalls, a.deps.deviceRepo.updateCalls, a.deps.deviceRepo.findCalls, len(a.deps.deviceRepo.devices))
	fmt.Printf("Duration: %s\n", time.Since(start))

	if !noMutation {
		return fmt.Errorf("expected no device registration repository mutation")
	}
	return nil
}

func (a *steps) theDeviceUpdateMutationShouldUpdateOneDevice() error {
	start := time.Now()
	fmt.Println("Given: device update should mutate one device")
	fmt.Println("Input: expecting find=2 update=1")
	fmt.Println("Action: inspect in-memory device counters")

	matches := a.deps.deviceRepo.findCalls == 2 &&
		a.deps.deviceRepo.updateCalls == 1 &&
		len(a.deps.deviceRepo.updatedIDs) == 1

	fmt.Printf("Output: mutation_match=%t\n", matches)
	fmt.Printf("Mutation: device_create_calls=%d device_update_calls=%d device_find_calls=%d updated_ids=%v\n",
		a.deps.deviceRepo.createCalls, a.deps.deviceRepo.updateCalls, a.deps.deviceRepo.findCalls, a.deps.deviceRepo.updatedIDs)
	fmt.Printf("Duration: %s\n", time.Since(start))

	if !matches {
		return fmt.Errorf("expected update mutation find=2 update=1 updated_ids=1")
	}
	return nil
}

func (a *steps) theDeviceUpdateMutationShouldNotTouchRepository() error {
	start := time.Now()
	fmt.Println("Given: device update should fail before repository access")
	fmt.Println("Input: expecting find=0 update=0")
	fmt.Println("Action: inspect in-memory device counters")

	noTouch := a.deps.deviceRepo.findCalls == 0 && a.deps.deviceRepo.updateCalls == 0

	fmt.Printf("Output: repository_untouched=%t\n", noTouch)
	fmt.Printf("Mutation: device_create_calls=%d device_update_calls=%d device_find_calls=%d devices=%d\n",
		a.deps.deviceRepo.createCalls, a.deps.deviceRepo.updateCalls, a.deps.deviceRepo.findCalls, len(a.deps.deviceRepo.devices))
	fmt.Printf("Duration: %s\n", time.Since(start))

	if !noTouch {
		return fmt.Errorf("expected device update to avoid repository access")
	}
	return nil
}

func (a *steps) theDeviceUpdateMutationShouldFindOnceWithoutUpdate() error {
	start := time.Now()
	fmt.Println("Given: device update should read once and avoid mutation")
	fmt.Println("Input: expecting find=1 update=0")
	fmt.Println("Action: inspect in-memory device counters")

	matches := a.deps.deviceRepo.findCalls == 1 && a.deps.deviceRepo.updateCalls == 0

	fmt.Printf("Output: mutation_match=%t\n", matches)
	fmt.Printf("Mutation: device_create_calls=%d device_update_calls=%d device_find_calls=%d devices=%d\n",
		a.deps.deviceRepo.createCalls, a.deps.deviceRepo.updateCalls, a.deps.deviceRepo.findCalls, len(a.deps.deviceRepo.devices))
	fmt.Printf("Duration: %s\n", time.Since(start))

	if !matches {
		return fmt.Errorf("expected device update find=1 update=0")
	}
	return nil
}

func (a *steps) theDeviceShouldRemain(deviceID, name, platform string) error {
	start := time.Now()
	fmt.Println("Given: a device row should retain its previous mutable fields")
	fmt.Printf("Input: device_id=%s expected_name=%q expected_platform=%s\n", deviceID, name, platform)
	fmt.Println("Action: inspect in-memory device row")

	d, ok := a.deps.deviceRepo.devices[deviceID]
	if !ok {
		return fmt.Errorf("device %s not found", deviceID)
	}
	matches := d.Name == name && d.Platform.String() == platform

	fmt.Printf("Output: actual_name=%q actual_platform=%s match=%t\n", d.Name, d.Platform.String(), matches)
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(start))

	if !matches {
		return fmt.Errorf("expected device %s to remain %q/%s, got %q/%s", deviceID, name, platform, d.Name, d.Platform.String())
	}
	return nil
}

func (a *steps) theDeviceResponseShouldInclude(deviceID, name, platform string) error {
	start := time.Now()
	fmt.Println("Given: device response body should include identity fields")
	fmt.Printf("Input: expected_device_id=%s expected_name=%q expected_platform=%s\n", deviceID, name, platform)
	fmt.Println("Action: decode and compare device response")

	body, err := a.deviceResponseBody()
	if err != nil {
		return err
	}
	actualID := responseStringField(body, "device_id")
	actualName := responseStringField(body, "device_name")
	actualPlatform := responseStringField(body, "platform")
	matches := actualID == deviceID &&
		actualName == name &&
		actualPlatform == platform

	fmt.Printf("Output: actual_device_id=%s actual_name=%q actual_platform=%s match=%t\n",
		actualID, actualName, actualPlatform, matches)
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(start))

	if !matches {
		return fmt.Errorf("expected device response %s/%q/%s, got %s/%q/%s",
			deviceID, name, platform, actualID, actualName, actualPlatform)
	}
	return nil
}

func (a *steps) theDeviceResponseShouldIncludeCreatedAt(createdAt string) error {
	return a.theDeviceResponseShouldIncludeTimestamp("created_at", createdAt)
}

func (a *steps) theDeviceResponseShouldIncludeUpdatedAt(updatedAt string) error {
	return a.theDeviceResponseShouldIncludeTimestamp("updated_at", updatedAt)
}

func (a *steps) theDeviceResponseShouldIncludeTimestamp(field, expected string) error {
	start := time.Now()
	fmt.Println("Given: device response body should include a timestamp")
	fmt.Printf("Input: field=%s expected_value=%s\n", field, expected)
	fmt.Println("Action: decode and compare device timestamp")

	body, err := a.deviceResponseBody()
	if err != nil {
		return err
	}
	actual := responseStringField(body, field)
	matches := actual == expected

	fmt.Printf("Output: actual_value=%s match=%t\n", actual, matches)
	fmt.Println("Mutation: none")
	fmt.Printf("Duration: %s\n", time.Since(start))

	if !matches {
		return fmt.Errorf("expected %s %s, got %s", field, expected, actual)
	}
	return nil
}

func (a *steps) deviceResponseBody() (map[string]any, error) {
	var body map[string]any
	if err := json.Unmarshal(a.ResponseBody, &body); err != nil {
		return nil, fmt.Errorf("decode device response: %w; body=%s", err, string(a.ResponseBody))
	}
	return body, nil
}

func responseStringField(body map[string]any, field string) string {
	if value, ok := body[field].(string); ok {
		return value
	}
	return ""
}
