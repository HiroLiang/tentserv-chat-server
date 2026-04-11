Feature: Email verification
  The app verifies registration by posting a verification token and 6-digit code to the backend.

  Scenario: Verify registered email with the correct code
    Given account registration state is clean
    When I register an account with email "new@example.com", account "new_account", display name "New Display", and password "redacted-password"
    Then the response status should be 201
    And the register response should include a verification token and expiry timestamp
    And the account registration mutation should include account, user, role, token, and email
    When I verify the registered email with the correct code
    Then the response status should be 200
    And the verify email response should be an empty JSON object
    And the email verification mutation should activate the account and consume the token

  Scenario: Reject reused verification token after success
    Given account registration state is clean
    When I register an account with email "new@example.com", account "new_account", display name "New Display", and password "redacted-password"
    Then the response status should be 201
    When I verify the registered email with the correct code
    Then the response status should be 200
    When I verify the same email token again
    Then the response status should be 400
    And the response error code should be "TOKEN_INVALID"
    And the reused email verification token should remain consumed

  Scenario: Reject invalid verification code and return remaining attempts
    Given account registration state is clean
    When I register an account with email "new@example.com", account "new_account", display name "New Display", and password "redacted-password"
    Then the response status should be 201
    When I verify the registered email with code "000000"
    Then the response status should be 400
    And the response error code should be "VERIFY_CODE_INVALID"
    And the response remaining attempts should be 2

  Scenario: Invalidate the verification session after three failed attempts
    Given account registration state is clean
    When I register an account with email "new@example.com", account "new_account", display name "New Display", and password "redacted-password"
    Then the response status should be 201
    When I verify the registered email with code "000000"
    Then the response status should be 400
    And the response error code should be "VERIFY_CODE_INVALID"
    And the response remaining attempts should be 2
    When I verify the registered email with code "000000"
    Then the response status should be 400
    And the response error code should be "VERIFY_CODE_INVALID"
    And the response remaining attempts should be 1
    When I verify the registered email with code "000000"
    Then the response status should be 400
    And the response error code should be "VERIFY_ATTEMPTS_EXCEEDED"
    And the response remaining attempts should be 0
    And the failed verification attempts should invalidate the session without activating the account

  Scenario: Reject expired verification token
    Given account registration state is clean
    When I register an account with email "new@example.com", account "new_account", display name "New Display", and password "redacted-password"
    Then the response status should be 201
    When the verification token expires
    And I verify the registered email with the correct code
    Then the response status should be 400
    And the response error code should be "TOKEN_INVALID"
    And the expired token verification should not activate the account

  Scenario: Resend verification email using a valid token
    Given account registration state is clean
    When I register an account with email "new@example.com", account "new_account", display name "New Display", and password "redacted-password"
    Then the response status should be 201
    And the register response should include a verification token and expiry timestamp
    When I resend the verification email using the registered token
    Then the response status should be 200
    And the resend response should include a new verification token and expiry timestamp
    And the resend email mutation should store a new token and send an email

  Scenario: Reject resend for an invalid token
    Given account registration state is clean
    When I resend verification email with an invalid token
    Then the response status should be 400
    And the response error code should be "TOKEN_INVALID"

  Scenario: Reject resend for an expired registered token
    Given account registration state is clean
    When I register an account with email "new@example.com", account "new_account", display name "New Display", and password "redacted-password"
    Then the response status should be 201
    When the verification token expires
    And I resend verification email with the expired registered token
    Then the response status should be 400
    And the response error code should be "TOKEN_INVALID"
