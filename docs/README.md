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
9. The operator guide has two parts, named after the ITIL change types.
   *Routine operations* are standard changes: pre-approved, low risk, and
   done by a set procedure. *Planned changes* are normal changes: assessed
   each time, and needing an understanding of the deployment.

## Open questions

- Whether the user guide needs a tutorial.
- Cron expressions: explain them in concepts, or point to the plugin's
  [cron expression reference](https://github.com/cloudfoundry-community/ocf-scheduler-cf-plugin/blob/develop/cron-expression-reference.md).
- Build tooling, deferred until the first PDF or book is needed.
- README sections that belong in the operator guide, and a pointer from the
  README to these docs, both agreed with the README's maintainers.
