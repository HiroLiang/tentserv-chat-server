Feature: Cross-feature backend integration
  Backend HTTP flows should remain consistent when registration, login, E2EE bootstrap, friendship accept, unfriend, and direct room recreation are chained together.

  Scenario: Register verify login bootstrap E2EE accept friend unfriend and recreate a new direct room
    Given account registration state is clean
    And E2EE key bootstrap state is clean
    And a registered login device "11111111-1111-1111-1111-111111111111" named "Cross Test Device" exists
    When I register an account with email "cross@example.com", account "cross_account", display name "Cross User", and password "redacted-password"
    Then the response status should be 201
    And the account registration mutation should include account, user, role, token, and email
    When I verify the registered email
    Then the response status should be 200
    And the verify email response should be an HTML success page
    And the email verification mutation should activate the account and consume the token
    When I login with identifier "cross@example.com", password "redacted-password", and device "11111111-1111-1111-1111-111111111111"
    Then the response status should be 200
    And login response should include a bearer token
    When I request the E2EE key policy using the login token
    Then the response status should be 200
    And the E2EE key policy should be target 20 and threshold 5
    When I upload E2EE identity key "alpha" for device "11111111-1111-1111-1111-111111111111"
    Then the response status should be 200
    And the E2EE identity key response fingerprint should equal SHA-256 of key "alpha"
    When I upload E2EE signed pre-key "alpha" with key id 1 for device "11111111-1111-1111-1111-111111111111"
    Then the response status should be 204
    When I upload 3 E2EE one-time pre-keys starting at key id 1 for device "11111111-1111-1111-1111-111111111111"
    Then the response status should be 200
    When I request E2EE key status for the logged in user and device "11111111-1111-1111-1111-111111111111"
    Then the response status should be 200
    And E2EE key status should expose identity "alpha", signed pre-key "alpha", key id 1, and 3 OTP keys for device "11111111-1111-1111-1111-111111111111"
    And the E2EE key status check should not consume OTP keys
    Given friendship state is clean
    And the logged in user exists in the friendship directory as "Cross User" with account "cross_account" and public id "cross-public"
    And a searchable user "Luna" exists with id 701, account "luna_account", public id "luna-public", and avatar "avatars/luna.png"
    And an inbound pending friend request exists from user 701
    When I accept the inbound friend request from user 701
    Then the response status should be 200
    And a direct chat room should exist between the logged in user and user 701
    And the direct room should have type "direct"
    And exactly 1 direct chat room should exist between the logged in user and user 701
    And both members should have role "owner" in that direct room
    When I create a direct chat room for user 701 named "Luna" through the chat API
    Then the response status should be 200
    And the direct room create response should reuse the existing direct room between the logged in user and user 701
    And exactly 1 direct chat room should exist between the logged in user and user 701
    When I unfriend user 701
    Then the response status should be 200
    And the remove friend response should include the deleted direct room
    And the last direct room should be marked deleted
    And both members should be marked deleted in that direct room
    And exactly 0 direct chat room should exist between the logged in user and user 701
    Given an inbound pending friend request exists from user 701
    When I accept the inbound friend request from user 701
    Then the response status should be 200
    And a direct chat room should exist between the logged in user and user 701
    And the active direct room between the logged in user and user 701 should be different from the previously reused direct room
    And exactly 1 direct chat room should exist between the logged in user and user 701
