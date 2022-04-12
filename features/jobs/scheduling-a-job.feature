Feature: Scheduling a Job
  I have an awesome Job configured in the scheduler. Right now, I can only
  manually trigger it for execution, which is great, but not GREAT. So, I'd
  like to be able to schedule it to run regularly.

  Background:
    Given I'm authorized to use the scheduler API
    And there's a job with the GUID job-1
    And my JSON payload looks like this:
    """
    {
      "enabled" : true,
      "expression" : "*/5 * ? * *",
      "expression_type" : "cron_expression"
    }
    """

  Scenario: Successfully scheduling a Job
    When I POST my payload with authentication to /jobs/job-1/schedules
    Then the response code is 201
    And I receive a Schedule object in the response body
    And it has a GUID
    And it has a Job GUID
    And it has timestamps
    And it is enabled
    And its expression is "*/5 * ? * *"
    And the expression is a cron expression

  @failure
  Scenario: User does not provide auth info
    When I POST my payload without authentication to /jobs/job-1/schedules
    Then the response code is 401
    And the response body is empty

  @failure
  Scenario: User lacks permission to use the scheduler
    Given I'm not authorized to use the scheduler API
    When I POST my payload with authentication to /jobs/job-1/schedules
    Then the response code is 401
    And the response body is empty

  @failure
  Scenario: No such Job
    When I POST my payload with authentication to /jobs/1-boj/schedules
    Then the response code is 404
    And the response body is empty
