Feature: Cross-feature backend integration
  Backend HTTP flows should remain consistent when registration, login, E2EE bootstrap, friendship accept, unfriend, and direct room recreation are chained together.

  Scenario: Register verify login bootstrap E2EE accept friend unfriend and recreate a new direct room
    Given account registration state is clean
    And E2EE key bootstrap state is clean
    And a registered login device "11111111-1111-1111-1111-111111111111" named "Cross Test Device" exists
    When I register an account with email "cross@example.com", account "cross_account", display name "Cross User", and password "redacted-password"
    Then the response status should be 201
    And the register response should include a verification token and expiry timestamp
    And the account registration mutation should include account, user, role, token, and email
    When I verify the registered email
    Then the response status should be 200
    And the verify email response should be an empty JSON object
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

  Scenario: Register verify first device login and verify a second device into pending self sync
    Given account registration state is clean
    And E2EE key bootstrap state is clean
    And a registered login device "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa" named "Cross Test Desktop" exists
    And a registered login device "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb" named "Cross Test Phone" exists
    When I register an account with email "cross-sync@example.com", account "cross_sync_account", display name "Cross Sync User", and password "redacted-password"
    Then the response status should be 201
    And the register response should include a verification token and expiry timestamp
    When I verify the registered email
    Then the response status should be 200
    And the verify email response should be an empty JSON object
    And the email verification mutation should activate the account and consume the token
    When I login with identifier "cross-sync@example.com", password "redacted-password", and device "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
    Then the response status should be 200
    And login response should include a bearer token
    And the login account is already bound to device "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa" with status "ready"
    When I login with identifier "cross-sync@example.com", password "redacted-password", and device "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
    Then the response status should be 202
    And login response should require device verification and include a verification token and expiry timestamp
    When I verify the login device using the stored token
    Then the response status should be 200
    And login response should include a bearer token
    And verifying the login device should create a pending sync device binding and self sender key sync

  Scenario: Register verify first device login and complete second-device self sync with distinct device ids
    Given account registration state is clean
    And E2EE key bootstrap state is clean
    And a registered login device "cccccccc-cccc-4ccc-8ccc-cccccccccccc" named "Cross Test Desktop" exists
    And a registered login device "dddddddd-dddd-4ddd-8ddd-dddddddddddd" named "Cross Test Phone" exists
    When I register an account with email "cross-sync-complete@example.com", account "cross_sync_complete_account", display name "Cross Sync Complete", and password "redacted-password"
    Then the response status should be 201
    And the register response should include a verification token and expiry timestamp
    When I verify the registered email
    Then the response status should be 200
    And the verify email response should be an empty JSON object
    When I login with identifier "cross-sync-complete@example.com", password "redacted-password", and device "cccccccc-cccc-4ccc-8ccc-cccccccccccc"
    Then the response status should be 200
    And login response should include a bearer token
    And the login account is already bound to device "cccccccc-cccc-4ccc-8ccc-cccccccccccc" with status "ready"
    When I login with identifier "cross-sync-complete@example.com", password "redacted-password", and device "dddddddd-dddd-4ddd-8ddd-dddddddddddd"
    Then the response status should be 202
    And login response should require device verification and include a verification token and expiry timestamp
    When I verify the login device using the stored token
    Then the response status should be 200
    And login response should include a bearer token
    And verifying the login device should create a pending sync device binding and self sender key sync
    And the current logged in user is available to sender key use cases
    When I login with identifier "cross-sync-complete@example.com", password "redacted-password", and device "cccccccc-cccc-4ccc-8ccc-cccccccccccc"
    Then the response status should be 200
    And login response should include a bearer token
    When I accept the self sender key sync as device "cccccccc-cccc-4ccc-8ccc-cccccccccccc"
    Then the response status should be 200
    And the self sender key sync response should show status "syncing"
    And the self sender key sync requester device "dddddddd-dddd-4ddd-8ddd-dddddddddddd" should now be bound as "syncing"
    And a self sender key sync distribution exists for the current logged in participant with room id 1001, sender member id 10011, requester device "dddddddd-dddd-4ddd-8ddd-dddddddddddd", provider device "cccccccc-cccc-4ccc-8ccc-cccccccccccc", and sender key version 21
    And a self sender key sync distribution exists for the current logged in participant with room id 1002, sender member id 10021, requester device "dddddddd-dddd-4ddd-8ddd-dddddddddddd", provider device "cccccccc-cccc-4ccc-8ccc-cccccccccccc", and sender key version 22
    When I mark the self sender key sync uploaded as device "cccccccc-cccc-4ccc-8ccc-cccccccccccc"
    Then the response status should be 200
    And the self sender key sync response should show status "uploaded"
    When I login with identifier "cross-sync-complete@example.com", password "redacted-password", and device "dddddddd-dddd-4ddd-8ddd-dddddddddddd"
    Then the response status should be 200
    And login response should include a bearer token
    When I list pending self sender key sync distributions
    Then the response status should be 200
    And pending self sender key sync distributions should include sender member 10011 device "cccccccc-cccc-4ccc-8ccc-cccccccccccc" and version 21
    When I mark the first pending self sender key sync distribution as "consumed"
    Then the response status should be 204
    And the first pending self sender key sync distribution should now be "consumed"
    When I list pending self sender key sync distributions
    Then the response status should be 200
    And pending self sender key sync distributions should include sender member 10021 device "cccccccc-cccc-4ccc-8ccc-cccccccccccc" and version 22
    When I mark the first pending self sender key sync distribution as "consumed"
    Then the response status should be 204
    And the first pending self sender key sync distribution should now be "consumed"
    When I complete the self sender key sync
    Then the response status should be 200
    And the self sender key sync response should show status "completed"
    And the self sender key sync requester device "dddddddd-dddd-4ddd-8ddd-dddddddddddd" should now be bound as "ready"
