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
    And self sender key sync snapshot lookups will fail after mutation
    When I accept the self sender key sync
    Then the response status should be 200
    And the self sender key sync response should show status "syncing"

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
