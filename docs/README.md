# OCF Scheduler documentation

> Work in progress. This page holds the plan for these docs until the
> guides are written.

## Who reads what

| Document | Audience | Covers |
|---|---|---|
| [README](https://github.com/cloudfoundry-community/ocf-scheduler#readme), with the boshrelease and kit READMEs | Developers and deployers | Building, testing, configuring, and deploying the scheduler |
| [Concepts](concepts/README.md) | Everyone | What the scheduler's parts are and how they behave |
| [User guide](user-guide/README.md) | App developers, and owners of the endpoints that calls reach | Scheduling jobs and calls |
| [Operator guide](operator-guide/README.md) | Operations teams | Running the scheduler once it is deployed |

## Proposed directions

1. Each guide's `README.md` is its contents and its reading order, as an
   ordered list of links. A book or PDF is built from that list.
2. Each guide reads and prints on its own, so each one lists the concepts
   pages it needs. A combined book includes them once.
3. A cross-reference names its target as well as linking to it, such as
   "see *Operator guide → Upgrading*", so it still works on paper.
4. Headings are unique across all the documents, so their anchors survive
   when the pages are joined.
5. Links inside `docs/` are relative. Links to anything outside it are full
   URLs.
6. Diagrams are image files, not Mermaid blocks, so they print.
7. Concepts say what things are and how they behave. The guides say how to
   do things.
8. Each operator procedure starts with a **Requires:** line that names the
   access it needs, such as a CF admin token, `psql` access to the
   scheduler's database, or BOSH director access. It names the access, not
   a team.
9. The operator guide has four parts after the concepts, and each one
   matches an ITIL 4 management practice:

   | Part | Holds | ITIL 4 practice |
   |---|---|---|
   | *How the scheduler runs* | Background that operators need and users don't, such as where time zone data comes from, running one instance or several, the worker pool, and migrations | Knowledge management |
   | *Routine operations* | Working with the running system without changing the deployment, such as checking health, reading logs, and cleaning up orphans | Service request management; monitoring and event management |
   | *Planned changes* | Changing an existing deployment, such as upgrading, rolling back, scaling, and updating time zone data | Change enablement |
   | *Troubleshooting* | Diagnosing and recovering, such as executions that stay PENDING and tasks that never start. An emergency links to the planned-change procedure it uses, so each procedure is written once | Incident management; problem management |

   *Routine operations* are ITIL standard changes: pre-approved, low risk,
   and done by a set procedure. *Planned changes* are ITIL normal changes:
   assessed each time, and needing an understanding of the deployment.
10. First-time setup, such as registering the service broker, belongs with
    the deployment docs, because a new foundation repeats it. That matches
    the ITIL 4 deployment management practice. Each deployment level
    documents its own part. The service README covers the scheduler
    itself, including deploying it by hand. The BOSH release, the Genesis
    kit, and any later packaging, such as a Kubernetes release, each cover
    only what they add, and link to the level below for the rest.
11. Any guide may have its own troubleshooting section, covering what its
    own audience can see and fix. Its heading names that scope, such as
    "Troubleshooting jobs and calls", so it stays unique across the docs.
12. The guides describe the scheduler itself. When a procedure has a step
    that depends on how the scheduler was deployed, such as redeploying
    with BOSH, it links to that deployment level's docs for the step.

## Open questions

- Whether the user guide needs a tutorial.
- Cron expressions: explain them in concepts, or point to the plugin's
  [cron expression reference](https://github.com/cloudfoundry-community/ocf-scheduler-cf-plugin/blob/develop/cron-expression-reference.md).
- Build tooling, deferred until the first PDF or book is needed.
- README sections that belong in the operator guide, and a pointer from the
  README to these docs, both agreed with the README's maintainers.
