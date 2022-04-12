Feature: Getting all Schedules for a Job
  I want to know all the times at which my configured Job will run, so I
  want to be able to retrieve that list of Schedules.

  Background:
    Given I'm authorized to use the scheduler API
    And there's a job with the GUID job-1
    And that job has several schedules

  Scenario: Successfully getting the Job Schedules list
    When I GET with authentication from /jobs/job-1/schedules
    Then the response code is 200
    And the response body is a JSON object
    And that JSON contains pagination information
    And that JSON contains an array of resources
    And each of those resources is a complete Schedule object
    And all schedules for job-1 are represented in that array

  @failure
  Scenario: User does not provide auth info
    When I GET without authentication to /jobs/job-1/schedules
    Then the response code is 401
    And the response body is empty

  @failure
  Scenario: User lacks permission to use the scheduler
    Given I'm not authorized to use the scheduler API
    When I GET with authentication from /jobs/job-1/schedules
    Then the response code is 401
    And the response body is empty


  @failure
  Scenario: No such Job
    When I GET with authentication from /jobs/1-boj/schedules
    Then the response code is 404
    And the response body is empty
