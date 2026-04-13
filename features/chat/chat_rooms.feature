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
    And the direct chat rooms response should include "Luna" with avatar "avatars/luna.png", latest message "e2ee:v1:ciphertext", latest message sender member id from user 701, and latest message created_at

  Scenario: Direct room summary does not count the caller's own latest message as unread
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
    And the direct chat room has latest message "visible-from-me" from the logged in user
    When I request my chat rooms
    Then the response status should be 200
    And the direct chat rooms response should include "Luna" with unread count 0

  Scenario: Marking a direct room as read updates caller last_read_at and clears unread summary
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
    And the direct chat room has latest message "peer-message" from user 701
    When I request my chat rooms
    Then the response status should be 200
    And the direct chat rooms response should include "Luna" with unread count 1
    When I mark the last direct room as read
    Then the response status should be 200
    And the member status response should include the logged in member with last_read_at
    When I request my chat rooms
    Then the response status should be 200
    And the direct chat rooms response should include "Luna" with unread count 0

  Scenario: Direct room summary includes online presence for the peer
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
    And the chat presence for user 701 is "online"
    When I request my chat rooms
    Then the response status should be 200
    And the direct chat rooms response should include "Luna" with peer user id 701 and presence "online"

  Scenario: Direct room summary includes offline last seen for the peer
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
    And the chat presence for user 701 is "offline" with last seen "2026-04-12T02:03:04Z"
    When I request my chat rooms
    Then the response status should be 200
    And the direct chat rooms response should include "Luna" with peer user id 701, presence "offline", and last seen "2026-04-12T02:03:04Z"

  Scenario: Deleted direct room is not accessible through chat APIs
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
    And the direct chat room is deleted
    When I request my chat rooms
    Then the response status should be 200
    And the direct chat rooms response should not include "Luna"
    When I request chat room detail for the last direct room
    Then the response status should be 404
    When I request chat room messages for the last direct room
    Then the response status should be 404
    When I send a text message "hello" to the last direct room
    Then the response status should be 404

  Scenario: Direct room remains visible when the peer blocks me but peer messages are filtered
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
    And the direct chat room has latest message "hidden-from-peer" from user 701
    And the direct chat room has latest message "visible-from-me" from the logged in user
    And user 701 has blocked the logged in chat user
    When I request my chat rooms
    Then the response status should be 200
    And the direct chat rooms response should include "Luna" marked blocked by peer with latest message "visible-from-me"
    When I request chat room detail for the last direct room
    Then the response status should be 200
    And the chat room detail response should be marked blocked by peer
    And the chat room response should include message "visible-from-me"
    And the chat room response should not include message "hidden-from-peer"
    When I request chat room messages for the last direct room
    Then the response status should be 200
    And the chat room response should include message "visible-from-me"
    And the chat room response should not include message "hidden-from-peer"
    When I send a text message "blocked send" to the last direct room
    Then the response status should be 403

  Scenario: Direct room remains visible when I block the peer and sending stays disabled
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
    And the direct chat room has latest message "hidden-from-peer" from user 701
    And the direct chat room has latest message "visible-from-me" from the logged in user
    And the logged in chat user has blocked user 701
    When I request my chat rooms
    Then the response status should be 200
    And the direct chat rooms response should include "Luna" marked blocked by me with latest message "visible-from-me"
    When I request chat room detail for the last direct room
    Then the response status should be 200
    And the chat room detail response should be marked blocked by me
    And the chat room response should include message "visible-from-me"
    And the chat room response should not include message "hidden-from-peer"
    When I send a text message "blocked send" to the last direct room
    Then the response status should be 403

  Scenario: Group room messages from blocked users are filtered
    Given login state is clean
    And a registered login device "11111111-1111-4111-8111-111111111111" named "MacBook" exists
    And an "active" account exists for login with email "hiro@example.com", account "hiro_account", and password "correct-password"
    When I login with identifier "hiro@example.com", password "correct-password", and device "11111111-1111-4111-8111-111111111111"
    Then the response status should be 200
    And login response should include a bearer token
    Given chat room summary state is clean
    And the logged in chat user exists as "Hiro"
    And a direct chat peer "Luna" exists with id 701 and avatar "avatars/luna.png"
    And a direct chat peer "Mina" exists with id 702 and avatar "avatars/mina.png"
    And I have a group chat room "Crew" with users 701 and 702
    And user 701 has blocked the logged in chat user
    And the last chat room has latest message "hidden-from-blocked-member" from user 701
    And the last chat room has latest message "visible-from-member" from user 702
    When I request chat room messages for the last chat room
    Then the response status should be 200
    And the chat room response should include message "visible-from-member"
    And the chat room response should not include message "hidden-from-blocked-member"
