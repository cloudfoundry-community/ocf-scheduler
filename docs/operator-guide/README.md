# Operator guide

For the teams that run the scheduler once it is deployed. Each procedure
starts with a **Requires:** line that names the access it needs.
First-time setup, such as registering the service broker, is in the
deployment docs for each way of deploying the scheduler.

## Contents

- Concepts (included from [Concepts](../concepts/README.md))

### How the scheduler runs

Background that operators need and users don't. Planned:

- Where time zone data comes from
- Running one instance or several
- The worker pool
- Database migrations

### Routine operations

Working with the running system without changing the deployment.
Planned:

- [Connecting to the scheduler's database](database-access.md)
- Checking health and reading logs
- Finding and removing orphaned jobs, calls, and schedules

### Planned changes

Changing an existing deployment. Planned:

- Upgrading, and rolling back
- Scaling
- Updating time zone data

### Troubleshooting

Diagnosing and recovering. Planned:

- Executions that stay PENDING
- Tasks that never start
