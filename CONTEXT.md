# Semantic Cache Domain

Semantic Cache is an AI-agent serving platform that reuses model responses only when the current request remains safely equivalent to a previously answered request. This glossary defines the product language shared by documentation, code, tests, and evaluations.

## Request and isolation

**Request**:
A tenant-scoped demand for a model response, including the conversation input and response constraints that can affect the correct result.
_Avoid_: Query, prompt

**Tenant**:
The organization or customer boundary within which requests, policy, and cached responses may be shared. A tenant is established from trusted credentials rather than request content.
_Avoid_: User, account

## Cache safety

**Cache entry**:
A complete successful model response retained with the request identity, tenant scope, and revisions needed to determine whether it may be reused.
_Avoid_: Cached answer, cache item

**Semantic equivalence**:
The relationship between two requests when the same response remains correct for both under the same tenant scope, relevant facts, revisions, and response constraints. Similar wording alone does not establish semantic equivalence.
_Avoid_: Semantic similarity, semantic closeness

**False hit**:
A cache reuse decision that returns a response which is not correct for the current request, despite treating that request as equivalent to the cached request.
_Avoid_: False positive, cache collision

## Change boundaries

**Corpus revision**:
An identifier for a specific state of the support knowledge used to answer requests. A change to relevant facts creates a new corpus revision so responses based on older knowledge are not silently reused.
_Avoid_: Corpus version, knowledge version

**Policy revision**:
An identifier for a specific state of the rules that govern cache eligibility and reuse. It distinguishes rule changes from changes to support knowledge or models.
_Avoid_: Policy version, configuration version

## Support workload

**Incident**:
A support situation in which a product or service is disrupted, degraded, or behaving unexpectedly for affected customers.
_Avoid_: Exception, bug, alert

**Resolution**:
A validated outcome that addresses an incident and restores the expected service or gives the affected customer an effective remedy.
_Avoid_: Fix, answer, workaround
