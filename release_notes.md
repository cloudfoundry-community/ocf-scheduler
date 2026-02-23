# Breaking Change

* THe Day of Month(DOM) and Day of Week(DOW) fields 

# Bug Fixes

*

# Improvements

## CAPI v3 Migration

* Upgraded go-cfclient from v3.0.0-alpha.9 to v3.0.0-alpha.18
* Removed legacy /v2/info endpoint from mock CF API server

## Cyclic Ranges

The OCF Scheduler ranges now supports cyclic ranges for the minutes, hours, days of the week, and months fields.
A cyclic range means the start value can be larger than the ending value.   This allows cron expressions
to be more expressive instead of creating multiple ranges.  It also makes fields with named values work without
needing to know the actual values.

Examples
|expression|field|description|
|:---------|:-------: |-----------|
|57-3      | minutes | 57, 58, 59, 0, 1, 2, 3|
|23-1      |  hours  | 23, 0, 1 (spans midnight)|
|FRI-MON   |   DOW   | FRI, SAT, SUN, MON (spans the weekend)|
|DEC-FEB   |  month  | DEC, JAN, FEB (spans year boundary)|
|26-3      |  DOM    | Wraps correctly for every month

### Cyclic Step Expressions

Cyclic ranges also work in step expressions

|expression|field|description|
|:---------|:-------: |-----------|
|20-2/2    |hours     | every 2 hours from 8 pm(20) to 2 am|

# Software Components

## Core Components

| Release | Version | Release Date | Type | Changed |
| ------- | ------- | ------------ | ---- | :-----: |
| go-cron | [1.12.0][go-cron]|16-Jan-2026|source|[X](## was robfig cron v3.0.1)
| go-cfclient | [v3.0.0-alpha.18][go-cfclient]|22-Feb-2026|source|[X](## was v3.0.0-alpha.9)

[go-cron]: https://github.com/netresearch/go-cron/releases/tag/v1.12.0
[go-cfclient]: https://github.com/cloudfoundry/go-cfclient/releases/tag/v3.0.0-alpha.18
