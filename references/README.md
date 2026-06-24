# snir Skill Bundle Resources

This repository is structured as a single Anthropic-compatible skill bundle rooted at `SKILL.md`.

## Required Entry

- `SKILL.md` is the skill entrypoint. It contains the frontmatter `name` and `description`, then short operating instructions.

## Bundled Resources

- `references/` contains task-specific documentation that should be opened only when needed.
- `scripts/` contains helper scripts that make skill execution more deterministic.
- `evals/` contains realistic test prompts for checking whether the skill helps an agent use snir correctly.

## Existing Project Documentation

The original project documentation remains in `docs/` to preserve public links and human-facing docs. The skill entrypoint points to both the canonical skill resources and the deeper project docs.

Use the concise files in `references/` first. Open `docs/superpowers/*.md` when the task needs full flag tables or command-specific details.
