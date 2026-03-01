# Monit — Distributed Uptime Monitoring System

A **high-performance, fault-tolerant, scalable, reliable** uptime monitoring service built in Go, designed to handle **1M+ monitoring jobs with a single server** with distributed Redis-based scheduling, atomic Lua scripts, and a channel-based worker pipeline

## Table of Contents

- [System Overview](#system-overview)
- [Architecture](#architecture)
- [Core Pipeline](#core-pipeline)
- [Distributed Scheduling](#distributed-scheduling)
- [Reliability & Fault Tolerance](#reliability--fault-tolerance)
- [Performance Optimizations](#performance-optimizations)
- [Scalability Analysis](#scalability-analysis)
- [Engineering Challenges](#engineering-challenges)
- [Why Go](#why-go)
- [Code Quality & Design](#code-quality--design)
- [Project Structure](#project-structure)
- [Tech Stack](#tech-stack)
- [Configuration](#configuration)
- [Database Schema](#database-schema)
- [Getting Started](#getting-started)
- [API Reference](#api-reference)

---

## System Overview

Monit is a production-grade website monitoring service that continuously checks the health of registered URLs. When a monitored endpoint fails, the system detects it, retries intelligently, creates incidents, and triggers alerts — all while maintaining high throughput and zero job loss guarantees.

### What It Does

1. **Schedules** millions of HTTP health checks using Redis sorted sets
2. **Executes** checks concurrently with configurable worker pools and semaphore-controlled HTTP concurrency
3. **Processes results** through dedicated success/failure pipelines with retry logic
4. **Detects incidents** using Redis-backed state machines with atomic operations
5. **Alerts** users when sustained failures exceed configurable thresholds
6. **Recovers** automatically from crashes via inflight job reclamation

### Key Properties

#### Highly Scalable

- **Channel-based pipeline architecture** — each stage (Scheduler → Executor → Result Processor → Alert Service) is connected via buffered Go channels, allowing independent scaling of each stage
- **Configurable worker pools** — the number of executor workers, HTTP workers, success/failure workers, and alert workers are all configurable, so you can scale each pool independently based on bottlenecks
- **Redis sorted sets for O(log N) scheduling** — finding due jobs among 1M monitors is logarithmic, not linear. A database `SELECT WHERE next_run <= NOW()` would require a full table scan
- **Horizontal scaling support** — multiple instances can run simultaneously because Lua scripts guarantee atomic job dispatch, so no two instances ever grab the same job

#### Highly Reliable

- **Lua script atomicity** — all scheduling operations (fetch due jobs, move to inflight, reclaim stalled jobs) are executed as atomic Lua scripts on Redis, eliminating race conditions across distributed instances
- **Inflight visibility timeout** — every dispatched job is tracked in an inflight sorted set with a timeout score. If a worker doesn't acknowledge the job within the timeout, the system automatically considers it lost
- **Automatic job reclamation** — the Reclaimer runs independently on a ticker, scanning for expired inflight jobs and atomically moving them back to the schedule set for re-execution
- **HSETNX for alert deduplication** — even with multiple failure workers processing the same monitor's failures, `MarkIncidentAlertedIfNotSet` uses Redis `HSETNX` to ensure only one worker triggers the alert

#### Fault Tolerant

- **Graceful shutdown with ordered channel closure** — channels are closed in strict dependency order (`jobChan` → executor stop → `resultChan` → result processor wait → `alertChan` → alert service wait), ensuring every in-flight message is fully processed before the process exits
- **Backpressure protection** — if the executor can't keep up and `jobChan` is full, the scheduler reschedules jobs with a 2-second backoff + random jitter instead of dropping them
- **Retry with exponential backoff** — all Redis operations use a `retry()` helper with progressive delays (50ms, 100ms, 150ms), and the result processor retries failed HTTP checks before escalating to incidents
-  **Robust Error handling** — all the errors related to databases, redis, services are handling properly with a custom error type , this ensures separation of concern, security, easier debugging, and robust system.

#### High Performance

- **Redis caching eliminates DB reads on the hot path** — monitor configuration is cached in Redis with a 24-hour TTL. At 1 Million checks/min, this avoids ~16,666 PostgreSQL queries/second for config lookups. Cache hits are sub-millisecond vs. 1-5ms for database queries
- **HTTP semaphore caps concurrent connections** — a `chan struct{}` of configurable size (e.g., 5000) prevents file descriptor exhaustion and allows precise control over outbound network pressure
- **Batch operations** — Lua scripts fetch up to `batchSize` jobs per tick in a single Redis roundtrip, and `ScheduleBatch()` uses multi-member `ZADD` to schedule many jobs in one call
- **Connection pool tuning** — both Redis and PostgreSQL pools are fully configurable (pool size, idle connections, max lifetime) to optimize for the deployment environment

#### Highly Maintainable

- **Clean module boundaries** — each domain module (`user`, `monitor`, `scheduler`, `executor`, `result`, `alert`) is self-contained with its own handler, service, repository, and types
- **Dependency injection via Container** — a single `Container` struct in `internals/app` wires all dependencies, making it trivial to trace the dependency graph and modify it
- **Custom structured error handling** — the `apperror` package provides `Kind`, `Op`, and wrapped errors that flow cleanly from repository → service → handler → HTTP response
- **Idiomatic Go patterns** — interfaces defined by consumers (not producers), concrete types by default, `context.Context` threaded everywhere, explicit error returns

---

## Architecture

### High-Level Architecture

```mermaid
graph TB
    subgraph "External"
        CLIENT["Client / Browser"]
        TARGETS["Monitored Websites"]
    end

    subgraph "API Layer"
        ROUTER["Chi Router"]
        MW["Middleware Stack<br/>(Logger, RequestID, Recoverer, Timeout)"]
        AUTH["Auth Middleware<br/>(JWT + UUID parsing)"]
        UH["User Handler<br/>(Register, Login, Profile)"]
        MH["Monitor Handler<br/>(CRUD, Status)"]
    end

    subgraph "Background Pipeline"
        SCH["Scheduler<br/>(ticker + Lua scripts)"]
        REC["Reclaimer<br/>(ticker + Lua script)"]
        EXEC["Executor<br/>(100 worker goroutines)"]
        SEM["HTTP Semaphore<br/>(5000 concurrent slots)"]
        ROUTER_RP["Result Router"]
        SW["Success Workers"]
        FW["Failure Workers"]
        ALERT["Alert Service<br/>(worker pool)"]
    end

    subgraph "Infrastructure"
        REDIS[("Redis<br/>(Sorted Sets, Hashes, Cache)")]
        PG[("PostgreSQL<br/>(Users, Monitors, Incidents)")]
    end

    CLIENT --> ROUTER
    ROUTER --> MW --> AUTH
    AUTH --> UH
    AUTH --> MH
    UH --> PG
    MH --> PG
    MH -->|"cache monitor data"| REDIS

    SCH -->|"Lua: fetch + move to inflight"| REDIS
    SCH -->|"jobChan"| EXEC
    REC -->|"Lua: reclaim expired inflight"| REDIS

    EXEC -->|"load monitor (cache hit)"| REDIS
    EXEC -->|"load monitor (cache miss)"| PG
    EXEC --> SEM
    SEM -->|"HTTP GET"| TARGETS
    EXEC -->|"resultChan"| ROUTER_RP

    ROUTER_RP -->|"successChan"| SW
    ROUTER_RP -->|"failureChan"| FW

    SW -->|"ack, store status, clear incident"| REDIS
    SW -->|"schedule next run"| REDIS
    SW -->|"close DB incident"| PG

    FW -->|"retry / incident state"| REDIS
    FW -->|"create incident record"| PG
    FW -->|"alertChan"| ALERT
    FW -->|"schedule next run"| REDIS
```

### Component Interaction

```mermaid
sequenceDiagram
    participant S as Scheduler
    participant R as Redis
    participant E as Executor
    participant T as Target URL
    participant RP as Result Processor
    participant DB as PostgreSQL
    participant A as Alert Service

    loop Every tick (1-2s)
        S->>R: Lua: ZRANGEBYSCORE + ZREM + ZADD inflight
        R-->>S: Due monitor IDs
        S->>E: jobChan <- JobPayload
    end

    E->>R: GetMonitor (cache lookup)
    R-->>E: Cached monitor data
    E->>T: HTTP GET (health check)
    T-->>E: HTTP Response
    E->>RP: resultChan <- HTTPResult

    alt Success
        RP->>R: AckJob, StoreStatus, ClearIncident
        RP->>R: Schedule next run
    else Failure (retryable)
        RP->>R: IncrementRetry
        RP->>R: Schedule retry (5s)
    else Failure (threshold exceeded)
        RP->>R: IncrementIncident
        RP->>DB: Create incident record
        RP->>A: alertChan <- AlertEvent
    end
```

---

## Core Pipeline

The system uses a **channel-based pipeline** that connects five independent stages. Each stage runs as a pool of goroutines, communicating exclusively via Go channels.

```
┌───────────┐    jobChan     ┌──────────┐   resultChan   ┌─────────────────┐   alertChan   ┌──────────────┐
│ Scheduler │ ──────────────>│ Executor │ ──────────────>│ Result Processor│ ─────────────>│ Alert Service│
└───────────┘                └──────────┘                └─────────────────┘              └──────────────┘
     │                            │                           │        │
     │                            │                           │        │
   Redis                     Redis+HTTP                    Redis     PostgreSQL
 (sorted set)               (cache+check)             (state machine)  (incidents)
```

### Stage 1: Scheduler

**Responsibility**: Pull due monitoring jobs from Redis and dispatch them to the executor.

- Runs on a configurable tick interval (typically 1-2 seconds)
- Uses **Lua scripts** to atomically fetch due jobs and move them to an inflight set
- Implements **backpressure protection**: if `jobChan` is full, jobs are rescheduled with jitter instead of being dropped
- Adds random jitter to prevent thundering herd on reschedule

### Stage 2: Executor

**Responsibility**: Load monitor config, execute HTTP health checks, and emit results.

- **Worker Pool**: `N` goroutines (configurable, e.g., 100) read from `jobChan`
- **HTTP Semaphore**: A separate `chan struct{}` of size `M` (e.g., 5000) limits concurrent HTTP connections, preventing file descriptor exhaustion
- **Two-tier concurrency**: Workers acquire from `jobChan`, then spawn a goroutine per HTTP check that must acquire the semaphore
- **Error Classification**: Distinguishes DNS failures (terminal), timeouts (retryable), and network errors (retryable) for downstream routing
- **Monitor Caching**: Loads monitor config from Redis cache first, falling back to PostgreSQL — eliminating DB reads on the hot path

```go
// Executor concurrency model
for job := range jobChan {           // N workers compete for jobs
    monitor := loadFromCacheOrDB()   // Redis first, DB fallback
    httpSem <- struct{}{}            // Acquire HTTP slot (blocks if M in-flight)
    go func() {
        defer func() { <-httpSem }() // Release HTTP slot
        result := executeHTTPCheck(monitor)
        resultChan <- result
    }()
}
```

### Stage 3: Result Processor

**Responsibility**: Route results, manage retry/incident state machines, trigger alerts.

- **Router goroutine** reads from `resultChan` and fans out to `successChan` or `failureChan`
- **Success workers** clear retry/incident state and schedule the next check
- **Failure workers** implement a multi-stage decision tree:

```mermaid
flowchart TD
    F["Failure Received"] --> T{"Terminal?<br/>(INVALID_REQUEST,<br/>DNS_FAILURE)"}
    T -->|Yes| STOP["Store status<br/>Do NOT reschedule"]
    T -->|No| R{"Retryable?"}
    R -->|Yes| RC{"Retry count<br/><= threshold?"}
    RC -->|Yes| RETRY["Schedule retry<br/>(5s interval)"]
    RC -->|No| CLEAR["Clear retry state"]
    CLEAR --> INC["Increment incident"]
    R -->|No| INC
    INC --> TH{"Fail count<br/>>= threshold?"}
    TH -->|No| RESCHED["Reschedule<br/>(normal interval)"]
    TH -->|Yes| ALERT_CHECK{"Already<br/>alerted?"}
    ALERT_CHECK -->|Yes| RESCHED
    ALERT_CHECK -->|No| CREATE["Create DB incident<br/>+ Send alert"]
    CREATE --> RESCHED
```

### Stage 4: Alert Service

**Responsibility**: Process alert events from the alert channel using a worker pool.

### Stage 5: Reclaimer (Independent)

**Responsibility**: Recover jobs stuck in the inflight set (crashed/slow workers).

- Runs on its own ticker (every 5-10 seconds)
- Uses a Lua script to atomically move expired inflight jobs back to the schedule set
- Ensures **zero job loss** even if executor workers crash

---

## Distributed Scheduling

### The Problem

In a distributed system with multiple instances, how do you ensure:
1. Each monitoring job runs **exactly once** per interval?
2. No jobs are **lost** if a worker crashes mid-execution?
3. Scheduling remains **atomic** even under high concurrency?

### The Solution: Dual Sorted Sets + Lua Scripts

Redis sorted sets (`ZSET`) are used as a priority queue where:
- **Score** = Unix timestamp (milliseconds) of when the job should run
- **Member** = Monitor UUID

Two sorted sets work together:

| Set | Purpose |
|---|---|
| `monitor:schedule` | Jobs waiting to be executed. Score = next run time |
| `monitor:inflight` | Jobs currently being processed. Score = visibility timeout deadline |

### Three Lua Scripts

All scheduling operations are implemented as **Lua scripts** executed atomically on Redis, eliminating race conditions across multiple instances.

#### 1. `fetchAndMoveToInflight` — Atomic Job Dispatch

```lua
-- Atomically: fetch due jobs AND move them to inflight in one operation
local items = redis.call("ZRANGEBYSCORE", scheduleKey, "-inf", now, "LIMIT", 0, limit)
for i, member in ipairs(items) do
    redis.call("ZREM", scheduleKey, member)
    redis.call("ZADD", inflightKey, now + visibilityTimeout, member)
end
return items
```

**Why this matters**: Without atomicity, two scheduler instances could both fetch the same job. The Lua script guarantees that fetch + remove + add-to-inflight happens as a single Redis operation — no locks, no races.

#### 2. `reclaimMonitors` — Crash Recovery

```lua
-- Move expired inflight jobs (visibility timeout exceeded) back to schedule
local items = redis.call("ZRANGEBYSCORE", inflightKey, "-inf", now, "LIMIT", 0, limit)
for i, member in ipairs(items) do
    redis.call("ZREM", inflightKey, member)
    redis.call("ZADD", scheduleKey, now, member)
end
return #items
```

**Why this matters**: If a worker takes a job but crashes before acknowledging it, the visibility timeout expires and the Reclaimer automatically moves it back for re-execution.

#### 3. `fetchDueMonitors` — Simple Fetch (Non-Reliable Mode)

```lua
-- Fetch and remove due jobs (for benchmarking against reliable mode)
local items = redis.call("ZRANGEBYSCORE", key, "-inf", now, "LIMIT", 0, limit)
for i, member in ipairs(items) do
    redis.call("ZREM", key, member)
end
return items
```

### Job Lifecycle

```mermaid
stateDiagram-v2

[*] --> Scheduled: Monitor created/rescheduled

Scheduled --> Inflight: Lua script<br/>(atomic fetch+move)

Inflight --> Processing: Worker picks up job

Processing --> Acknowledged: Result processed

Acknowledged --> Scheduled: Next run scheduled

Inflight --> Scheduled: Visibility timeout expired<br/>(Reclaimer)

Processing --> Inflight: Worker crash<br/>(job stays in inflight)
```

---

## Reliability & Fault Tolerance

### 1. Zero Job Loss Guarantee

Every job that enters the system will eventually be processed:

- **Inflight visibility timeout**: Jobs in the inflight set have a deadline. If not acknowledged in time, the Reclaimer moves them back to the schedule set
- **Backpressure protection**: If `jobChan` is full, the scheduler reschedules the job with a 2-second backoff + jitter instead of dropping it
- **Graceful shutdown**: Channels are closed in strict dependency order, ensuring every in-flight message is drained

### 2. Graceful Shutdown

The shutdown sequence is carefully ordered to prevent data loss:

```
1. close(jobChan)           ← Scheduler stops producing
2. executor.Stop()          ← Wait for all workers + HTTP goroutines to finish
3. close(resultChan)        ← Executor output is drained
4. resultProcessor.Wait()   ← Wait for all success/failure workers
5. close(alertChan)         ← Result processor output is drained
6. alertService.Wait()      ← Wait for all alert workers
7. redis.Close()            ← Infrastructure cleanup
```

Each `close()` triggers the downstream `for range` loop to exit, ensuring every message in every channel is processed before shutdown completes.

### 3. Resilient Error Handling

- **Custom `apperror` package** with `Kind`, `Op`, and stack traces for structured error classification
- **`WrapRepoError`** utility function consistently applied across all repositories, mapping `pgx.ErrNoRows` → `apperror.NotFound` and other DB errors → `apperror.Internal`
- **Retry helper** with exponential backoff (50ms, 100ms, 150ms) for all Redis operations.

### 4. Incident State Machine

Redis hashes track incident state per monitor with the following fields:

```
monitor:incident:<uuid>
├── failure_count: int        ← Incremented on each failure
├── first_failure_at: unix_ts ← Set on first failure
├── last_failure_at: unix_ts  ← Updated on each failure
├── alerted: bool             ← Set atomically via HSETNX (prevents duplicate alerts)
└── db_incident: bool         ← Tracks if DB incident record was created
```

The `MarkIncidentAlertedIfNotSet` method uses Redis `HSETNX` for **atomic alert deduplication** — even with multiple workers processing failures for the same monitor, only one will trigger the alert.

---

## Performance Optimizations

### 1. Redis Cache on Hot Path (Eliminating DB Reads)

The executor's hot path (called for every health check) loads monitor configuration. Without caching, this would be a PostgreSQL query per check — at 1 Million checks/min, that's ~16,666 QPS just for monitor lookups.

```
Hot path WITHOUT cache:    Executor → PostgreSQL → Execute HTTP check
Hot path WITH cache:       Executor → Redis (sub-ms) → Execute HTTP check
                                       ↓ (cache miss only)
                                    PostgreSQL
```

Monitor data is cached in Redis for 24 hours.

### 2. HTTP Semaphore (Bounded Concurrency)

Instead of spawning unbounded goroutines for HTTP checks, a semaphore channel limits concurrent connections:

```go
httpSem := make(chan struct{}, 5000) // Max 5000 concurrent HTTP requests

// Before HTTP check:
httpSem <- struct{}{}  // Block if 5000 already in-flight

// After HTTP check:
<-httpSem  // Release slot
```

This prevents file descriptor exhaustion and allows fine-tuning of network pressure.

### 3. O(log N) Scheduling with Sorted Sets

Redis sorted sets provide `ZRANGEBYSCORE` in O(log N + M) where M is the result count. For 1M monitors, finding all due jobs is logarithmic — compared to scanning a database table which would be O(N).

### 4. Batch Operations

- `ScheduleBatch()` uses `ZADD` with multiple members in a single Redis call
- Lua scripts process multiple jobs atomically in a single Redis roundtrip
- `FetchAndMoveToInflight` fetches up to `batchSize` (configurable: 100-1000) jobs per tick

### 5. Connection Pool Tuning

Both Redis and PostgreSQL connection pools are configurable:

```yaml
redis:
  pool_size: 10          # Concurrent Redis connections
  min_idle_conns: 5      # Pre-warmed connections
  conn_max_lifetime: 2m  # Prevent stale connections

db:
  max_open_conns: 50     # Max PostgreSQL connections
  min_idle_conns: 5
  conn_max_lifetime: 1h
```

---

## Scalability Analysis

### How This Handles 1 Million Monitors

| Component | Scaling Strategy | Capacity |
|---|---|---|
| **Scheduler** | Lua script fetches up to 1000 jobs/tick. At 1s tick = 1000 jobs/sec = 3.6M/hour | Exceeds 1M easily |
| **Executor Workers** | 100 workers × 50 jobs/sec/worker = 5000 jobs/sec | CPU-bound scaling |
| **HTTP Semaphore** | 5000 concurrent HTTP connections | Network-bound scaling |
| **Redis** | Single Redis handles 100K+ ops/sec; sorted sets are O(log N) | Handles millions |
| **Result Processor** | Separate success/failure worker pools with configurable counts | Independent scaling |
| **Channels** | Buffered channels (configurable 100-5000) act as shock absorbers | Handles burst traffic |

### Horizontal Scaling

Multiple instances of this service can run simultaneously because:
1. **Lua scripts are atomic** — no two instances can grab the same job
2. **Inflight tracking** — each instance only processes what it fetched
3. **Reclaimer** — if any instance fails, another will recover its jobs

---

## Engineering Challenges

### **Challenge 1: Making Scheduling Atomic Across Multiple Instances**

**Problem**: When multiple instances of the scheduler run concurrently, how do you prevent two instances from fetching the same job?

**Failed approaches considered**:
- Distributed locks (Redis SETNX) → too much overhead per job
- Database-based scheduling → too slow for high throughput

**Solution :** <br/>
Redis Lua scripts execute atomically on the Redis server. The `fetchAndMoveToInflight` script combines `ZRANGEBYSCORE` + `ZREM` + `ZADD` into a single atomic operation. Redis guarantees no other command runs between these operations, eliminating race conditions without external coordination.

### **Challenge 2: Managing Hundreds of Goroutines Without Race Conditions**

**Problem**: With 100+ executor workers, success/failure workers, alert workers, the scheduler, and the reclaimer all running concurrently, how do you prevent data races and ensure clean shutdown?

**Solution :**
- **Channels as the sole communication mechanism** — no shared mutable state between pipeline stages
- **`sync.WaitGroup`** for coordinating goroutine lifecycle within each stage
- **Semaphore pattern** (`chan struct{}`) to bound HTTP concurrency without mutexes
- **Context cancellation** propagated to all goroutines for coordinated shutdown
- **Ordered channel closure** ensuring every message is drained before shutdown (see Graceful Shutdown section above)

### **Challenge 3: Optimizing the Hot Path**

**Problem**: Every monitoring check requires 

- Loading monitor configuration. 
- Storing monitor check result.

At scale, this means thousands of database queries per second , and it will blast the DB and reduce performance.

**Solution :** <br/>
Using Redis as hot storage which stores 
- Monitor configrations with 24-hour TTL 
- Real-time monitor status
- Ongoing incidents

This reduce the read & write latency and enhance the performance and throughtput of whole pipeline by huge margin. 

---

## Why Go
I love this fucking language 😎

Lets put my love aside and talk practically so,
Go was chosen for this project for specific technical reasons aligned with the system's requirements:

### 2. Nature of System — IO Bound Workload

This system is mainly doing IO Bound tasks, like making HTTP/HTTPs calls, accessing Redis and Database, redis based scheduling, sending alerts. All these are IO operations, and very minimal CPU operations. and who can handle IO better than Golang.

### 1. Goroutines — Lightweight Concurrency at Scale

Each goroutine uses only ~2-8KB of stack (vs. ~1MB per OS thread). Running 100 executor workers + 5000 concurrent HTTP goroutines + background workers costs less than 50MB of memory. The Go scheduler efficiently multiplexes these onto a small number of OS threads.

### 2. Channels — CSP Model for Pipeline Architecture

Go's channel primitive is the exact abstraction needed for this pipeline architecture. `for range ch` blocks until the channel is closed, `select` enables non-blocking operations with timeouts, and `close()` propagates shutdown signals downstream. This was used for the entire `Scheduler → Executor → ResultProcessor → AlertService` pipeline, with zero shared state between stages.

### 3. Static Binary + Fast Startup

The multi-stage Dockerfile produces a **~10MB static binary** running on `distroless`. There's no runtime (JVM, V8, etc.) to warm up — the service starts in milliseconds, which is critical for container orchestration (Kubernetes liveness probes, rolling deployments).


### 4. Strong Standard Library

`net/http`, `encoding/json`, `context`, `sync`, `errors` — Go's standard library covers HTTP clients, JSON serialization, synchronization primitives, and error handling without external dependencies. The entire HTTP executor is built on the standard `http.Client` with no additional frameworks.

### 6. Explicit Error Handling

This is one of the best thing, I love about Go
It's explicit `if err != nil` pattern forces handling every failure path. In a monitoring system where reliability is paramount, this is an advantage over exception-based languages where errors can silently propagate.

**I can go on writing the advantages of Go and my love for this language.
Its simiplicity, maintainablity, low-footprint, in-built and simpler concurrency model. 
Go makes it easier to architecture complex systems like this.**

---

## Code Quality & Design

### Design Principles Applied

#### SOLID Principles

| Principle | Implementation |
|---|---|
| **Single Responsibility** | Each module owns one domain: `user` handles auth, `monitor` handles CRUD+caching, `scheduler` handles job dispatch, `executor` handles HTTP checks, `result` handles outcome processing |
| **Open/Closed** | The `Cache` interface allows swapping Redis for any cache backend. `MetricsRecorder` interface allows plugging in Prometheus, StatsD, etc. |
| **Liskov Substitution** | All interfaces (`Cache`, `MonitorService`, `UserService`) are defined by consumers, not producers (Go idiom) |
| **Interface Segregation** | `MonitorService` in executor only requires `LoadMonitor` + `ScheduleMonitor` — not the full service |
| **Dependency Inversion** | High-level modules depend on abstractions. Executor depends on `MonitorService` interface, not `*monitor.Service` |

#### Go Interface Philosophy

Interfaces are discovered, not designed upfront. Make interfaces when you really need them.

```go
// Defined by the CONSUMER (executor), not the producer (monitor service)
type MonitorService interface {
    LoadMonitor(context.Context, uuid.UUID) (monitor.Monitor, error)
    ScheduleMonitor(context.Context, uuid.UUID, int32, string)
}
```

Concrete types are used everywhere else — no unnecessary abstraction for single-consumer dependencies.

#### DRY — `WrapRepoError` Pattern

All repository error handling is consolidated into a single utility:

```go
func WrapRepoError(op string, err error, logger *zerolog.Logger, isNotFoundErrPossible bool) error
```

This function:
- Maps `pgx.ErrNoRows` → `apperror.NotFound` (when `isNotFoundErrPossible` is true)
- Maps all other DB errors → `apperror.Internal`
- Logs the error with operation context
- Returns a structured `apperror.Error` with `Kind`, `Op`, and `Message`

This reduced repository code by **~40%** while making error handling more consistent.

#### KISS — No Unnecessary Abstraction

- No ORM — `sqlc` generates type-safe Go code directly from SQL queries
- No dependency injection framework — a simple `Container` struct wires everything
- No message broker — Go channels provide exactly the right abstraction for in-process queues
- No distributed lock library — Lua scripts provide atomicity without external coordination

### Module Boundaries

| Module | Layer | Responsibility |
|---|---|---|
| `cmd/api` | Entry point | Bootstraps the application, loads config, initializes all dependencies, starts background workers, runs the HTTP server, and handles OS signal-based graceful shutdown |
| `config` | Configuration | Loads YAML config via Viper with environment variable overrides, sets sensible defaults, and validates all fields using struct tags before the app starts |
| `internals/app` | Orchestration | Wires all dependencies together in a `Container` struct (dependency injection), registers Chi routes with middleware, and implements ordered shutdown of channels and workers |
| `internals/middleware` | HTTP middleware | Provides authentication (JWT parsing + UUID context storage), authorization (role-based access control), structured request logging, request ID propagation, and pluggable metrics recording |
| `internals/security` | Security | Handles JWT token generation and validation (HS256), and password hashing/comparison using Argon2id |
| `internals/modules/user` | Domain — User | Manages user registration (with Argon2id hashing), login (with JWT issuance), and profile retrieval. Owns its own handler, service, repository, DTOs, and route definitions |
| `internals/modules/monitor` | Domain — Monitor | Handles monitor CRUD operations, Redis caching (with JSON marshal/unmarshal in the service layer), and scheduling of new monitors. Defines the `Cache` interface consumed by the service |
| `internals/modules/scheduler` | Domain — Scheduling | Runs a ticker-based loop that fetches due monitoring jobs from Redis using atomic Lua scripts and dispatches them to the executor via `jobChan`. Includes the Reclaimer for recovering stalled inflight jobs |
| `internals/modules/executor` | Domain — Execution | Runs a pool of worker goroutines that load monitor config (cache-first), execute HTTP health checks through an HTTP semaphore, classify errors (DNS, timeout, network), and emit results to `resultChan` |
| `internals/modules/result` | Domain — Result Processing | Routes results to success/failure worker pools. Success workers clear incidents and reschedule. Failure workers handle retries, increment incidents, create DB records, and trigger alerts via `alertChan` |
| `internals/modules/alert` | Domain — Alerting | Consumes alert events from `alertChan` using a worker pool and dispatches notifications (e.g., email) |
| `pkg/apperror` | Shared — Errors | Defines the structured `Error` type with `Kind` (NotFound, Internal, Unauthorised, etc.), `Op` (operation trace), and `Message`. Maps error kinds to HTTP status codes |
| `pkg/db` | Shared — Database | Manages the pgx connection pool initialization, and contains all sqlc-generated type-safe query functions for users, monitors, incidents, and alerts |
| `pkg/redisstore` | Shared — Redis | Encapsulates all Redis operations organized by domain: scheduling (sorted sets), monitor caching (`[]byte`), incident tracking (hashes), retry counters, status storage, and a generic retry helper with backoff |
| `pkg/httpclient` | Shared — HTTP | Provides a pre-configured `http.Client` with sensible timeouts and transport settings for outbound health checks |
| `pkg/logger` | Shared — Logging | Initializes zerolog with JSON output, log level configuration, and caller information |
| `pkg/utils` | Shared — Utilities | Contains `WrapRepoError` (centralized repo error handling), JSON response helpers, typed response message constants, and string converters |

### Error Handling Architecture

```go
// Custom error type with structured fields
type Error struct {
    Kind    Kind    // NotFound, Internal, Unauthorised, InvalidInput, Conflict
    Op      string  // "service.monitor.get_monitor"
    Message string  // User-facing message
    Err     error   // Wrapped underlying error
}
```

Errors flow upward through the stack: Repository → Service → Handler → HTTP response. At each layer, context is added via `Op`. The handler layer uses `utils.FromAppError()` to automatically map `Kind` to HTTP status codes.

---

## Project Structure

```
project-k/
├── cmd/api/
│   └── main.go                    # Entry point, signal handling, graceful shutdown
├── config/
│   ├── loader.go                  # Viper config loading with env override + validation
│   └── models.go                  # Config structs with validate tags
├── internals/
│   ├── app/
│   │   ├── container.go           # Dependency injection, ordered shutdown
│   │   └── router.go              # Chi route registration
│   ├── middleware/
│   │   ├── authentication.go      # JWT validation, uuid.UUID context storage
│   │   ├── authorization.go       # Role-based access control (extensible)
│   │   ├── logger.go              # Request logging
│   │   ├── metrics.go             # Pluggable metrics via MetricsRecorder interface
│   │   └── types.go               # Middleware type alias
│   ├── modules/
│   │   ├── user/
│   │   │   ├── domain.go          # User entity and CreateUserCmd
│   │   │   ├── dto.go             # RegisterRequest, LogInRequest, response types
│   │   │   ├── handler.go         # HTTP handlers: Register, LogIn, GetProfile
│   │   │   ├── repository.go      # PostgreSQL queries via sqlc
│   │   │   ├── routes.go          # Chi route definitions for /users
│   │   │   └── service.go         # Business logic: hashing, token generation, profile
│   │   ├── monitor/
│   │   │   ├── cache.go           # Cache interface (GetMonitor, SetMonitor, DelMonitor, etc.)
│   │   │   ├── domain.go          # Monitor entity and CreateMonitorCmd
│   │   │   ├── dto.go             # CreateMonitorRequest, UpdateMonitorRequest, responses
│   │   │   ├── handler.go         # HTTP handlers: Create, Get, List, UpdateStatus
│   │   │   ├── repository.go      # PostgreSQL queries via sqlc
│   │   │   ├── routes.go          # Chi route definitions for /monitors
│   │   │   └── service.go         # Business logic: CRUD, caching, scheduling helpers
│   │   ├── scheduler/
│   │   │   ├── lua_scripts.go     # 3 Lua scripts: fetchDue, fetchAndMoveToInflight, reclaim
│   │   │   ├── models.go          # JobPayload struct
│   │   │   ├── reclaimer.go       # Background ticker that reclaims stalled inflight jobs
│   │   │   └── scheduler.go       # Background ticker that dispatches due jobs to jobChan
│   │   ├── executor/
│   │   │   ├── executor.go        # Worker pool, HTTP semaphore, health check execution
│   │   │   └── models.go          # HTTPResult struct with zerolog marshaling
│   │   ├── result/
│   │   │   ├── processor.go       # Result router + worker pool lifecycle management
│   │   │   ├── success_worker.go  # Clears incidents, stores status, schedules next run
│   │   │   ├── failure_worker.go  # Retry logic, incident creation, alert triggering
│   │   │   ├── repository.go      # MonitorIncident PostgreSQL queries
│   │   │   └── types.go           # MonitorService interface for result processing
│   │   └── alert/
│   │       ├── models.go          # AlertEvent struct
│   │       └── service.go         # Alert worker pool, email dispatch
│   └── security/
│       ├── tokenizer.go           # JWT generation + validation (HS256)
│       ├── hasher.go              # Argon2id password hashing + comparison
│       └── types.go               # RequestClaims (JWT payload struct)
├── pkg/
│   ├── apperror/
│   │   ├── apperror.go            # Error struct with Kind, Op, Message, wrapped Err
│   │   ├── kind.go                # Error kinds: NotFound, Internal, Unauthorised, etc.
│   │   ├── http.go                # Maps error Kind → HTTP status code
│   │   └── code.go                # Reserved for future error codes
│   ├── db/
│   │   ├── dbConn.go              # pgx connection pool initialization + health check
│   │   ├── db.go                  # sqlc Queries struct
│   │   ├── models.go              # sqlc generated Go types for all tables
│   │   ├── users.sql.go           # sqlc generated: CreateUser, GetUserByID, etc.
│   │   ├── monitors.sql.go        # sqlc generated: CreateMonitor, GetMonitor, etc.
│   │   ├── monitor_incidents.sql.go  # sqlc generated: CreateIncident, CloseIncident
│   │   └── alerts.sql.go          # sqlc generated: alert queries
│   ├── redisstore/
│   │   ├── client.go              # Redis client initialization + connection config
│   │   ├── scheduler.go           # Schedule, PopDue, FetchAndMoveToInflight, AckJob
│   │   ├── reclaimer.go           # ReclaimMonitors (Lua script execution)
│   │   ├── monitor.go             # SetMonitor, GetMonitor, DelMonitor ([]byte cache)
│   │   ├── status.go              # StoreStatus, GetStatus, DelStatus
│   │   ├── incident.go            # IncrementIncident, ClearIncident, MarkAlerted, etc.
│   │   ├── retry_counter.go       # IncrementRetry, ClearRetry (with TTL)
│   │   ├── retry.go               # Generic retry helper with exponential backoff
│   │   └── schema.md              # Redis key schema documentation
│   ├── httpclient/
│   │   └── httpclient.go          # Pre-configured http.Client with timeouts
│   ├── logger/
│   │   └── logger.go              # Zerolog initialization with JSON output
│   └── utils/
│       ├── errorbuilder.go        # WrapRepoError: centralized repository error handling
│       ├── response.go            # WriteJSON, WriteError, FromAppError helpers
│       ├── constants.go           # Typed ResponseMessage constants
│       └── converters.go          # String/type conversion utilities
├── migration/                     # Goose SQL migrations
├── sqlc/                          # SQL query definitions for sqlc code generation
├── Dockerfile                     # Multi-stage build → distroless (~10MB image)
└── env.yaml                       # Configuration file (YAML)
```

---

## Tools and Packages 

I also try to minimize it.

| Layer | Technology | Rationale |
|---|---|---|
| **Language** | Go 1.24 | Goroutines, channels, static binary |
| **HTTP Router** | chi/v5 | Lightweight, middleware-friendly, stdlib compatible |
| **Database** | PostgreSQL + pgx/v5 | Robust RDBMS, connection pooling, prepared statements |
| **SQL Generation** | sqlc | Type-safe queries from SQL, no ORM overhead |
| **Cache + Scheduling** | Redis + go-redis/v9 | Sorted sets, Lua scripts, pub/sub ready |
| **Authentication** | JWT (golang-jwt/v5) | Stateless auth, HS256 signing |
| **Password Hashing** | Argon2id | OWASP recommended, timing-attack resistant |
| **Config** | Viper | YAML + env vars + validation |
| **Logging** | zerolog | Zero-allocation JSON logging |
| **Validation** | go-playground/validator | Struct tag validation |
| **Container** | Multi-stage Docker + distroless | ~10MB final image |

---

## Configuration

All system parameters are configurable via `env.yaml` with environment variable overrides:

```yaml
# ─── General ───────────────────────────────────────────
env: production                     # Environment: development | staging | production
service_name: monitor-service       # Service identifier for logging
port: 8080                          # HTTP server port

# ─── Authentication ───────────────────────────────────
auth:
  secret: "your-jwt-secret"         # HMAC-SHA256 signing key for JWT tokens
  token_ttl: 30m                    # Access token time-to-live

# ─── Pipeline Channels ───────────────────────────────
app:
  job_channel_size: 1000            # Buffer between Scheduler → Executor
  result_channel_size: 1000         # Buffer between Executor → Result Processor
  alert_channel_size: 500           # Buffer between Result Processor → Alert Service

# ─── Scheduler ────────────────────────────────────────
scheduler:
  interval: 1s                      # Ticker interval: how often to poll Redis for due jobs
  batch_size: 10                    # Max jobs fetched per tick via Lua script
  visibility_timeout: 30s           # Inflight expiry: after this, Reclaimer moves job back

# ─── Reclaimer ────────────────────────────────────────
reclaimer:
  interval: 5s                      # How often to scan for stalled inflight jobs
  limit: 10                         # Max jobs to reclaim per cycle

# ─── Executor ─────────────────────────────────────────
executor:
  worker_count: 100                 # Goroutines reading from jobChan
  http_semaphore_count: 500         # Max concurrent outbound HTTP connections

# ─── Alert Service ────────────────────────────────────
alert:
  worker_count: 50                  # Goroutines processing alert events
  owner_email: "you@example.com"    # Sender email for alert notifications
  access_key: "your-email-api-key"  # API key for email provider

# ─── Result Processor ────────────────────────────────
result_processor:
  success_worker_count: 50          # Goroutines handling successful check results
  success_channel_size: 100         # Buffer for successChan (router → success workers)
  failure_worker_count: 10          # Goroutines handling failed check results
  failure_channel_size: 50          # Buffer for failureChan (router → failure workers)

# ─── Redis ────────────────────────────────────────────
redis:
  url: "redis://localhost:6379"     # Redis connection URL
  dial_timeout: 5s                  # Timeout for establishing new connections
  read_timeout: 3s                  # Timeout for Redis read operations
  write_timeout: 3s                 # Timeout for Redis write operations
  pool_size: 20                     # Maximum number of connections in the pool
  min_idle_conns: 5                 # Pre-warmed idle connections kept ready
  conn_max_lifetime: 10m            # Max time a connection can be reused
  conn_max_idle_time: 5m            # Max time a connection can sit idle before closing

# ─── PostgreSQL ───────────────────────────────────────
db:
  url: "your-db-url"
  max_open_conns: 50                # Max connections in the pgx pool
  min_idle_conns: 5                 # Pre-warmed idle connections
  conn_max_lifetime: 1h             # Max time a connection can be reused
  conn_max_idle_time: 30m           # Max time a connection can sit idle
  health_timeout: 5s                # Timeout for the DB health ping on startup
```

---

## Database Schema

```mermaid
erDiagram
    users {
        uuid id PK
        text name
        text email UK
        text password_hash
        int monitors_count
        boolean is_paid_user
        timestamptz created_at
    }

    monitors {
        uuid id PK
        uuid user_id FK
        text url
        text alert_email
        int interval_sec
        int timeout_sec
        int latency_threshold_ms
        int expected_status
        boolean enabled
        timestamptz updated_at
        timestamptz created_at
    }

    monitor_incidents {
        uuid id PK
        uuid monitor_id FK
        timestamptz start_time
        timestamptz end_time
        boolean alerted
        int http_status
        int latency_ms
        timestamptz created_at
    }

    users ||--o{ monitors : "has many"
    monitors ||--o{ monitor_incidents : "has many"
```

---

## Getting Started

### Prerequisites

- Go 1.24+
- Goose ( to apply migrations)
- PostgreSQL 15+
- Redis 7+
- Docker (optional)

### Build Locally

```bash
# 1. Clone
git clone https://github.com/ashishDevv/monit.git && cd monit

# 2. Configure env 
cp env.yaml.example env.yaml  # Edit with your DB/Redis URLs

# 3. Run migrations - goose should be installed on your system
goose -dir migration postgres "your_connection_string" up

# 5. Run
go run cmd/api/main.go
```

### If you don't want headache, and docker is installed on your system

```bash
docker build -t monit .
docker run -p 8080:8080 --env-file .env monit
```

---

## API Reference

### Authentication

| Method | Endpoint | Description |
|---|---|---|
| `POST` | `/api/v1/users/register` | Register a new user |
| `POST` | `/api/v1/users/login` | Login, returns JWT |
| `GET` | `/api/v1/users/profile` | Get user profile (requires auth) |

### Monitors (all require authentication)

| Method | Endpoint | Description |
|---|---|---|
| `POST` | `/api/v1/monitors` | Create a monitor |
| `GET` | `/api/v1/monitors/:id` | Get a specific monitor |
| `GET` | `/api/v1/monitors?limit=10&offset=0` | List all monitors |
| `PATCH` | `/api/v1/monitors/:id` | Enable/disable a monitor |

---

*Built with Go, designed for scale, engineered for reliability.*
