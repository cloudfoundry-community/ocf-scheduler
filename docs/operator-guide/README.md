# Operator guide

For the teams that run the scheduler once it is deployed. Each procedure
starts with a **Requires:** line that names the access it needs.

## Contents

- Concepts (included from [Concepts](../concepts/README.md))

### Routine operations

Repeatable tasks that follow a set procedure. Planned:

- Checking health and reading logs
- Executions that stay PENDING, and tasks that never start
- Finding and removing orphaned jobs, calls, and schedules

### Planned changes

Changes that need an understanding of the deployment. Planned:

- Registering the service broker
- Upgrading, and rolling back
- Running more than one instance
- Where time zone data comes from
