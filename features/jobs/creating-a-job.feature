Feature: Creating a Job
  So as to have a task in place for my application that can be executed at
  my whim or at random times, I want to be able to store that in the scheduler
  as a Job.

  Background:
    Given I'm authorized to use the scheduler API
    And there's an application with the GUID 123
    And there are no Jobs for that application
    And my JSON payload looks like this:
    """
    {
      "name" : "sausages",
      "command" : "gold"
    }
    """

  Scenario: Successfully creating a Job
    When I POST my payload with authentication to /jobs?app_guid=123
    Then the response code is 201
    And I receive a Job object in the response body
    And the Job has a GUID
    And the Job is named "sausages"
    And the Job's command is "gold"
    And the Job can use up to 1024MB of memory
    And the Job can use up to 1024MB of disk space
    And the Job has an app GUID
    And the Job has a space GUID
    And the Job has a state
    And the Job is timestamped

  Scenario: Specifying job disk quota
    Given my payload has a disk_in_mb int of 12
    When I POST my payload with authentication to /jobs?app_guid=123
    Then the response code is 201
    And I receive a Job object in the response body
    And the Job can use up to 12MB of disk space

  Scenario: Specifying job memory quota
    Given my payload has a memory_in_mb int of 13
    When I POST my payload with authentication to /jobs?app_guid=123
    Then the response code is 201
    And I receive a Job object in the response body
    And the Job can use up to 13MB of memory

  @failure
  Scenario: User does not provide auth info
    When I POST my payload without authentication to /jobs?app_guid=123
    Then the response code is 401
    And the response body is empty


  @failure
  Scenario: User lacks permission to use the scheduler
    Given I'm not authorized to use the scheduler API
    When I POST my payload with authentication to /jobs?app_guid=123
    Then the response code is 401
    And the response body is empty

  @failure
  Scenario: Name collision
    Given there's already a sausages app for my application
    When I POST my payload with authentication to /jobs?app_guid=123
    Then the response code is 422
    And the response body is empty

  @failure
  Scenario: No job name provided
    Given the name field is missing from my payload
    When I POST my payload with authentication to /jobs?app_guid=123
    Then the response code is 422
    And the response body is empty

  @failure
  Scenario: No job command provided
    Given the command field is missing from my payload
    When I POST my payload with authentication to /jobs?app_guid=123
    Then the response code is 422
    And the response body is empty

  @failure
  Scenario: App GUID not provdided
    When I POST my payload with authentication to /jobs
    Then the response code is 422
    And the response body is empty


