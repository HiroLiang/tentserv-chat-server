Feature: Self sender key sync mutations
  Self sender key sync endpoints should return the updated state when the mutation succeeds, even if the follow-up snapshot lookup fails.

  Scenario: Provider accept returns syncing snapshot even when authoritative lookup fails after mutation
    Given login state is clean
    And a registered login device "22222222-2222-4222-8222-222222222222" named "MacBook" exists
    And an "active" account exists for login with email "hiro@example.com", account "hiro_account", and password "correct-password"
    When I login with identifier "hiro@example.com", password "correct-password", and device "22222222-2222-4222-8222-222222222222"
    Then the response status should be 200
    And login response should include a bearer token
    Given E2EE key bootstrap state is clean
    And a self sender key sync exists for the logged in user with participant id 301, requester device "11111111-1111-4111-8111-111111111111", and status "pending_provider"
    And the self sender key sync requester device "11111111-1111-4111-8111-111111111111" is bound as "pending_sync"
    And self sender key sync snapshot lookups will fail after mutation
    When I accept the self sender key sync
    Then the response status should be 200
    And the self sender key sync response should show status "syncing"
    And the self sender key sync requester device "11111111-1111-4111-8111-111111111111" should now be bound as "syncing"

  Scenario: Requester complete returns completed snapshot even when authoritative lookup fails after mutation
    Given login state is clean
    And a registered login device "33333333-3333-4333-8333-333333333333" named "MacBook" exists
    And an "active" account exists for login with email "hiro@example.com", account "hiro_account", and password "correct-password"
    When I login with identifier "hiro@example.com", password "correct-password", and device "33333333-3333-4333-8333-333333333333"
    Then the response status should be 200
    And login response should include a bearer token
    Given E2EE key bootstrap state is clean
    And a self sender key sync exists for the logged in user with participant id 302, requester device "33333333-3333-4333-8333-333333333333", provider device "44444444-4444-4444-8444-444444444444", and status "uploaded"
    And self sender key sync snapshot lookups will fail after mutation
    When I complete the self sender key sync
    Then the response status should be 200
    And the self sender key sync response should show status "completed"

  Scenario: Requester fail returns pending provider snapshot even when authoritative lookup fails after mutation
    Given login state is clean
    And a registered login device "55555555-5555-4555-8555-555555555555" named "MacBook" exists
    And an "active" account exists for login with email "hiro@example.com", account "hiro_account", and password "correct-password"
    When I login with identifier "hiro@example.com", password "correct-password", and device "55555555-5555-4555-8555-555555555555"
    Then the response status should be 200
    And login response should include a bearer token
    Given E2EE key bootstrap state is clean
    And a self sender key sync exists for the logged in user with participant id 303, requester device "55555555-5555-4555-8555-555555555555", provider device "66666666-6666-4666-8666-666666666666", and status "syncing"
    And self sender key sync snapshot lookups will fail after mutation
    When I fail the self sender key sync with last error "provider upload timed out" and retryable true
    Then the response status should be 200
    And the self sender key sync response should show status "pending_provider"

  Scenario: Requester device still cannot accept its own self sender key sync
    Given login state is clean
    And a registered login device "77777777-7777-4777-8777-777777777777" named "MacBook" exists
    And an "active" account exists for login with email "hiro@example.com", account "hiro_account", and password "correct-password"
    When I login with identifier "hiro@example.com", password "correct-password", and device "77777777-7777-4777-8777-777777777777"
    Then the response status should be 200
    And login response should include a bearer token
    Given E2EE key bootstrap state is clean
    And a self sender key sync exists for the logged in user with participant id 304, requester device "77777777-7777-4777-8777-777777777777", and status "pending_provider"
    When I accept the self sender key sync
    Then the response status should be 403

  Scenario: Requester consumes self sync copies, records receipts, and completes with device ready
    Given login state is clean
    And a registered login device "88888888-8888-4888-8888-888888888888" named "MacBook" exists
    And an "active" account exists for login with email "hiro@example.com", account "hiro_account", and password "correct-password"
    When I login with identifier "hiro@example.com", password "correct-password", and device "88888888-8888-4888-8888-888888888888"
    Then the response status should be 200
    And login response should include a bearer token
    Given E2EE key bootstrap state is clean
    And a self sender key sync exists for the logged in user with participant id 305, requester device "88888888-8888-4888-8888-888888888888", provider device "99999999-9999-4999-8999-999999999999", and status "uploaded"
    And the self sender key sync requester device "88888888-8888-4888-8888-888888888888" is bound as "pending_sync"
    And the self sender key sync provider device "99999999-9999-4999-8999-999999999999" is bound as "ready"
    And a self sender key sync distribution exists for the logged in user with participant id 305, room id 901, sender member id 9011, requester device "88888888-8888-4888-8888-888888888888", provider device "99999999-9999-4999-8999-999999999999", and sender key version 7
    When I list pending self sender key sync distributions
    Then the response status should be 200
    And pending self sender key sync distributions should include sender member 9011 device "99999999-9999-4999-8999-999999999999" and version 7
    When I mark the first pending self sender key sync distribution as "consumed"
    Then the response status should be 204
    And the first pending self sender key sync distribution should now be "consumed"
    And a self sender key receipt should exist from sender member 9011 device "99999999-9999-4999-8999-999999999999" to requester device "88888888-8888-4888-8888-888888888888" with version 7 and source "self_sync"
    When I complete the self sender key sync
    Then the response status should be 200
    And the self sender key sync response should show status "completed"
    And the self sender key sync requester device "88888888-8888-4888-8888-888888888888" should now be bound as "ready"

  Scenario: Mixed consumed and failed self sync copies must retry instead of staying uploaded forever
    Given login state is clean
    And a registered login device "12121212-1212-4121-8121-121212121212" named "MacBook" exists
    And an "active" account exists for login with email "hiro@example.com", account "hiro_account", and password "correct-password"
    When I login with identifier "hiro@example.com", password "correct-password", and device "12121212-1212-4121-8121-121212121212"
    Then the response status should be 200
    And login response should include a bearer token
    Given E2EE key bootstrap state is clean
    And a self sender key sync exists for the logged in user with participant id 306, requester device "12121212-1212-4121-8121-121212121212", provider device "34343434-3434-4343-8343-343434343434", and status "uploaded"
    And the self sender key sync requester device "12121212-1212-4121-8121-121212121212" is bound as "pending_sync"
    And the self sender key sync provider device "34343434-3434-4343-8343-343434343434" is bound as "ready"
    And a self sender key sync distribution exists for the logged in user with participant id 306, room id 902, sender member id 9021, requester device "12121212-1212-4121-8121-121212121212", provider device "34343434-3434-4343-8343-343434343434", and sender key version 11
    And a self sender key sync distribution exists for the logged in user with participant id 306, room id 903, sender member id 9031, requester device "12121212-1212-4121-8121-121212121212", provider device "34343434-3434-4343-8343-343434343434", and sender key version 12
    When I list pending self sender key sync distributions
    Then the response status should be 200
    And pending self sender key sync distributions should include sender member 9021 device "34343434-3434-4343-8343-343434343434" and version 11
    When I mark the first pending self sender key sync distribution as "consumed"
    Then the response status should be 204
    And the first pending self sender key sync distribution should now be "consumed"
    And a self sender key receipt should exist from sender member 9021 device "34343434-3434-4343-8343-343434343434" to requester device "12121212-1212-4121-8121-121212121212" with version 11 and source "self_sync"
    When I list pending self sender key sync distributions
    Then the response status should be 200
    And pending self sender key sync distributions should include sender member 9031 device "34343434-3434-4343-8343-343434343434" and version 12
    When I mark the first pending self sender key sync distribution as "failed"
    Then the response status should be 204
    And the first pending self sender key sync distribution should now be "failed"
    When I complete the self sender key sync
    Then the response status should be 409
    When I fail the self sender key sync with last error "self sender key sync distribution failed distribution_id=2 sender_member_id=9031 sender_device_id=34343434-3434-4343-8343-343434343434 sender_key_version=12 copy_scope=peer_history" and retryable true
    Then the response status should be 200
    And the self sender key sync response should show status "pending_provider"
    And the self sender key sync requester device "12121212-1212-4121-8121-121212121212" should now be bound as "pending_sync"
