Feature: Login session
  LoginPage authenticates an account/email and restores the current user through the backend session API.

  Scenario: Login with email creates session, participant, login event, and profile
    Given login state is clean
    And a registered login device "11111111-1111-1111-1111-111111111111" named "Hiro's Mac" exists
    And an "active" account exists for login with email "login@example.com", account "login_account", and password "redacted-password"
    When I login with identifier "login@example.com", password "redacted-password", and device "11111111-1111-1111-1111-111111111111"
    Then the response status should be 200
    And login response should include a bearer token
    And login mutation should include session, device link, participant, and login event
    When I request my auth profile using the login token and device "11111111-1111-1111-1111-111111111111"
    Then the response status should be 200
    And auth profile response should describe the current login user

  Scenario: Login with email is case-insensitive
    Given login state is clean
    And a registered login device "11111111-1111-1111-1111-111111111111" named "Hiro's Mac" exists
    And an "active" account exists for login with email "login@example.com", account "login_account", and password "redacted-password"
    When I login with identifier "Login@Example.com", password "redacted-password", and device "11111111-1111-1111-1111-111111111111"
    Then the response status should be 200
    And login response should include a bearer token
    When I request my auth profile using the login token and device "11111111-1111-1111-1111-111111111111"
    Then the response status should be 200
    And auth profile response should describe the current login user

  Scenario: Login with account reuses an existing participant
    Given login state is clean
    And a registered login device "11111111-1111-1111-1111-111111111111" named "Hiro's Mac" exists
    And an "active" account exists for login with email "login@example.com", account "login_account", and password "redacted-password"
    And the login account has an existing participant
    When I login with identifier "login_account", password "redacted-password", and device "11111111-1111-1111-1111-111111111111"
    Then the response status should be 200
    And login response should include a bearer token
    And login mutation should reuse the existing participant

  Scenario: Login allows the same account on multiple devices simultaneously
    Given login state is clean
    And a registered login device "11111111-1111-1111-1111-111111111111" named "Hiro's Mac" exists
    And a registered login device "22222222-2222-2222-2222-222222222222" named "Hiro's iPhone" exists
    And an "active" account exists for login with email "login@example.com", account "login_account", and password "redacted-password"
    When I login with identifier "login@example.com", password "redacted-password", and device "11111111-1111-1111-1111-111111111111"
    Then the response status should be 200
    And I remember the login token as "device-a"
    When I login with identifier "login@example.com", password "redacted-password", and device "22222222-2222-2222-2222-222222222222"
    Then the response status should be 200
    And I remember the login token as "device-b"
    When I request my auth profile using remembered login token "device-a" and device "11111111-1111-1111-1111-111111111111"
    Then the response status should be 200
    And auth profile response should describe the current login user
    When I request my auth profile using remembered login token "device-b" and device "22222222-2222-2222-2222-222222222222"
    Then the response status should be 200
    And auth profile response should describe the current login user

  Scenario: Reject incorrect password
    Given login state is clean
    And a registered login device "11111111-1111-1111-1111-111111111111" named "Hiro's Mac" exists
    And an "active" account exists for login with email "login@example.com", account "login_account", and password "redacted-password"
    When I login with identifier "login@example.com", password "wrong-password", and device "11111111-1111-1111-1111-111111111111"
    Then the response status should be 401
    And the response error code should be "PASSWORD_ERROR"
    And login mutation should stop before session creation

  Scenario: Reject missing account
    Given login state is clean
    And a registered login device "11111111-1111-1111-1111-111111111111" named "Hiro's Mac" exists
    When I login with identifier "missing@example.com", password "redacted-password", and device "11111111-1111-1111-1111-111111111111"
    Then the response status should be 404
    And the response error code should be "ACCOUNT_NOT_FOUND"
    And login mutation should stop before session creation

  Scenario: Lock login after five consecutive failed attempts
    Given login state is clean
    And a registered login device "11111111-1111-1111-1111-111111111111" named "Hiro's Mac" exists
    And an "active" account exists for login with email "login@example.com", account "login_account", and password "redacted-password"
    When I attempt login 5 times with identifier "login@example.com", password "wrong-password", and device "11111111-1111-1111-1111-111111111111"
    Then the response status should be 401
    And the response error code should be "PASSWORD_ERROR"
    And the login identifier "login@example.com" should be locked
    When I login with identifier "login@example.com", password "redacted-password", and device "11111111-1111-1111-1111-111111111111"
    Then the response status should be 429
    And the response error code should be "LOGIN_LOCKED"
    And login mutation should stop before session creation

  Scenario Outline: Reject unavailable account status
    Given login state is clean
    And a registered login device "11111111-1111-1111-1111-111111111111" named "Hiro's Mac" exists
    And an "<status>" account exists for login with email "login@example.com", account "login_account", and password "redacted-password"
    When I login with identifier "login@example.com", password "redacted-password", and device "11111111-1111-1111-1111-111111111111"
    Then the response status should be <http_status>
    And the response error code should be "<code>"
    And login mutation should stop before session creation

    Examples:
      | status   | http_status | code             |
      | applying | 403         | ACCOUNT_APPLYING |
      | inactive | 403         | ACCOUNT_INACTIVE |
      | banned   | 403         | ACCOUNT_BANNED   |
      | deleted  | 404         | ACCOUNT_NOT_FOUND |

  Scenario: Reject invalid device id
    Given login state is clean
    And an "active" account exists for login with email "login@example.com", account "login_account", and password "redacted-password"
    When I login with identifier "login@example.com", password "redacted-password", and device "not-a-device-id"
    Then the response status should be 400
    And the response error code should be "INVALID_DEVICE_ID"
    And login mutation should stop before session creation

  Scenario: Reject invalid login payload
    Given login state is clean
    When I login with an invalid payload
    Then the response status should be 400
    And the response error code should be "INVALID_REQUEST"
    And login mutation should stop before session creation

  Scenario: Logout revokes the current session token
    Given login state is clean
    And a registered login device "11111111-1111-1111-1111-111111111111" named "Hiro's Mac" exists
    And an "active" account exists for login with email "login@example.com", account "login_account", and password "redacted-password"
    When I login with identifier "login@example.com", password "redacted-password", and device "11111111-1111-1111-1111-111111111111"
    Then the response status should be 200
    And I remember the login token as "logout-token"
    When I logout using remembered login token "logout-token" and device "11111111-1111-1111-1111-111111111111"
    Then the response status should be 200
    And logout mutation should revoke the remembered login token "logout-token"
    When I request my auth profile using remembered login token "logout-token" and device "11111111-1111-1111-1111-111111111111"
    Then the response status should be 401

  Scenario: Reject revoked session when requesting auth profile
    Given login state is clean
    And a registered login device "11111111-1111-1111-1111-111111111111" named "Hiro's Mac" exists
    And an "active" account exists for login with email "login@example.com", account "login_account", and password "redacted-password"
    When I login with identifier "login@example.com", password "redacted-password", and device "11111111-1111-1111-1111-111111111111"
    Then the response status should be 200
    And I remember the login token as "stale-token"
    And the remembered login token "stale-token" is revoked
    When I request my auth profile using remembered login token "stale-token" and device "11111111-1111-1111-1111-111111111111"
    Then the response status should be 401
