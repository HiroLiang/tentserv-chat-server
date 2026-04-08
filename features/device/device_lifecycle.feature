Feature: Device lifecycle
  The desktop app registers and updates the local device through the backend HTTP API.

  Scenario: Register a missing device
    Given device lifecycle state is clean
    When I register a device with id "550e8400-e29b-41d4-a716-446655440000", name "Hiro MacBook", and platform "macos"
    Then the response status should be 201
    And the device response should include id "550e8400-e29b-41d4-a716-446655440000", name "Hiro MacBook", platform "macos"
    And the device response should include created_at "2026-04-08T01:02:03Z"
    And the device register mutation should create one device without update

  Scenario: Register an existing device updates mutable fields
    Given a device exists with id "550e8400-e29b-41d4-a716-446655440000", name "Old Workstation", and platform "linux"
    When I register a device with id "550e8400-e29b-41d4-a716-446655440000", name "Renamed MacBook", and platform "macos"
    Then the response status should be 201
    And the device response should include id "550e8400-e29b-41d4-a716-446655440000", name "Renamed MacBook", platform "macos"
    And the device response should include created_at "2026-04-07T01:02:03Z"
    And the device register mutation should update an existing device

  Scenario: Reject device registration with invalid UUID
    Given device lifecycle state is clean
    When I register a device with id "not-a-uuid", name "Invalid Device", and platform "macos"
    Then the response status should be 400
    And the device register mutation should not create or update a device

  Scenario: Reject device registration with invalid platform
    Given device lifecycle state is clean
    When I register a device with id "550e8400-e29b-41d4-a716-446655440000", name "Invalid Platform", and platform "beos"
    Then the response status should be 400
    And the device register mutation should not create or update a device

  Scenario: Update an existing device
    Given a device exists with id "660e8400-e29b-41d4-a716-446655440000", name "Old Device", and platform "linux"
    When I update device "660e8400-e29b-41d4-a716-446655440000" with name "Renamed Device" and platform "macos"
    Then the response status should be 200
    And the device response should include id "660e8400-e29b-41d4-a716-446655440000", name "Renamed Device", platform "macos"
    And the device response should include updated_at "2026-04-08T09:10:11Z"
    And the device update mutation should update one device

  Scenario: Reject device update with invalid path UUID
    Given device lifecycle state is clean
    When I update device "not-a-uuid" with name "Invalid Device" and platform "macos"
    Then the response status should be 400
    And the device update mutation should not touch repository

  Scenario: Reject device update for a missing device
    Given device lifecycle state is clean
    When I update device "660e8400-e29b-41d4-a716-446655440000" with name "Missing Device" and platform "macos"
    Then the response status should be 404
    And the device update mutation should find once without update

  Scenario: Reject device update with invalid platform
    Given a device exists with id "660e8400-e29b-41d4-a716-446655440000", name "Old Device", and platform "linux"
    When I update device "660e8400-e29b-41d4-a716-446655440000" with name "Invalid Platform" and platform "beos"
    Then the response status should be 400
    And the device update mutation should find once without update
    And the device "660e8400-e29b-41d4-a716-446655440000" should remain name "Old Device" and platform "linux"
