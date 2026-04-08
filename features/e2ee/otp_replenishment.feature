Feature: E2EE OTP pre-key replenishment and consumption
  OTP pre-keys are consumed by key-bundle requests and replenished by low-watermark notifications.

  Background:
    Given login state is clean
    And E2EE key bootstrap state is clean
    And a registered login device "11111111-1111-1111-1111-111111111111" named "Hiro Mac" exists
    And an "active" account exists for login with email "e2ee-otp@example.com", account "e2eeotp", and password "correct-password"
    When I login with identifier "e2ee-otp@example.com", password "correct-password", and device "11111111-1111-1111-1111-111111111111"
    Then the response status should be 200
    And login response should include a bearer token

  Scenario: Key bundle consumes one available OTP pre-key
    When I upload E2EE identity key "alpha" for device "11111111-1111-1111-1111-111111111111"
    Then the response status should be 200
    When I upload E2EE signed pre-key "alpha" with key id 1 for device "11111111-1111-1111-1111-111111111111"
    Then the response status should be 204
    When I upload 3 E2EE one-time pre-keys starting at key id 10 for device "11111111-1111-1111-1111-111111111111"
    Then the response status should be 200
    When I request E2EE key bundle for the logged in user and device "11111111-1111-1111-1111-111111111111"
    Then the response status should be 200
    And E2EE key bundle should include OTP key id 10 and server should have 2 OTP keys for device "11111111-1111-1111-1111-111111111111"

  Scenario: Key bundle falls back to identity and signed pre-key when OTP pool is empty
    When I upload E2EE identity key "alpha" for device "11111111-1111-1111-1111-111111111111"
    Then the response status should be 200
    When I upload E2EE signed pre-key "alpha" with key id 1 for device "11111111-1111-1111-1111-111111111111"
    Then the response status should be 204
    When I request E2EE key bundle for the logged in user and device "11111111-1111-1111-1111-111111111111"
    Then the response status should be 200
    And E2EE key bundle should omit OTP and server should have 0 OTP keys for device "11111111-1111-1111-1111-111111111111"

  Scenario: Key bundle queues OTP replenish when remaining count is below threshold
    When I upload E2EE identity key "alpha" for device "11111111-1111-1111-1111-111111111111"
    Then the response status should be 200
    When I upload E2EE signed pre-key "alpha" with key id 1 for device "11111111-1111-1111-1111-111111111111"
    Then the response status should be 204
    When I upload 5 E2EE one-time pre-keys starting at key id 1 for device "11111111-1111-1111-1111-111111111111"
    Then the response status should be 200
    When I request E2EE key bundle for the logged in user and device "11111111-1111-1111-1111-111111111111"
    Then the response status should be 200
    And E2EE OTP replenish event should be queued for device "11111111-1111-1111-1111-111111111111"

  Scenario: Key bundle does not queue replenish when remaining count equals threshold
    When I upload E2EE identity key "alpha" for device "11111111-1111-1111-1111-111111111111"
    Then the response status should be 200
    When I upload E2EE signed pre-key "alpha" with key id 1 for device "11111111-1111-1111-1111-111111111111"
    Then the response status should be 204
    When I upload 6 E2EE one-time pre-keys starting at key id 1 for device "11111111-1111-1111-1111-111111111111"
    Then the response status should be 200
    When I request E2EE key bundle for the logged in user and device "11111111-1111-1111-1111-111111111111"
    Then the response status should be 200
    And no E2EE OTP replenish event should be queued
