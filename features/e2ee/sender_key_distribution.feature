Feature: E2EE sender key distribution lifecycle
  Background:
    Given login state is clean
    And a registered login device "33333333-3333-3333-3333-333333333333" named "SKD Test Device" exists
    And an "active" account exists for login with email "skd_hiro@example.com", account "skd_hiro_account", and password "skd-password"
    When I login with identifier "skd_hiro_account", password "skd-password", and device "33333333-3333-3333-3333-333333333333"
    Then the response status should be 200
    And login response should include a bearer token
    Given E2EE keys are bootstrapped for the logged in user

  Scenario: Provider uploads a latest sender key distribution for one receiver with a millisecond sender key version
    Given a sender key provider setup exists with room id 11, provider member id 301, and receiver member id 302 in the same room
    When I provide a sender key distribution for room 11 to receiver member 302 with sender key version 1775758701055
    Then the response status should be 204
    When I request sender key distribution status for room 11
    Then the response status should be 200
    And sender key distribution status should show own key exists as true
    And sender key distribution status should list available receiver member 302

  Scenario: Receiver sees an available sender key distribution in room summary and pending list
    Given a sender key receiver setup exists with room id 12, sender member id 304, and receiver member id 303 with available distribution version 77
    When I request sender key distribution status for room 12
    Then the response status should be 200
    And sender key distribution status should show own key exists as false
    And sender key distribution status should list available sender member 304
    When I list pending sender key distributions for room 12
    Then the response status should be 200
    And pending sender key distributions should include sender member 304, receiver member 303, and version 77

  Scenario: Repeated upload of the same sender key version stays idempotent
    Given a sender key provider setup exists with room id 15, provider member id 309, and receiver member id 310 in the same room
    When I provide a sender key distribution for room 15 to receiver member 310 with sender key version 1776018315645
    Then the response status should be 204
    When I provide a sender key distribution for room 15 to receiver member 310 with sender key version 1776018315645
    Then the response status should be 204
    When I request sender key distribution status for room 15
    Then the response status should be 200
    And sender key distribution status should list available receiver member 310

  Scenario: Receiver consumes an available sender key distribution
    Given a sender key receiver setup exists with room id 13, sender member id 306, and receiver member id 305 with available distribution version 88
    When I list pending sender key distributions for room 13
    Then the response status should be 200
    And pending sender key distributions should include sender member 306, receiver member 305, and version 88
    When I mark the first pending sender key distribution as "consumed"
    Then the response status should be 204
    And the first pending sender key distribution should now be "consumed"

  Scenario: Failed consume requeues a sender key request and notifies provider
    Given a sender key receiver setup exists with room id 14, sender member id 308, and receiver member id 307 with available distribution version 99
    When I list pending sender key distributions for room 14
    Then the response status should be 200
    And pending sender key distributions should include sender member 308, receiver member 307, and version 99
    When I mark the first pending sender key distribution as "failed"
    Then the response status should be 204
    And the first pending sender key distribution should now be "failed"
    And a sender key request row should exist from member 307 to provider 308
    And an e2ee.sender_key_needed event should have been broadcast for provider member 308
