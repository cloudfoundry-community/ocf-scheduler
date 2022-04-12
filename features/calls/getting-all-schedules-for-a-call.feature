Feature: Getting all Schedules for a Call
  I want to know all the times at which my configured Call will run, so I
  want to be able to retrieve that list of Schedules.

  Background:
    Given I'm authorized to use the scheduler API
    And there's a call with the GUID call-1
    And that call has several schedules

  Scenario: Successfully getting the Call Schedules list
    When I GET with authentication from /calls/call-1/schedules
    Then the response code is 200
    And the response body is a JSON object
    And that JSON contains pagination information
    And that JSON contains an array of resources
    And each of those resources is a complete Schedule object
    And all schedules for call-1 are represented in that array

  @failure
  Scenario: User does not provide auth info
    When I GET without authentication to /calls/call-1/schedules
    Then the response code is 401
    And the response body is empty

  @failure
  Scenario: User lacks permission to use the scheduler
    Given I'm not authorized to use the scheduler API
    When I GET with authentication from /calls/call-1/schedules
    Then the response code is 401
    And the response body is empty

  @failure
  Scenario: No such Call
    When I GET with authentication from /calls/1-llac/schedules
    Then the response code is 404
    And the response body is empty
