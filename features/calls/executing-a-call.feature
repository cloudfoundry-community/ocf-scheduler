Feature: Executing a Call
  I have an awesome Call configured in the scheduler. I want to trigger it
  manually.

  Background:
    Given I'm authorized to use the scheduler API
    And there's a call with the GUID call-1

  Scenario: Successfully executing a Call
    When I POST my empty payload with authentication to /calls/call-1/execute
    Then the response code is 201
    And I receive an Execution object in the response body
    And it has a GUID
    And it has a Call GUID
    And it has a Task GUID
    And it has a start time
    And it has an end time
    And it has a state
    And it has a message

  @failure
  Scenario: User does not provide auth info
    When I POST my empty payload without authentication to /calls/call-1/execute
    Then the response code is 401
    And the response body is empty

  @failure
  Scenario: User lacks permission to use the scheduler
    Given I'm not authorized to use the scheduler API
    When I POST my empty payload with authentication to /calls/call-1/execute
    Then the response code is 401
    And the response body is empty

  @failure
  Scenario: No such Call
    When I POST my empty payload with authentication to /calls/1-llac/execute
    Then the response code is 404
    And the response body is empty
