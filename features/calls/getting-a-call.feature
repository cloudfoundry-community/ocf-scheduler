Feature: Getting a Call
  I know that I have calls configured. I want to be able to see the details for
  one of them.

  Background:
    Given I'm authorized to use the scheduler API
    And there's an application with the GUID 123
    And the application has a call with the GUID abc

  Scenario: Successfully getting a Call
    When I GET with authentication from /calls/abc
    Then the response code is 200
    And I receive the requested Call object in the response body

  @failure
  Scenario: Call doesn't exist
    Given there's no call with the GUID def
    When I GET with authentication from /calls/def
    Then the response code is 404
    And the response body is empty

  @failure
  Scenario: User does not provide auth info
    When I GET without authentication to /calls/abc
    Then the response code is 401
    And the response body is empty

  @failure
  Scenario: User lacks permission to use the scheduler
    Given I'm not authorized to use the scheduler API
    When I GET with authentication from /calls/abc
    Then the response code is 401
    And the response body is empty
