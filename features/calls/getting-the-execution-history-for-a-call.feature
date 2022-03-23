Feature: Getting the execution history for a Call
  I've been executing my Call all over the place. Show me how it went.

  Background:
    Given I'm authorized to use the scheduler API
    And there's a call with the GUID call-1
    And I've executed that Call several times

  Scenario: Successfully getting the history for a call
    When I GET with authentication from /calls/call-1/history
    Then the response code is 200
    And the response body is a JSON object
    And that JSON contains pagination information
    And that JSON contains an array of resources
    And each of those resources is a complete Call Execution object
    And all manual executions for the Call are represented in that array

  @failure
  Scenario: User does not provide auth info
    When I GET with authentication from /calls/call-1/history
    Then the response code is 401
    And the response body is empty

  @failure
  Scenario: User lacks permission to use the scheduler
    Given I'm not authorized to use the scheduler API
    When I GET with authentication from /calls/call-1/history
    Then the response code is 401
    And the response body is empty

  @failure
  Scenario: No such Call
    When I GET with authentication from /calls/1-llac/history
    Then the response code is 404
    And the response body is empty
