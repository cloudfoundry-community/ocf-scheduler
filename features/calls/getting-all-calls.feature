Feature: Getting all Calls
  I know that I have calls configured. I want to be able to see the details for
  all of them. This requires that I pass the space GUID for the Calls that I
  want to know about, even though the system can only operate within one
  space.

  Background:
    Given I'm authorized to use the scheduler API
    And I have several apps, each with several Calls

  Scenario: Successfully getting the Calls list
    When I GET with authentication from /calls?space_guid=abcdef-1
    Then the response code is 200
    And the response body is a JSON object
    And that JSON contains pagination information
    And that JSON contains an array of resources
    And each of those resources is a complete Call object
    And all calls in the space are represented in that array

  @failure
  Scenario: User does not provide auth info
    When I GET without authentication to /calls?space_guid=abcdef-1
    Then the response code is 401
    And the response body is empty

  @failure
  Scenario: User lacks permission to use the scheduler
    Given I'm not authorized to use the scheduler API
    When I GET with authentication from /calls?space_guid=abcdef-1
    Then the response code is 401
    And the response body is empty
