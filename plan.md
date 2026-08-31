# Semantic Cache: Production AI-Agent Serving Platform

## Summary

Build a public Apache-2.0 monorepo demonstrating production AI engineering through:

- An OpenAI-compatible Python/FastAPI inference gateway.
- Exact and guarded semantic caching backed by Redis.
- A fictional support incident-management workload.
- Retrieval, fine-tuned cache-safety classification, and LangGraph multi-agent orchestration.
- Fault tolerance, concurrency control, observability, evaluation, cost accounting, load testing, CI/CD, containers, and Kubernetes.
- A Go CLI for workload generation, fault scenarios, and reproducible benchmarks.

Estimated duration: 20 weeks at 8–10 hours/week. Issues target 2–4 hours each.

Ownership:

- You implement gateway policy, cache behavior, resilience, evaluations, agents, fine-tuning, and Go tooling.
- Codex scaffolds repository metadata, contracts, fixtures, workflow skeletons, and reviews your work.
- GitHub Issues and one GitHub Project are the canonical task tracker.

## Compact operating contract

- **Ponytail:** understand first; prefer existing code, standard library, native platform, installed dependency, then minimum new code. Favor root-cause fixes, boring code, deletion, and few files without weakening safety.
- **Caveman:** issues, updates, and handoffs remain terse and evidence-heavy.
- **Bounded execution:** batch independent inspections; sequence dependent work.
- **Cache-first context:** consult this plan, GrayMatter, and code graph when applicable before broad scans.
- **Success and Failure Criteria:** every issue states both.
- **Proof:** each work slice ends with one proportionate verification batch and file-level evidence.

## Architecture

```text
Support clients / LangGraph agents / Go load generator
                         |
                 OpenAI-compatible API
                         |
                  FastAPI Gateway
       auth -> admission -> policy -> cache decision
                    /                 \
          Redis exact/vector       Provider port
          cache + leases       OpenAI / Ollama adapters
                    \                 /
                 response + usage/cost
                         |
        OpenTelemetry + Prometheus + structured logs

Support knowledge base -> separate RedisVL retrieval index
LangGraph checkpoints     -> PostgreSQL
Offline evaluator         -> gold sets, public benchmarks, reports
```

Use a modular monolith initially. Service boundaries are represented by ports, but no component becomes a separate network service without measured scaling or ownership need.

## Semantic layer

The client integration is nearly plug-and-play: change the OpenAI base URL and API key. Safe semantic reuse is not plug-and-play; it requires explicit eligibility, scope, model-version, threshold, and invalidation policy.

### Modes

| Mode | Exact lookup | Semantic lookup | Semantic result served |
|---|---:|---:|---:|
| `disabled` | No | No | No |
| `exact` | Yes | No | No |
| `semantic_shadow` | Yes | Yes | No; counterfactual logged |
| `semantic` | Yes | Yes | Yes |

Defaults:

- Fresh installations start in `exact`.
- `semantic` cannot be enabled until the release evaluation gate passes.
- `Cache-Control: no-store` bypasses lookup and write.
- A request can downgrade caching but cannot upgrade beyond operator policy.
- Tool mutations, personalized/account facts, creative requests, and multi-turn requests bypass semantic reuse.
- Redis failure causes pass-through provider calls; Redis is not required for gateway readiness.

### Configuration and toggle

Versioned TOML provides static policy. Environment variables override deployment values and secrets. Redis holds only a revisioned runtime mode override.

```toml
[cache]
default_mode = "exact"
maximum_mode = "semantic"
ttl_seconds = 3600
namespace_version = "v1"
max_entry_bytes = 262144
semantic_top_k = 5

[embedding]
model = "sentence-transformers/all-MiniLM-L6-v2"
revision = "<pinned-commit>"

[safe_hit]
model_path = "models/safe-hit"
revision = "<artifact-digest>"
calibration_path = "artifacts/policy-calibration.json"

[eligibility]
operations = ["support_kb_answer"]
single_turn_only = true
```

Runtime API:

```http
PUT /admin/v1/cache-mode
Authorization: Bearer <admin-key>
Content-Type: application/json

{
  "mode": "semantic_shadow",
  "expected_revision": 17,
  "reason": "release-0.4 shadow trial"
}
```

The update uses compare-and-set, writes a new revision to Redis, and publishes an invalidation. Replicas retain an in-memory snapshot, consume Pub/Sub updates, and periodically compare revisions. After Redis recovery, a replica refreshes policy before serving semantic hits.

This demonstrates feature flags, kill switches, control plane/data plane separation, optimistic concurrency, eventual consistency, and progressive delivery.

### Cache decision path

1. Authenticate the API key and derive the tenant; never trust a tenant supplied in request content.
2. Validate the supported Chat Completions subset.
3. Resolve static policy, runtime revision, and request downgrade.
4. Classify eligibility as `bypass`, `lookup`, or `refresh`.
5. Canonicalize JSON keys while preserving message text exactly.
6. Hash the full request and scope fingerprint for exact lookup.
7. On eligible semantic requests, embed the current user query locally.
8. Query Redis KNN, prefiltered by tenant, provider, model, locale, plan, region, prompt revision, corpus revision, and output contract.
9. Pass candidates through the fine-tuned safe-hit pair classifier.
10. Serve an accepted hit or record it only in shadow mode.
11. On a miss, acquire a bounded Redis lease to coalesce identical requests.
12. Call the provider with deadline, cancellation, and concurrency controls.
13. Cache only a complete successful response; never cache truncated streams.
14. Emit decision, latency, token, and estimated-cost telemetry without prompt text.

RedisVL supplies vector search, filtering, TTL, and cache storage primitives; this project owns eligibility, scoping, calibration, versioning, and failure policy.

Embedding, prompt, provider model, corpus, or output-schema changes produce a new namespace. Old entries expire; they are never silently reused.

## Public interfaces

### HTTP

- `POST /v1/chat/completions`
  - Supported: `model`, `messages`, `temperature`, `max_tokens`, `stop`, `stream`, `response_format`, `user`.
  - JSON and SSE responses.
  - Unsupported fields return a documented `400`.
- `GET /health/live`
- `GET /health/ready`
- `GET /metrics`
- `GET /admin/v1/cache-mode`
- `PUT /admin/v1/cache-mode`

Response headers:

- `X-Semantic-Cache-Status`: `bypass`, `miss`, `exact-hit`, `semantic-hit`, `shadow-accept`, or `shadow-reject`.
- `X-Semantic-Cache-Policy-Revision`
- `X-Request-ID`
- W3C `traceparent`

The endpoint follows a documented subset rather than claiming complete OpenAI compatibility.

### Core types and ports

- `CacheMode`
- `CacheDecision`
- `RequestFingerprint`
- `ScopeFingerprint`
- `SemanticCandidate`
- `PolicySnapshot`
- `ProviderRequest` / `ProviderResponse`
- `EvaluationRecord`
- `Provider` port: OpenAI and Ollama adapters.
- `CacheStore` port: in-memory test adapter and RedisVL production adapter.
- `EmbeddingProvider` port: local Sentence Transformers implementation.
- `PolicyRepository` port: static policy plus revisioned Redis override.

These interfaces are test surfaces. A second real adapter is added only where the project genuinely needs one.

## Selected stack

| Concern | Selection | Reason |
|---|---|---|
| Python project | `uv`, PEP 621, committed lockfile | Fast, reproducible dependency management |
| HTTP API | FastAPI, Pydantic, Uvicorn | Widely adopted typed async Python stack |
| Configuration | `pydantic-settings`, TOML | Typed validation and environment overrides |
| HTTP clients | HTTPX, official OpenAI SDK | Pooled async clients and official hosted adapter |
| Cache/vector search | `redis-py`, RedisVL | One scalable cache dependency with vector/filter support |
| Embeddings/reranker | Sentence Transformers, PyTorch, Transformers | Established local embedding and cross-encoder tooling |
| Reranker runtime | ONNX Runtime | Portable CPU inference |
| Agent orchestration | LangGraph, `langchain-core` | Explicit state graph and durable checkpoints |
| Checkpoints | PostgreSQL LangGraph checkpointer | Durable resumable workflows |
| Retry | Tenacity | Mature deadline-aware retry composition |
| Telemetry | OpenTelemetry, Prometheus client, structlog | Traces, metrics, and structured logs |
| Testing | pytest, pytest-asyncio, Hypothesis, respx, Testcontainers, Schemathesis | Unit, property, HTTP mock, integration, and schema tests |
| Evaluation | NumPy, Pandas, scikit-learn, MTEB utilities | Transparent metrics and reproducible analysis |
| Go CLI | Cobra, Vegeta library, standard `net/http` | CLI structure plus programmable load generation |
| Fault injection | Toxiproxy | Repeatable latency, reset, and outage scenarios |
| Containers/K8s | Docker Buildx, Compose, Helm, kind | Local demo and realistic small-cluster validation |
| Security | Trivy, pip-audit, govulncheck, CodeQL | Dependency, image, and static security checks |
| Documentation | MkDocs Material | Searchable portfolio and learning documentation |

LiteLLM and a full agent platform are intentionally excluded from the core path because they would conceal provider routing, policy, failure, and measurement behavior that this project is intended to demonstrate.

## Testing data strategy

Hand-authored support-workload cases remain necessary because public corpora do not encode this product’s tenant, plan, region, corpus revision, freshness, and side-effect rules. They are supplemented rather than replaced.

| Dataset | Use | Repository treatment |
|---|---|---|
| vCache SemBenchmark | Semantic-cache classification and threshold comparison | Pin an Apache-2.0 smoke slice with attribution |
| Redis LangCache sentence pairs | Large-scale pair training/generalization | Download manually by pinned revision; never run 40M rows in CI |
| CacheEval | Adversarial false-hit evaluation across domains | Optional downloader; do not vendor due mixed/share-alike constraints |
| MASSIVE | Multilingual intents and workload skew | Pin an attributed Apache-2.0 smoke slice |
| MultiWOZ | Multi-turn state shapes and bypass cases | Optional evaluation, not a semantic-hit release gate |
| MTEB/BEIR tasks | Embedding and retrieval baselines | Run selected tasks manually; verify each dataset license |

Every external sample records source URL, revision, checksum, license, transformations, and intended use. Public scores are regression and generalization signals, not proof of workload correctness.

The bespoke gold set uses group-aware splits by canonical incident, customer, and conversation thread to prevent paraphrase leakage.

## Granular implementation backlog

Issue convention:

- `[U]`: you implement; Codex may review.
- `[S]`: Codex may scaffold; you inspect and approve.
- Every issue contains Goal, Read Before, Inputs, Acceptance, Failure Criteria, and Evidence.
- A task fails if its tests are absent, nondeterministic, silently skipped, or cannot reproduce the claimed result.

### Weeks 1–2: Public foundation

1. **SC-001 [S] Initialize the monorepo** — Read: modular monolith, monorepo, reproducible builds. Create Python and Go workspaces, Apache-2.0 license, lockfiles, `.editorconfig`, and minimal commands. Done when clean checkout runs one Python and one Go smoke test.
2. **SC-002 [S] Add public governance files** — Read: open-source governance, responsible disclosure. Add contributing, security, code of conduct, support, issue forms, PR template, and attribution policy. Done when all public contact paths are documented without personal secrets.
3. **SC-003 [U] Define the domain model** — Read: ubiquitous language, bounded context. Create a concise `CONTEXT.md` defining request, tenant, cache entry, semantic equivalence, false hit, corpus revision, policy revision, incident, and resolution. Done when tests and docs use the same terms.
4. **SC-004 [U] Record foundational decisions** — Read: ADRs, reversible vs irreversible decisions. Record modular monolith, Redis fail-open, explicit OpenAI subset, and shadow-before-serve. Done when each ADR includes context, decision, alternatives, and consequences.
5. **SC-005 [S] Configure GitHub Project and labels** — Read: trunk-based development, WIP limits. Add milestones, issue labels for subsystem/principle/risk, required-check names, and issue import data. Done when every backlog item can be imported once without a parallel Markdown tracker.

### Weeks 3–4: Gateway vertical slice

6. **SC-006 [U] Specify the HTTP contract** — Read: API compatibility, Postel’s law limits, schema evolution. Write OpenAPI examples and rejection behavior. Done when Schemathesis and golden JSON fixtures agree.
7. **SC-007 [U] Implement API-key tenant resolution** — Read: multi-tenancy, least privilege, trust boundaries. Store only hashed keys and attach tenant context server-side. Done when forged tenant content cannot cross scope.
8. **SC-008 [U] Define the provider port** — Read: dependency inversion, ports and adapters, deep modules. Keep provider differences behind one small typed interface. Done when a deterministic fake passes the same contract tests.
9. **SC-009 [U] Build the Ollama adapter** — Read: async I/O, connection pooling, deadlines. Reuse one lifespan-managed HTTPX client. Done when JSON and provider errors map to gateway responses.
10. **SC-010 [U] Build the OpenAI adapter** — Read: adapter pattern, error translation. Use the official SDK with a pooled client. Done when mocked 429/500/timeout responses preserve the gateway error contract.
11. **SC-011 [U] Proxy non-streaming completions** — Read: reverse proxy, stateless services, request correlation. Done when an official OpenAI client can target the gateway base URL.
12. **SC-012 [U] Proxy SSE completions** — Read: streaming, cancellation, backpressure. Stop upstream work on disconnect and never buffer without a limit. Done when finite, cancelled, and malformed streams are tested.
13. **SC-013 [U] Implement health semantics** — Read: liveness vs readiness, fail-open dependencies. Redis/provider outages must not cause restart loops. Done when probes reflect process/config health and fault tests confirm behavior.

### Weeks 5–6: Exact caching

14. **SC-014 [U] Canonicalize requests** — Read: canonical representation, deterministic keys. Sort structural JSON keys but preserve message bytes and list order. Done with property tests for determinism and semantic-field sensitivity.
15. **SC-015 [U] Construct scope fingerprints** — Read: cache partitioning, tenant isolation, versioned namespaces. Include tenant, provider/model, prompt, corpus, locale, plan, region, and output contract. Done when changing any dimension forces a miss.
16. **SC-016 [U] Implement the in-memory CacheStore** — Read: seam vs adapter, test doubles. Use it only for contract tests and local development. Done when TTL and atomic set-if-absent semantics pass a fake-clock suite.
17. **SC-017 [U] Implement Redis exact caching** — Read: cache-aside, TTL, serialization. Done when Testcontainers verifies hit, miss, expiry, corruption handling, and maximum-entry limits.
18. **SC-018 [U] Add `disabled` and `exact` modes** — Read: feature flags, kill switches. Done when mode transitions are observable and `no-store` bypasses both read and write.
19. **SC-019 [U] Add exact-hit SSE replay** — Read: protocol adaptation, bounded buffering. Re-emit valid deterministic SSE from a complete cached response. Done when the official client consumes both cached JSON and streaming responses.
20. **SC-020 [U] Test exact-cache isolation** — Read: negative testing, defense in depth. Generate tenant/model/revision matrices. Failure criterion: any cross-scope hit.

### Weeks 7–9: Semantic shadow layer

21. **SC-021 [U] Add the embedding port and local adapter** — Read: vector embeddings, model versioning. Pin model revision and batch with bounded concurrency. Done when identical inputs are stable and revision changes namespace.
22. **SC-022 [U] Create the RedisVL vector schema** — Read: ANN/HNSW, metadata filtering. Done when KNN queries cannot return another tenant before application reranking.
23. **SC-023 [U] Implement semantic eligibility rules** — Read: policy engines, safe defaults. Allow only declared single-turn support knowledge answers. Done with explicit reason codes for every bypass.
24. **SC-024 [U] Retrieve semantic candidates** — Read: two-stage retrieval/reranking, precision vs recall. Return top-k candidate metadata without yet serving it. Done when deterministic fixtures verify ordering and filters.
25. **SC-025 [U] Add `semantic_shadow`** — Read: dark launches, counterfactual evaluation. Record candidate, acceptance decision, and avoided-cost estimate while serving the provider result. Done when response bytes remain identical to `exact` mode.
26. **SC-026 [U] Build the revisioned mode repository** — Read: control plane/data plane, optimistic concurrency, eventual consistency. Implement Redis CAS, Pub/Sub, local snapshot, and periodic revision check. Done when two replicas converge and stale writers receive `409`.
27. **SC-027 [U] Implement the admin mode API** — Read: authenticated operations, auditability. Require admin credentials, expected revision, and reason. Done when unauthorized and lost-update cases fail closed.
28. **SC-028 [U] Add namespace lifecycle handling** — Read: cache invalidation, immutable versions. New versions write to new prefixes; old versions expire. No destructive global flush.
29. **SC-029 [U] Run the first shadow analysis** — Read: calibration, confusion matrix, Wilson confidence interval. Produce threshold curves, false-hit rate, hit rate, and cost avoidance. Done when no serving threshold is selected from the test split.

### Weeks 10–11: Reliability and concurrency

30. **SC-030 [U] Enforce end-to-end deadlines** — Read: timeout budgets, cancellation propagation. Split one request budget among queue, embedding, Redis, and provider. Done when no child operation outlives the request.
31. **SC-031 [U] Add bounded retries** — Read: exponential backoff, jitter, retry storms. Use Tenacity only for retryable pre-response failures. Never retry after the first streamed byte.
32. **SC-032 [U] Add provider bulkheads** — Read: bounded concurrency, backpressure, admission control. Use per-provider capacity limits and bounded queue time. Done when overload rejects promptly instead of exhausting memory.
33. **SC-033 [U] Implement circuit-breaker state** — Read: closed/open/half-open state machines. Inject the clock and test transitions deterministically. Done when outages stop repeated calls and probes remain bounded.
34. **SC-034 [U] Implement distributed rate limiting** — Read: token bucket, atomic Redis scripts, fail-open vs fail-closed. Apply per-tenant/provider limits; document local fallback behavior.
35. **SC-035 [U] Coalesce identical misses** — Read: cache stampede, singleflight, leases. Use bounded Redis leases with owner tokens. Done when 100 concurrent identical misses produce at most one provider call per lease window.
36. **SC-036 [U] Handle lease-owner failure** — Read: leases, fencing, crash recovery. Done when a killed owner cannot block the key beyond lease expiry or overwrite a newer result.
37. **SC-037 [U] Add optional idempotency keys** — Read: idempotency, replay scope. Scope keys by tenant and canonical request. Reject key reuse with different content.
38. **SC-038 [U] Complete the resilience fault suite** — Read: fault injection, graceful degradation. Cover Redis down, slow Redis, 429, 500, timeout, truncated stream, disconnect, and pod termination.

### Weeks 12–13: Evaluation system

39. **SC-039 [S] Create dataset provenance tooling** — Read: data lineage, licensing, reproducibility. Manifest every source, revision, checksum, transformation, and license.
40. **SC-040 [U] Import the vCache smoke suite** — Read: external validity, benchmark contamination. Pin an attributed Apache-2.0 slice. Done when offline runs are deterministic.
41. **SC-041 [U] Add optional CacheEval/MASSIVE downloaders** — Read: license compatibility, dataset shift. Keep separately licensed data outside the Apache code artifact.
42. **SC-042 [U] Build the support knowledge base** — Read: synthetic data, referential integrity. Define products, incidents, regions, plan rules, changing facts, and corpus revisions.
43. **SC-043 [U] Build the support-workload gold set** — Read: train/validation/test leakage, group-aware splitting. Include paraphrases, negations, entity swaps, number changes, stale facts, and cross-tenant traps.
44. **SC-044 [U] Implement the evaluator** — Read: precision, recall, false-positive cost, confidence intervals. Output machine-readable JSON plus a human Markdown report.
45. **SC-045 [U] Add metamorphic tests** — Read: metamorphic and differential testing. Paraphrases should preserve eligible meaning; changed entities, quantities, negation, scope, or time should force rejection.
46. **SC-046 [U] Define the release-quality gate** — Read: SLI/SLO, error budgets. Block semantic serving unless the held-out support-workload false-hit upper 95% confidence bound is at most 1%.
47. **SC-047 [U] Add cost accounting** — Read: unit economics, cardinality control. Version provider price data outside code and compute input/output tokens, avoided calls, and estimated savings.

### Weeks 14–15: Retrieval and multi-agent workload

48. **SC-048 [U] Index the support corpus** — Read: chunking, hybrid retrieval, corpus versioning. Keep the RAG index separate from cached responses.
49. **SC-049 [U] Evaluate retrieval independently** — Read: Recall@k, MRR, nDCG, offline/online skew. Target Recall@5 ≥0.85 on held-out support questions before agent integration.
50. **SC-050 [U] Define LangGraph state** — Read: state machines, DAGs, durable execution. Model incident context, retrieved evidence, draft, verification result, and bounded retry count.
51. **SC-051 [U] Implement triage and retrieval specialists** — Read: orchestration vs choreography, tool contracts. Use structured outputs and deterministic routing.
52. **SC-052 [U] Implement composer and verifier specialists** — Read: separation of duties, grounded generation. The verifier must cite retrieved document IDs or reject the draft.
53. **SC-053 [U] Add PostgreSQL checkpoints** — Read: at-least-once execution, idempotent steps, recovery. Done when a killed workflow resumes without duplicating side effects.
54. **SC-054 [U] Evaluate the complete agent** — Read: component vs end-to-end evaluation. Report retrieval failure, generation failure, verification rejection, latency, tokens, and cache effects separately.

### Week 16: Fine-tuned safe-hit model

55. **SC-055 [U] Construct pair-classification data** — Read: hard negatives, class imbalance, leakage. Combine training-only support-workload pairs with permitted public pairs; keep evaluation incidents isolated.
56. **SC-056 [U] Establish heuristic and pretrained baselines** — Read: champion/challenger, matched operating points. Compare distance-only and pretrained cross-encoder decisions.
57. **SC-057 [U] Fine-tune the MiniLM cross-encoder** — Read: transfer learning, early stopping, calibration. Run GPU training only through a documented manual workflow.
58. **SC-058 [U] Export and version the model** — Read: model artifacts, reproducible inference. Pin tokenizer/base revision, export ONNX, record digest and model card.
59. **SC-059 [U] Calibrate the two-stage policy** — Read: threshold calibration, selective classification. Select embedding and reranker thresholds on validation data for the declared false-hit budget.
60. **SC-060 [U] Promote `semantic` mode** — Read: progressive delivery, rollback. Promotion requires held-out evaluation, shadow evidence, and a working runtime kill switch. Otherwise remain in shadow.

### Week 17: Observability and Go load tooling

61. **SC-061 [U] Add structured logs** — Read: data minimization, correlation IDs. Never log prompts, completions, credentials, embeddings, or unbounded tenant labels.
62. **SC-062 [U] Instrument traces** — Read: distributed tracing, context propagation. Trace policy, Redis, embedding, reranker, provider, and agent nodes through OpenTelemetry.
63. **SC-063 [U] Add RED metrics** — Read: rate/errors/duration, histogram buckets, cardinality. Measure decisions, queueing, provider calls, retries, tokens, and estimated cost.
64. **SC-064 [U] Build dashboards and alerts** — Read: SLIs, SLOs, burn rates. Include false-hit proxy, cache hit mix, provider avoidance, p50/p95/p99, saturation, and circuit state.
65. **SC-065 [U] Build the Go CLI shell** — Read: command-query separation, stable CLI contracts. Commands: `replay`, `load`, `fault`, and `report`.
66. **SC-066 [U] Implement workload models** — Read: open vs closed workloads, Poisson arrivals, Zipf skew, Little’s Law. Support fixed seeds and recorded scenario manifests.
67. **SC-067 [U] Integrate Vegeta and Toxiproxy** — Read: coordinated omission, tail latency, fault injection. Produce JSON and Markdown reports from load-plus-fault runs.
68. **SC-068 [U] Publish the benchmark method** — Read: capacity planning, reproducibility. Record hardware, versions, dataset, arrival model, warm-up, duration, and confidence intervals.

### Weeks 18–19: Containers, Kubernetes, and CI/CD

69. **SC-069 [S] Add one-command Compose** — Read: twelve-factor configuration, dependency health. Start gateway, Redis, PostgreSQL, Ollama, collector, Prometheus, Tempo, and Grafana.
70. **SC-070 [U] Build hardened images** — Read: multi-stage builds, non-root containers, immutable filesystems. Pin base digests and include health checks.
71. **SC-071 [S] Create the Helm chart** — Read: declarative infrastructure, configuration separation. Include Deployment, Service, ConfigMap, Secret references, and ServiceMonitor.
72. **SC-072 [U] Add production pod behavior** — Read: graceful shutdown, rolling updates, disruption budgets. Configure probes, termination grace, resource requests/limits, and PDB.
73. **SC-073 [U] Add autoscaling policy** — Read: horizontal scaling, saturation Semantic s. HPA uses CPU initially; document why request/concurrency metrics would be a later improvement.
74. **SC-074 [S] Build PR CI** — Read: quality gates, test pyramid. Run Python lint/type/unit/property/integration, Go format/vet/test/race, OpenAPI tests, evaluation smoke, image build, Helm validation, and security scans.
75. **SC-075 [S] Add ephemeral kind validation** — Read: environment parity, deployment smoke tests. Helm install with `--atomic --wait`; verify exact hit, bypass, tenant isolation, TTL, and Redis fail-open.
76. **SC-076 [S] Add nightly workflows** — Read: test-cost partitioning. Run public benchmark slices, threshold sweeps, load, fuzzing, and Toxiproxy faults on schedule.
77. **SC-077 [S] Add manual paid/GPU workflows** — Read: protected environments, budget controls. Hosted OpenAI evaluation and fine-tuning require `workflow_dispatch`, explicit budget input, and protected secrets.
78. **SC-078 [S] Add release provenance** — Read: build once/promote, SBOM, supply-chain provenance. Publish one immutable image digest and chart to GHCR with attestations only after gates pass.
79. **SC-079 [U] Define rollback** — Read: rollback, canary deployment, database compatibility. Helm uses `--atomic`; cache namespaces are additive; runtime mode reverts immediately to `exact`.

### Week 20: Portfolio release

80. **SC-080 [U] Write the architecture narrative** — Read: C4 model, trade-off communication. Explain decisions, rejected alternatives, measured limitations, and failure modes.
81. **SC-081 [U] Publish evaluation and cost reports** — Read: honest benchmarking, statistical uncertainty. Include raw machine-readable summaries and reproducible commands.
82. **SC-082 [U] Create the demo scenario** — Read: incident response, graceful degradation. Demonstrate cold miss, exact hit, shadow candidate, semantic hit, Redis outage, provider fault, kill switch, and recovery.
83. **SC-083 [U] Record the demo video** — Keep it under ten minutes and show dashboards, traces, GitHub checks, benchmark output, and Kubernetes recovery.
84. **SC-084 [U] Run the public-repo audit** — Check secret history, licenses, attribution, generated artifacts, links, clean-clone setup, accessibility, and security policy.
85. **SC-085 [U] Tag `v1.0.0`** — Publish signed/attested images, chart, SBOM, model card, dataset card, evaluation report, and documented known limitations.

## CI/CD policy

### Pull requests

Required checks:

- Python: locked `uv` install, Ruff, mypy, pytest unit/property/contract/integration.
- Go: gofmt, vet, test, race detector, bounded fuzz smoke.
- API: OpenAPI validation and Schemathesis.
- Evaluation: support-workload gold smoke plus pinned public sample.
- Hard gates: zero tenant leaks, 100% `no-store` bypass, no semantic serving before promotion.
- Supply chain: dependency review, pip-audit, govulncheck, CodeQL, secret scan, Trivy, SBOM.
- Deployment: image build without push, Helm lint/template, kubeconform, ephemeral kind smoke.
- Evidence: JUnit, coverage, evaluation JSON, rendered Helm manifest, image digest, and redacted traces.

Third-party actions are commit-SHA pinned. Workflow permissions are minimal. Dependabot opens dependency and action updates.

### Nightly and manual

- Nightly: larger public evaluation, threshold sweeps, race/fuzz runs, load tests, and fault injection.
- Manual protected workflows: paid OpenAI evaluation, GPU fine-tuning, and deployment to a user-provided cluster.
- Manual cluster deployment uses OIDC rather than long-lived cloud credentials.

### Release

This repository implements Continuous Delivery, not unattended production deployment:

1. Build the image once.
2. Verify the same immutable digest.
3. Generate SBOM and provenance.
4. Publish the image and Helm chart to GHCR.
5. Create a GitHub Release for a SemVer tag.
6. Optionally deploy that digest through a protected manual workflow.
7. Run health/evaluation smoke checks.
8. Roll back Helm and set cache mode to `exact` on failure.

## Acceptance criteria

The project is portfolio-ready when:

- A clean checkout starts locally through one Compose command.
- An official OpenAI client can use the documented gateway subset.
- Exact, shadow, and semantic modes behave according to policy.
- Zero cross-tenant hits occur in all automated suites.
- `no-store` bypasses lookup and write in every mode.
- Support-workload semantic false-hit upper 95% confidence bound is ≤1%.
- Semantic caching avoids at least 30% of hosted calls on the declared replay workload without violating the false-hit budget.
- Exact-hit p95 is ≤20% of the deterministic uncached-provider p95.
- At 200 concurrent clients and 10,000 requests, gateway error rate is <1% excluding injected upstream failure.
- One hundred concurrent identical misses produce at most one upstream call per lease window.
- Redis outage degrades to provider pass-through without restart loops.
- Truncated or cancelled responses are never cached.
- Retrieval Recall@5 is ≥0.85 on the held-out support-workload set.
- Fine-tuned reranking improves false-hit rate at matched hit rate over distance-only baseline, or the model is not promoted.
- Agent results report component-level retrieval, generation, verification, latency, token, cache, and cost metrics.
- PR, nightly, release, SBOM, provenance, Helm, and rollback workflows are documented and reproducible.

## Failure criteria

Do not claim completion if:

- Semantic caching is only a similarity threshold with no tenant scoping, eligibility rules, calibration, or kill switch.
- Benchmark test data leaks into training or threshold selection.
- External datasets lack pinned revisions, checksums, licenses, or attribution.
- Semantic hits are enabled because a mean score looks good while false-hit uncertainty is unreported.
- Redis/provider failures cause unbounded retries, queues, memory, or restart loops.
- CI depends on paid APIs or GPUs for ordinary pull requests.
- Images are rebuilt between verification and release.
- Prompts, completions, API keys, or high-cardinality tenant identifiers appear in telemetry.
- The multi-agent workflow is an unmeasured collection of LLM calls without explicit state, bounded transitions, or component evaluation.
- The public README claims complete OpenAI compatibility or production certification beyond the tested subset and workload.

## Assumptions

- Project name: **Semantic Cache**; repository and deployment slug: `semantic-cache`; Python package: `semantic_cache`.
- Workload: fictional, unbranded support incident-management SaaS data and workflows.
- Python owns gateway, policy, evaluation, retrieval, and agents; Go owns load and fault tooling.
- Providers: local Ollama plus one hosted OpenAI adapter.
- Redis is an expendable fail-open cache; PostgreSQL persists LangGraph checkpoints.
- One process runs per Kubernetes pod; horizontal scale comes from replicas.
- Semantic thresholds are generated by calibration, not committed as arbitrary defaults.
- No permanent public endpoint is required; Compose, kind, reports, and video form the demo.
- GPU training and paid-provider evaluations remain manual.
- The public code is Apache-2.0; externally sourced data and models retain their own licenses.
