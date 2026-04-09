Feature: Friendship search and requests
  Background:
    Given login state is clean
    And a registered login device "11111111-1111-1111-1111-111111111111" named "Friend Test Device" exists
    And an "active" account exists for login with email "hiro@example.com", account "hiro_account", and password "correct-password"
    When I login with identifier "hiro_account", password "correct-password", and device "11111111-1111-1111-1111-111111111111"
    Then the response status should be 200
    And login response should include a bearer token
    Given friendship state is clean
    And the logged in user exists in the friendship directory as "Hiro" with account "hiro_account" and public id "hiro-public"

  Scenario: Search users returns avatar public id and excludes current user
    Given a searchable user "Mina" exists with id 601, account "mina_account", public id "mina-public", and avatar "avatars/mina.png"
    When I search friends by name "Mi"
    Then the response status should be 200
    And search results should include "Mina" with public id "mina-public", avatar "avatars/mina.png", and no friendship status
    And search results should not include the logged in user

  Scenario: Search users returns pending and accepted friendship statuses
    Given a searchable user "Pending Friend" exists with id 602, account "pending_account", public id "pending-public", and avatar "avatars/pending.png"
    And I have sent a pending friend request to user 602
    And a searchable user "Accepted Friend" exists with id 603, account "accepted_account", public id "accepted-public", and avatar "avatars/accepted.png"
    And I am accepted friends with user 603
    When I search friends by name "Friend"
    Then the response status should be 200
    And search results should include "Pending Friend" with public id "pending-public", avatar "avatars/pending.png", and friendship status "pending"
    And search results should include "Accepted Friend" with public id "accepted-public", avatar "avatars/accepted.png", and friendship status "accepted"

  Scenario: Apply creates a pending friendship and friends tab returns sent requests
    Given a searchable user "Nova" exists with id 604, account "nova_account", public id "nova-public", and avatar "avatars/nova.png"
    When I apply to user 604
    Then the response status should be 200
    And friendship rows should contain "pending" from the logged in user to user 604
    When I request my friends
    Then the response status should be 200
    And the friends response should include "Nova" with status "pending"

  Scenario: Apply returns conflict for duplicate or reverse pending relationships
    Given a searchable user "Rhea" exists with id 605, account "rhea_account", public id "rhea-public", and avatar "avatars/rhea.png"
    And I have sent a pending friend request to user 605
    When I apply to user 605
    Then the response status should be 409
    And the response error code should be "FRIENDSHIP_EXISTS"
    Given friendship state is clean
    And the logged in user exists in the friendship directory as "Hiro" with account "hiro_account" and public id "hiro-public"
    And a searchable user "Rhea" exists with id 605, account "rhea_account", public id "rhea-public", and avatar "avatars/rhea.png"
    And an inbound pending friend request exists from user 605
    When I apply to user 605
    Then the response status should be 409
    And the response error code should be "FRIENDSHIP_EXISTS"

  Scenario: Apply rejects self targets and invalid payloads
    When I apply to the logged in user
    Then the response status should be 400
    And the response error code should be "INVALID_FRIENDSHIP_TARGET"
    When I apply with an invalid friend payload
    Then the response status should be 400
    And the response error code should be "INVALID_REQUEST"

  Scenario: Requests tab lists inbound pending requests and accept creates mutual accepted rows
    Given a searchable user "Luna" exists with id 606, account "luna_account", public id "luna-public", and avatar "avatars/luna.png"
    And an inbound pending friend request exists from user 606
    When I request incoming friend requests
    Then the response status should be 200
    And the incoming requests response should include "Luna"
    When I accept the inbound friend request from user 606
    Then the response status should be 200
    And mutual friendship rows between the logged in user and user 606 should both be "accepted"
    When I request incoming friend requests
    Then the response status should be 200
    And the incoming requests response should not include user 606
    When I request my friends
    Then the response status should be 200
    And the friends response should include "Luna" with status "accepted"

  Scenario: Accept creates a direct room with owner role for both participants
    Given a searchable user "Yuki" exists with id 607, account "yuki_account", public id "yuki-public", and avatar "avatars/yuki.png"
    And an inbound pending friend request exists from user 607
    When I accept the inbound friend request from user 607
    Then the response status should be 200
    And a direct chat room should exist between the logged in user and user 607
    And both members should have role "owner" in that direct room

  Scenario: Reject removes an inbound pending friend request
    Given a searchable user "Iris" exists with id 608, account "iris_account", public id "iris-public", and avatar "avatars/iris.png"
    And an inbound pending friend request exists from user 608
    When I request incoming friend requests
    Then the response status should be 200
    And the incoming requests response should include "Iris"
    When I reject the inbound friend request from user 608
    Then the response status should be 200
    And friendship rows between the logged in user and user 608 should not exist
    When I request incoming friend requests
    Then the response status should be 200
    And the incoming requests response should not include user 608

  Scenario: Cancel removes a sent pending friend request
    Given a searchable user "Kai" exists with id 609, account "kai_account", public id "kai-public", and avatar "avatars/kai.png"
    And I have sent a pending friend request to user 609
    When I request sent friend requests
    Then the response status should be 200
    And the sent requests response should include "Kai"
    When I cancel the sent friend request to user 609
    Then the response status should be 200
    And friendship rows between the logged in user and user 609 should not exist
    When I request sent friend requests
    Then the response status should be 200
    And the sent requests response should not include user 609

  Scenario: Unfriend removes both accepted friendship rows
    Given a searchable user "Aki" exists with id 610, account "aki_account", public id "aki-public", and avatar "avatars/aki.png"
    And I am accepted friends with user 610
    When I request my friends
    Then the response status should be 200
    And the friends response should include "Aki" with status "accepted"
    When I unfriend user 610
    Then the response status should be 200
    And friendship rows between the logged in user and user 610 should not exist
    When I request my friends
    Then the response status should be 200
    And the friends response should not include user 610

  Scenario: Block and unblock move a relationship into the blocked list
    Given a searchable user "Sora" exists with id 611, account "sora_account", public id "sora-public", and avatar "avatars/sora.png"
    And I am accepted friends with user 611
    When I block user 611
    Then the response status should be 200
    And friendship row from user 611 to the logged in user should not exist
    When I request blocked users
    Then the response status should be 200
    And the blocked response should include "Sora" with status "blocked"
    When I search friends by name "Sora"
    Then the response status should be 200
    And search results should include "Sora" with public id "sora-public", avatar "avatars/sora.png", and friendship status "blocked"
    When I unblock user 611
    Then the response status should be 200
    When I request blocked users
    Then the response status should be 200
    And the blocked response should not include user 611

  Scenario: Accept returns not found for an unknown friendship id
    When I accept friendship id 9999
    Then the response status should be 404
    And the response error code should be "NOT_FOUND"

  Scenario: Accept rejects an already accepted friendship
    Given a searchable user "Mio" exists with id 612, account "mio_account", public id "mio-public", and avatar "avatars/mio.png"
    And an inbound pending friend request exists from user 612
    When I accept the inbound friend request from user 612
    Then the response status should be 200
    When I accept the inbound friend request from user 612
    Then the response status should be 400
    And the response error code should be "FRIENDSHIP_NOT_PENDING"
