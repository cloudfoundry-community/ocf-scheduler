Feature: Deleting a Call Schedule
  My call is running too often, because it's scheduled several times. I want
  to clean that up a bit.

  Background:
    Given I'm authorized to use the scheduler API
    And there's a call with the GUID call-1
    And that call has a schedule with the GUID schedule-1

  Scenario: Successfully deleting the Call Schedule
    When I DELETE with authentication from /calls/call-1/schedules/schedule-1
    Then the response code is 204
    And the response body is empty

  @failure
  Scenario: User does not provide auth info
    When I DELETE without authentication to /calls/call-1/schedules/schedule-1
    Then the response code is 401
    And the response body is empty

  @failure
  Scenario: User lacks permission to use the scheduler
    Given I'm not authorized to use the scheduler API
    When I DELETE with authentication from /calls/call-1/schedules/schedule-1
    Then the response code is 401
    And the response body is empty

  @failure
  Scenario: No such Call
    When I DELETE with authentication from /calls/1-llac/schedules/schedule-1
    Then the response code is 404
    And the response body is empty

  @failure
  Scenario: No such Schedule
    When I DELETE with authentication from /calls/call-1/schedules/1-eludehcs
    Then the response code is 404
    And the response body is empty
