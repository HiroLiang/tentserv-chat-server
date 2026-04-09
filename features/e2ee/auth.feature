Feature: E2EE endpoints require authentication
  All /api/e2ee endpoints should reject missing or revoked bearer tokens with 401.

  Background:
    Given login state is clean
    And E2EE key bootstrap state is clean
    And a registered login device "11111111-1111-1111-1111-111111111111" named "Hiro Mac" exists
    And an "active" account exists for login with email "e2ee-auth@example.com", account "e2eeauth", and password "correct-password"
    When I login with identifier "e2ee-auth@example.com", password "correct-password", and device "11111111-1111-1111-1111-111111111111"
    Then the response status should be 200
    And login response should include a bearer token

  Scenario Outline: Protected E2EE endpoint rejects unauthenticated requests
    When I call the E2EE endpoint "<endpoint>" with "<auth_mode>" authentication
    Then the response status should be 401
    And the response error code should be "AUTH_FAILED"

    Examples:
      | endpoint                 | auth_mode     |
      | identity-key             | no token      |
      | identity-key             | revoked token |
      | signed-prekey            | no token      |
      | signed-prekey            | revoked token |
      | otp-prekeys              | no token      |
      | otp-prekeys              | revoked token |
      | otp-prekeys-count        | no token      |
      | otp-prekeys-count        | revoked token |
      | key-bundle               | no token      |
      | key-bundle               | revoked token |
      | key-status               | no token      |
      | key-status               | revoked token |
      | key-policy               | no token      |
      | key-policy               | revoked token |
      | sender-key               | no token      |
      | sender-key               | revoked token |
      | sender-keys              | no token      |
      | sender-keys              | revoked token |
      | sender-key-distributions | no token      |
      | sender-key-distributions | revoked token |
      | sender-key-distributions-pending | no token      |
      | sender-key-distributions-pending | revoked token |
      | sender-key-distributions-consume | no token      |
      | sender-key-distributions-consume | revoked token |
      | sender-key-request       | no token      |
      | sender-key-request       | revoked token |
