Feature: Getting the execution history for a Job Schedule
  My scheduled Job has been executing all over the place. Show me how it went.

  Background:
    Given I'm authorized to use the scheduler API
    And there's a job with the GUID job-1
    And that job has a schedule with the GUID schedule-1

  Scenario: Successfully getting the history for a job schedule
    When I GET with authentication from /jobs/job-1/schedules/schedule-1/history
    Then the response code is 200
    And the response body is a JSON object
    And that JSON contains pagination information
    And that JSON contains an array of resources
    And each of those resources is a complete Job Execution object
    And each of those Executions has a Schedule GUID and time
    And all executions for the Schedule are represented in that array

  @failure
  Scenario: User does not provide auth info
    When I GET with authentication from /jobs/job-1/schedules/schedule-1/history
    Then the response code is 401
    And the response body is empty

  @failure
  Scenario: User lacks permission to use the scheduler
    Given I'm not authorized to use the scheduler API
    When I GET with authentication from /jobs/job-1/schedules/schedule-1/history
    Then the response code is 401
    And the response body is empty

  @failure
  Scenario: No such Job
    When I GET with authentication from /jobs/1-bohj/schedules/schedule-1/history
    Then the response code is 404
    And the response body is empty

  @failure
  Scenario: No such Schedule
    When I GET with authentication from /jobs/job-1/schedules/1-eludehcs/history
    Then the response code is 404
    And the response body is empty
