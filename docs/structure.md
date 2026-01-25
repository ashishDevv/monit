Below is a **clean, production-grade folder structure** commonly used for **event-driven Go microservices** that:

* Run an **HTTP server**
* **Publish** Kafka events
* **Subscribe** to Kafka events
* Start **all components together** when the app boots

This follows Go community conventions (`cmd`, `internal`, composition over globals) and scales well as the service grows.

---

## High-level principles

* **One entrypoint** → `cmd/service/main.go`
* **Business logic isolated** → `internal/domain`
* **Kafka + HTTP are adapters** → `internal/transport`
* **No globals** → everything wired in `main`
* **Graceful shutdown** → context + signals
* **Event-driven first-class citizen**

---

## Recommended Folder Structure

```
.
├── cmd/
│   └── userservice/
│       └── main.go
│
├── internal/
│   ├── app/                     # Application orchestration
│   │   └── app.go
│   │
│   ├── config/                  # Config loading
│   │   └── config.go
│   │
│   ├── domain/                  # Core business logic
│   │   ├── user/
│   │   │   ├── service.go
│   │   │   ├── model.go
│   │   │   └── events.go
│   │
│   ├── transport/
│   │   ├── http/
│   │   │   ├── server.go
│   │   │   └── handler.go
│   │   │
│   │   └── kafka/
│   │       ├── producer.go
│   │       ├── consumer.go
│   │       └── handler.go
│   │
│   ├── repository/              # DB or external storage
│   │   └── user_repository.go
│   │
│   └── pkg/                     # Shared internal utilities
│       ├── logger/
│       └── shutdown/
│
├── go.mod
└── go.sum
```

---

## What Each Layer Does

### `cmd/userservice/main.go`

**Only wiring + startup logic**

```go
func main() {
    cfg := config.Load()

    app := app.New(cfg)
    app.Run()
}
```

---

### `internal/app/app.go`

**Orchestrates HTTP + Kafka**

```go
type App struct {
    httpServer *http.Server
    consumer   *kafka.Consumer
}

func (a *App) Run() {
    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
    defer stop()

    go a.httpServer.Start()
    go a.consumer.Start(ctx)

    <-ctx.Done()
    a.Shutdown()
}
```

This ensures:

* HTTP server runs
* Kafka consumer runs
* Graceful shutdown on SIGTERM/SIGINT

---

### `internal/transport/http`

Handles REST / HTTP concerns only.

```go
type Server struct {
    router *chi.Mux
}

func (s *Server) Start() error {
    return http.ListenAndServe(":8080", s.router)
}
```

---

### `internal/transport/kafka`

#### Producer

```go
type Producer struct {
    writer *kafka.Writer
}

func (p *Producer) Publish(event Event) error {
    // serialize + publish
}
```

#### Consumer

```go
type Consumer struct {
    reader  *kafka.Reader
    handler Handler
}

func (c *Consumer) Start(ctx context.Context) {
    for {
        msg, _ := c.reader.ReadMessage(ctx)
        c.handler.Handle(msg)
    }
}
```

---

### `internal/domain`

**Pure business logic** (NO Kafka / HTTP imports)

```go
type UserService struct {
    repo      Repository
    publisher EventPublisher
}

func (s *UserService) CreateUser(u User) error {
    s.repo.Save(u)
    s.publisher.Publish(UserCreatedEvent{ID: u.ID})
    return nil
}
```

---

### `internal/domain/events.go`

```go
type UserCreatedEvent struct {
    ID string
}
```

---

## Dependency Direction (Important)

```
HTTP ─┐
      ├──> Application ───> Domain
Kafka ─┘
```

Domain **never knows**:

* Kafka
* HTTP
* Frameworks

---

## Startup Flow (What You Asked For)

When the service starts:

1. Config loads
2. Kafka producer connects
3. Kafka consumer starts in goroutine
4. HTTP server starts in goroutine
5. App blocks until shutdown signal
6. Graceful shutdown of all components

✔ **All run together**

---

## Kafka Topic Handling Strategy

**One consumer per bounded context**

```
user.created
user.updated
```

Handler mapping:

```go
switch event.Type {
case "user.created":
    handler.HandleUserCreated(event)
}
```

---

## Common Mistakes to Avoid

❌ Putting Kafka logic in `main.go`
❌ Mixing business logic in handlers
❌ Global Kafka producers
❌ One goroutine per message
❌ Domain importing Kafka clients

---

## Optional Enhancements

* `internal/observability/` (metrics, tracing)
* `internal/health/` (`/health`, `/ready`)
* `Makefile`
* `docker/`
* `configs/`

---

## TL;DR

**Use this structure if you want clean, scalable, idiomatic Go microservices with Kafka + HTTP.**

If you want, I can:

* Provide a **minimal runnable example**
* Show **graceful shutdown code**
* Recommend **Kafka Go libraries**
* Adapt this for **Clean Architecture or Hexagonal**

Just tell me.
