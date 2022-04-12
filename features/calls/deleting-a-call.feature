Feature: Deleting a Call
  If I have a call that I need to delete, I want to be able to do that.

  Background:
    Given I'm authorized to use the scheduler API
    And there's an application with the GUID 123
    And the application has a call with the GUID abc

  Scenario: Successfully Delete a Call
    When I DELETE with authentication from /calls/abc
    Then the response code is 204
    And the response body is empty

  @failure
  Scenario: Call doesn't exist
    Given there's no call with the GUID def
    When I DELETE with authentication from /calls/def
    Then the response code is 404
    And the response body is empty

  @failure
  Scenario: User does not provide auth info
    When I DELETE without authentication to /calls/abc
    Then the response code is 401
    And the response body is empty

  @failure
  Scenario: User lacks permission to use the scheduler
    Given I'm not authorized to use the scheduler API
    When I DELETE with authentication from /calls/abc
    Then the response code is 401
    And the response body is empty
