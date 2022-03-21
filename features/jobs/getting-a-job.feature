Feature: Getting a Job
  I know that I have jobs configured. I want to be able to see the details for
  one of them.

  Background:
    Given I'm authorized to use the scheduler API
    And there's an application with the GUID 123
    And the application has a job with the GUID abc

  Scenario: Successfully getting a Job
    When I GET with authentication from /jobs/abc
    Then the response code is 200
    And I receive the requested Job object in the response body

  @failure
  Scenario: Job doesn't exist
    Given there's no job with the GUID def
    When I GET with authentication from /jobs/def
    Then the response code is 404
    And the response body is empty

  @failure
  Scenario: User does not provide auth info
    When I GET without authentication to /jobs/abc
    Then the response code is 401
    And the response body is empty

  @failure
  Scenario: User lacks permission to use the scheduler
    Given I'm not authorized to use the scheduler API
    When I GET with authentication from /jobs/abc
    Then the response code is 401
    And the response body is empty
