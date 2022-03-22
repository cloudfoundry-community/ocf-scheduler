Feature: Getting the execution history for a Job
  I've been executing my Job all over the place. Show me how it went.

  Background:
    Given I'm authorized to use the scheduler API
    And there's a job with the GUID job-1
    And I've executed that Job several times

  Scenario: Successfully getting the history for a job
    When I GET with authentication from /jobs/job-1/history
    Then the response code is 200
    And the response body is a JSON object
    And that JSON contains pagination information
    And that JSON contains an array of resources
    And each of those resources is a complete Job Execution object
    And all manual executions for the Job are represented in that array

  @failure
  Scenario: User does not provide auth info
    When I GET with authentication from /jobs/job-1/history
    Then the response code is 401
    And the response body is empty

  @failure
  Scenario: User lacks permission to use the scheduler
    Given I'm not authorized to use the scheduler API
    When I GET with authentication from /jobs/job-1/history
    Then the response code is 401
    And the response body is empty

  @failure
  Scenario: No such Job
    When I GET with authentication from /jobs/1-bohj/history
    Then the response code is 404
    And the response body is empty
