# Sub-Agent Review Index

The assessment used five sub-agents. All were requested with `gpt-5.5` and
`high` or `xhigh` reasoning.

| Agent | Model / Reasoning | Persona | Artifact |
| --- | --- | --- | --- |
| Curie | `gpt-5.5 high` | CLI UX researcher | [agent-cli-ergonomics.md](agent-cli-ergonomics.md) |
| Laplace | `gpt-5.5 high` | Go/config architecture reviewer | [agent-config-architecture.md](agent-config-architecture.md) |
| Faraday | `gpt-5.5 high` | Release manager and docs lead | [agent-release-docs.md](agent-release-docs.md) |
| Schrodinger | `gpt-5.5 xhigh` | Adversarial process auditor | [agent-process-adversarial.md](agent-process-adversarial.md) |
| Sagan | `gpt-5.5 xhigh` | QA/performance/composability analyst | [agent-quality-metrics.md](agent-quality-metrics.md) |

## Requirement Coverage

The README required:

- at least four sub-agents for the product outcome assessment, with at least two
  on `gpt-5.5 high` or better;
- at least five sub-agents for the meta-process assessment, using `gpt-5.5 high`
  or better.

This assessment used the same five reviewers for both phases. Each reviewed
product outcomes and process lessons through a different lens, and their notes
are preserved as markdown files in this directory.
