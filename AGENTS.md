# Semantic Cache Agent Router

This repository is a learning-first build of a production AI-agent serving platform.
The human writes core implementation; agents default to teaching, questioning, and review.

## Read routes

| Task | Read first | Then read |
|---|---|---|
| Understand the project | `CONTEXT.md` | Relevant section of `plan.md` |
| Start an issue | `work/CONTEXT.md` | Relevant roadmap entry and template |
| Continue an issue | `work/active/<issue>/notes.md` | Issue, relevant code, and tests |
| Review understanding | Active issue `notes.md` | Issue acceptance and primary references |
| Review implementation | Active issue `notes.md` | Diff, interface, and acceptance tests |
| Make an architecture decision | `docs/architecture/folder-map.md` | Relevant ADR in `docs/decisions/` when it exists |
| Run evaluations or training | Local `CONTEXT.md` in that folder | Its declared manifests/configuration |

## Working rules

- Keep one active issue at a time.
- Treat `plan.md` as read-only roadmap input. Edit or include it in an issue commit only when the user explicitly requests a roadmap change.
- Do not copy the full roadmap into work folders; link to the GitHub issue and `plan.md`.
- Learn by doing: write enough in `notes.md` to attempt the issue safely, get a review, then implement.
- Use a separate design or ADR only for cross-cutting, risky, or hard-to-reverse decisions.
- Append verification and lessons to the same `notes.md` after implementation.
- Agents do not implement core modules unless explicitly asked.
- Reviews test the module through its interface and cite concrete files and commands.
- Put each fact in one durable home and link to it elsewhere.
- Generated outputs belong under `artifacts/`; publish only reviewed summaries in `docs/`.
- Do not create generic `utils`, `common`, `shared`, or `misc` folders.
- Create target code folders only when the first real file needs them.

## Sources of truth

- Roadmap and acceptance criteria: `plan.md`
- Current understanding and evidence: active issue `notes.md`
- Runtime behavior: code and tests
- Durable decisions and learning: `docs/`
- Public scheduling: GitHub Issues and GitHub Project after repository initialization
