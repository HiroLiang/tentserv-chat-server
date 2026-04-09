Feature: Email verification
  The browser verifies a registered account email token through the backend HTTP API.

  Scenario: Verify registered email
    Given account registration state is clean
    When I register an account with email "new@example.com", account "new_account", display name "New Display", and password "redacted-password"
    Then the response status should be 201
    And the account registration mutation should include account, user, role, token, and email
    When I verify the registered email
    Then the response status should be 200
    And the verify email response should be an HTML success page
    And the email verification mutation should activate the account and consume the token

  Scenario: Reject reused verification token
    Given account registration state is clean
    When I register an account with email "new@example.com", account "new_account", display name "New Display", and password "redacted-password"
    Then the response status should be 201
    When I verify the registered email
    Then the response status should be 200
    When I verify the same email token again
    Then the response status should be 400
    And the response error code should be "TOKEN_INVALID"
    And the reused email verification token should remain consumed

  Scenario: Reject empty verification token
    Given account registration state is clean
    When I verify email with an empty token
    Then the response status should be 400
    And the response error code should be "TOKEN_INVALID"
    And the email verification mutation should not update an account

  Scenario: Reject invalid verification token
    Given account registration state is clean
    When I verify email with an invalid token
    Then the response status should be 400
    And the response error code should be "TOKEN_INVALID"
    And the email verification mutation should not update an account

  Scenario: Reject expired verification token
    Given account registration state is clean
    When I register an account with email "new@example.com", account "new_account", display name "New Display", and password "redacted-password"
    Then the response status should be 201
    When the verification token expires
    And I verify the registered email
    Then the response status should be 400
    And the response error code should be "TOKEN_INVALID"
    And the expired token verification should not activate the account

  Scenario: Resend verification email to applying account
    Given account registration state is clean
    When I register an account with email "new@example.com", account "new_account", display name "New Display", and password "redacted-password"
    Then the response status should be 201
    When I resend the verification email to "new@example.com"
    Then the response status should be 200
    And the resend email mutation should store a new token and send an email

  Scenario: Reject resend for already-active account
    Given account registration state is clean
    When an account exists with email "active@example.com" and account "active_account"
    And I resend the verification email to "active@example.com"
    Then the response status should be 500
    And the response error code should be "REGISTER_FAILED"

  Scenario: Reject resend for unknown email
    Given account registration state is clean
    When I resend the verification email to "unknown@example.com"
    Then the response status should be 500
    And the response error code should be "REGISTER_FAILED"
