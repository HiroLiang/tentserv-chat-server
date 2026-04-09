Feature: E2EE sender key request distribution
  Background:
    Given login state is clean
    And a registered login device "22222222-2222-2222-2222-222222222222" named "SKR Test Device" exists
    And an "active" account exists for login with email "skr_hiro@example.com", account "skr_hiro_account", and password "skr-password"
    When I login with identifier "skr_hiro_account", password "skr-password", and device "22222222-2222-2222-2222-222222222222"
    Then the response status should be 200
    And login response should include a bearer token
    Given E2EE keys are bootstrapped for the logged in user

  Scenario: Sender key request is created and stored when provider does not yet have a key
    Given a room member setup exists with room id 1, caller member id 201, and provider member id 202 in the same room with no existing sender key
    When I create a sender key request for room 1 and provider member 202
    Then the response status should be 204
    And a sender key request row should exist from member 201 to provider 202

  Scenario: Sender key request is skipped when a latest distribution is already available for the caller
    Given a room member setup exists with room id 2, caller member id 203, and provider member id 204 in the same room with an available latest distribution for the caller
    When I create a sender key request for room 2 and provider member 204
    Then the response status should be 204
    And no sender key request row should exist from member 203 to provider 204

  Scenario: Sender key request is rejected when provider member belongs to a different room
    Given a room member setup exists with room id 3, caller member id 205, and provider member id 206 where provider is in a different room
    When I create a sender key request for room 3 and provider member 206
    Then the response status should be 403
    And the response error code should be "NOT_ROOM_MEMBER"

  Scenario: Sender key request is rejected when caller is not a member of the room
    Given a room member setup exists with room id 4, provider member id 208 in that room, and the caller has no room membership
    When I create a sender key request for room 4 and provider member 208
    Then the response status should be 403
    And the response error code should be "NOT_ROOM_MEMBER"

  Scenario: Sender key request is rejected when caller and provider are blocked
    Given a room member setup exists with room id 6, caller member id 211, and provider member id 212 in the same room with no existing sender key
    And caller member 211 and provider member 212 are blocked from each other
    When I create a sender key request for room 6 and provider member 212
    Then the response status should be 403
    And the response error code should be "FORBIDDEN"

  Scenario: Repeated sender key request stays idempotent
    Given a room member setup exists with room id 5, caller member id 209, and provider member id 210 in the same room with no existing sender key
    When I create a sender key request for room 5 and provider member 210
    Then the response status should be 204
    When I create a sender key request for room 5 and provider member 210
    Then the response status should be 204
    And pending sender key request count from member 209 to provider 210 should be 1
