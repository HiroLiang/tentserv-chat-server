Feature: Account registration
  The Sign Up Page submits email, account, display name, and password to create an account.

  Scenario: Successful account registration
    Given account registration state is clean
    When I register an account with email "new@example.com", account "new_account", display name "New Display", and password "redacted-password"
    Then the response status should be 201
    And the account registration mutation should include account, user, role, token, and email

  Scenario: Reject invalid email
    Given account registration state is clean
    When I register an account with email "not-an-email", account "new_account", display name "New Display", and password "redacted-password"
    Then the response status should be 400
    And the response error code should be "INVALID_REQUEST"
    And the account registration mutation should stop before account creation

  Scenario: Reject duplicate email
    Given an account exists with email "taken@example.com" and account "existing_account"
    When I register an account with email "taken@example.com", account "new_account", display name "New Display", and password "redacted-password"
    Then the response status should be 409
    And the response error code should be "EMAIL_EXIST"
    And the account registration mutation should stop before account creation

  Scenario: Reject duplicate account
    Given an account exists with email "existing@example.com" and account "taken_account"
    When I register an account with email "new@example.com", account "taken_account", display name "New Display", and password "redacted-password"
    Then the response status should be 409
    And the response error code should be "ACCOUNT_EXIST"
    And the account registration mutation should stop before account creation

  Scenario: Reject short password
    Given account registration state is clean
    When I register an account with email "new@example.com", account "new_account", display name "New Display", and password "abc"
    Then the response status should be 400
    And the response error code should be "INVALID_REQUEST"
    And the account registration mutation should stop before account creation

  Scenario: Reject common password
    Given account registration state is clean
    When I register an account with email "new@example.com", account "new_account", display name "New Display", and password "password"
    Then the response status should be 400
    And the response error code should be "WEAK_PASSWORD"
    And the account registration mutation should stop before account creation

  Scenario: Reject invalid account name format
    Given account registration state is clean
    When I register an account with email "new@example.com", account "bad account!", display name "New Display", and password "redacted-password"
    Then the response status should be 400
    And the response error code should be "INVALID_ACCOUNT"
    And the account registration mutation should stop before account creation

  Scenario: Reject account name exceeding maximum length
    Given account registration state is clean
    When I register an account with email "new@example.com", account "aaaaaaaaaabbbbbbbbbbccccccccccddddddddddeeeeeeeeeefffff", display name "New Display", and password "redacted-password"
    Then the response status should be 400
    And the response error code should be "INVALID_REQUEST"
    And the account registration mutation should stop before account creation

  Scenario: Reject display name exceeding maximum length
    Given account registration state is clean
    When I register an account with email "new@example.com", account "new_account", display name "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", and password "redacted-password"
    Then the response status should be 400
    And the response error code should be "INVALID_REQUEST"
    And the account registration mutation should stop before account creation

  Scenario: Reject email exceeding maximum length
    Given account registration state is clean
    When I register an account with email "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa@example.com", account "new_account", display name "New Display", and password "redacted-password"
    Then the response status should be 400
    And the response error code should be "INVALID_REQUEST"
    And the account registration mutation should stop before account creation

  Scenario: Reject localhost domain email at binding layer
    Given account registration state is clean
    When I register an account with email "user@localhost", account "new_account", display name "New Display", and password "redacted-password"
    Then the response status should be 400
    And the response error code should be "INVALID_REQUEST"
    And the account registration mutation should stop before account creation

  Scenario: Reject registration when rate limit is exceeded
    Given account registration state is clean
    And the registration rate limit is exceeded
    When I register an account with email "new@example.com", account "new_account", display name "New Display", and password "redacted-password"
    Then the response status should be 429
    And the response error code should be "RATE_LIMIT_EXCEEDED"
    And the account registration mutation should stop before account creation
