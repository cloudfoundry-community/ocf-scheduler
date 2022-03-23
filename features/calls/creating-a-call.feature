Feature: Creating a Call
  I have a URL that I want to occasionally ping. To that end, I want to be
  able to store that request for execution as a Call.

  Background:
    Given I'm authorized to use the scheduler API
    And there's an application with the GUID 123
    And there are no Calls for that application
    And my JSON payload looks like this:
    """
    {
      "name" : "sausages",
      "url" : "http://sausages.io/turned/to/gold",
      "auth_header" : "i-am-the-law"
    }
    """

  Scenario: Successfully creating a Call
    When I POST my payload with authentication to /calls?app_guid=123
    Then the response code is 201
    And I receive a Call object in the response body
    And the Call has a GUID
    And the Call is named "sausages"
    And the Call's URL is "http://sausages.io/turned/to/gold"
    And the Call's auth header is "i-am-the-law"
    And the Call has an app GUID
    And the Call has a space GUID
    And the Call is timestamped

  @failure
  Scenario: User does not provide auth info
    When I POST my payload without authentication to /calls?app_guid=123
    Then the response code is 401
    And the response body is empty


  @failure
  Scenario: User lacks permission to use the scheduler
    Given I'm not authorized to use the scheduler API
    When I POST my payload with authentication to /calls?app_guid=123
    Then the response code is 401
    And the response body is empty

  @failure
  Scenario: Name collision
    Given there's already a sausages call for my application
    When I POST my payload with authentication to /calls?app_guid=123
    Then the response code is 422
    And the response body is empty

  @failure
  Scenario: No call name provided
    Given the name field is missing from my payload
    When I POST my payload with authentication to /calls?app_guid=123
    Then the response code is 422
    And the response body is empty

  @failure
  Scenario: No URL provided
    Given the url field is missing from my payload
    When I POST my payload with authentication to /calls?app_guid=123
    Then the response code is 422
    And the response body is empty

  @failure
  Scenario: No auth header provided
    Given the auth_header field is missing from my payload
    When I POST my payload with authentication to /calls?app_guid=123
    Then the response code is 422
    And the response body is empty
  
  @failure
  Scenario: App GUID not provdided
    When I POST my payload with authentication to /calls
    Then the response code is 422
    And the response body is empty


