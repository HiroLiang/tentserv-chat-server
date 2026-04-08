Feature: E2EE key bootstrap after login
  Login-time E2EE key bootstrap must expose non-consuming public key status and a server OTP policy.

  Background:
    Given login state is clean
    And E2EE key bootstrap state is clean
    And a registered login device "11111111-1111-1111-1111-111111111111" named "Hiro Mac" exists
    And an "active" account exists for login with email "e2ee-login@example.com", account "e2eelogin", and password "correct-password"
    When I login with identifier "e2ee-login@example.com", password "correct-password", and device "11111111-1111-1111-1111-111111111111"
    Then the response status should be 200
    And login response should include a bearer token

  Scenario: Key policy exposes the default OTP bootstrap policy
    When I request the E2EE key policy using the login token
    Then the response status should be 200
    And the E2EE key policy should be target 20 and threshold 5

  Scenario: Empty remote key status is public and non-consuming
    When I request E2EE key status for the logged in user and device "11111111-1111-1111-1111-111111111111"
    Then the response status should be 200
    And E2EE key status should be empty for device "11111111-1111-1111-1111-111111111111"
    And the E2EE key status check should not consume OTP keys

  Scenario: Uploaded identity, signed pre-key, and OTP keys can be read back by status
    When I upload E2EE identity key "alpha" for device "11111111-1111-1111-1111-111111111111"
    Then the response status should be 200
    When I upload E2EE signed pre-key "alpha" with key id 1 for device "11111111-1111-1111-1111-111111111111"
    Then the response status should be 204
    When I upload 3 E2EE one-time pre-keys starting at key id 1 for device "11111111-1111-1111-1111-111111111111"
    Then the response status should be 200
    When I request E2EE key status for the logged in user and device "11111111-1111-1111-1111-111111111111"
    Then the response status should be 200
    And E2EE key status should expose identity "alpha", signed pre-key "alpha", key id 1, and 3 OTP keys for device "11111111-1111-1111-1111-111111111111"
    And the E2EE key status check should not consume OTP keys

  Scenario: Re-uploaded public keys replace mismatched remote material
    When I upload E2EE identity key "alpha" for device "11111111-1111-1111-1111-111111111111"
    Then the response status should be 200
    When I upload E2EE signed pre-key "alpha" with key id 1 for device "11111111-1111-1111-1111-111111111111"
    Then the response status should be 204
    When I upload 2 E2EE one-time pre-keys starting at key id 10 for device "11111111-1111-1111-1111-111111111111"
    Then the response status should be 200
    When I upload E2EE identity key "beta" for device "11111111-1111-1111-1111-111111111111"
    Then the response status should be 200
    When I upload E2EE signed pre-key "beta" with key id 2 for device "11111111-1111-1111-1111-111111111111"
    Then the response status should be 204
    When I request E2EE key status for the logged in user and device "11111111-1111-1111-1111-111111111111"
    Then the response status should be 200
    And E2EE key status should expose identity "beta", signed pre-key "beta", key id 2, and 2 OTP keys for device "11111111-1111-1111-1111-111111111111"

  Scenario: Invalid identity key payload is rejected without writing key rows
    When I upload an invalid E2EE identity key for device "11111111-1111-1111-1111-111111111111"
    Then the response status should be 400
    And the response error code should be "INVALID_SIGNATURE"
    And E2EE key repositories should remain empty

  Scenario: Invalid signed pre-key payload is rejected without writing key rows
    When I upload an invalid E2EE signed pre-key for device "11111111-1111-1111-1111-111111111111"
    Then the response status should be 400
    And the response error code should be "INVALID_SIGNATURE"
    And E2EE key repositories should remain empty

  Scenario: Invalid OTP pre-key payload is rejected without writing key rows
    When I upload invalid E2EE one-time pre-keys for device "11111111-1111-1111-1111-111111111111"
    Then the response status should be 400
    And the response error code should be "INVALID_SIGNATURE"
    And E2EE key repositories should remain empty
