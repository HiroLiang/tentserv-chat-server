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
