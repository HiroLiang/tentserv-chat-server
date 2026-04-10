Feature: Chat room summaries
  Chat room summaries should provide enough metadata for clients to render encrypted latest-message previews.

  Scenario: Direct room summary includes the latest message sender member id
    Given login state is clean
    And a registered login device "11111111-1111-4111-8111-111111111111" named "MacBook" exists
    And an "active" account exists for login with email "hiro@example.com", account "hiro_account", and password "correct-password"
    When I login with identifier "hiro@example.com", password "correct-password", and device "11111111-1111-4111-8111-111111111111"
    Then the response status should be 200
    And login response should include a bearer token
    Given chat room summary state is clean
    And the logged in chat user exists as "Hiro"
    And a direct chat peer "Luna" exists with id 701 and avatar "avatars/luna.png"
    And I have a direct chat room with user 701
    And the direct chat room has latest message "e2ee:v1:ciphertext" from user 701
    When I request my chat rooms
    Then the response status should be 200
    And the direct chat rooms response should include "Luna" with avatar "avatars/luna.png", latest message "e2ee:v1:ciphertext", and latest message sender member id from user 701
