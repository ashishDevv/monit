# Multi - Consumer Guide

You’re now moving from **“one consumer” → “consumer system”**, which is a real production step.

I’ll explain this in **layers**, so you can *visualize it*, not just copy code.

---

# 1️⃣ First: what changes conceptually?

Right now you have:

```
RabbitMQ Connection
 └── Consumer (queue-1)
      └── workers
```

You want:

```
RabbitMQ Connection
 ├── Consumer-1 (queue-1)
 │    └── workers
 └── Consumer-2 (queue-2)
      └── workers
```

Key insight:

> **Consumers are independent, but the connection can be shared.**

---

# 2️⃣ Fundamental design rule (very important)

> **One queue = one consumer instance**

Do **not** try to make one `Consumer` read from multiple queues.

Instead:

* Same connection
* Different channels
* Different consumer instances

This matches RabbitMQ’s model **perfectly**.

---

# 3️⃣ Your current `Consumer` design is already correct ✅

Your `Consumer` struct:

```go
type Consumer struct {
	ch         *amqp091.Channel
	queueName string
	workers    int
	sem        chan struct{}
	wg         sync.WaitGroup
}
```

This is **exactly what we want**.

You don’t need to change it.

---

# 4️⃣ What needs to change? (High-level)

You need to:

1. Create **multiple consumers**
2. Start them **concurrently**
3. Shut them down **together**
4. Track them as a group

This logic belongs in:
👉 **`app` / `container` layer**, not inside RabbitMQ package.

---

# 5️⃣ Step 1: Create multiple consumers from the same connection

### `setupRabbitMQ` (updated idea)

```go
func setupRabbitMQ(cfg *config.RabbitMQConfig) (*amqp091.Connection, []*rabbitmq.Consumer, error) {
	conn, err := rabbitmq.NewConnection(cfg)
	if err != nil {
		return nil, nil, err
	}

	if err := rabbitmq.SetupTopology(conn, cfg); err != nil {
		return nil, nil, err
	}

	userConsumer, err := rabbitmq.NewConsumer(conn, cfg.UserQueue, cfg.UserWorkers)
	if err != nil {
		return nil, nil, err
	}

	paymentConsumer, err := rabbitmq.NewConsumer(conn, cfg.PaymentQueue, cfg.PaymentWorkers)
	if err != nil {
		return nil, nil, err
	}

	return conn, []*rabbitmq.Consumer{
		userConsumer,
		paymentConsumer,
	}, nil
}
```

Each call to `NewConsumer`:

* Opens a **new channel**
* Consumes from a **different queue**

---

# 6️⃣ Step 2: Store consumers as a slice (THIS IS IMPORTANT)

In `Container`:

```go
type Container struct {
	DB        *pgxpool.Pool
	Logger    *zerolog.Logger

	RMQConn   *amqp091.Connection
	Consumers []*rabbitmq.Consumer
}
```

Why slice?

* You don’t care *how many*
* You can iterate to start/shutdown
* Easy to extend later

---

# 7️⃣ Step 3: Start all consumers concurrently

### `internals/app/consumer.go`

```go
func StartConsumers(ctx context.Context, c *Container) error {
	for _, consumer := range c.Consumers {
		go func(cons *rabbitmq.Consumer) {
			handler := buildHandlerFor(cons) // explained below
			if err := cons.Consume(ctx, handler); err != nil {
				c.Logger.Error().Err(err).Msg("consumer stopped")
			}
		}(consumer)
	}
	return nil
}
```

Each consumer:

* Runs independently
* Shares shutdown context
* Stops when context is cancelled

---

# 8️⃣ Step 4: Different handlers per queue (CRITICAL DESIGN POINT)

Do **not** do this:

❌ One handler with `if queue == ...`

Instead:

> **Queue → handler mapping**

### Example

```go
userHandler := rabbitmq.NewUserEventHandler(c.userSvc)
paymentHandler := rabbitmq.NewPaymentEventHandler(c.paymentSvc)
```

Then wire explicitly:

```go
type ConsumerWithHandler struct {
	Consumer *rabbitmq.Consumer
	Handler  *rabbitmq.EventHandler
}
```

Or simpler: just keep the wiring in `app`.

This keeps **business logic clean**.

---

# 9️⃣ Step 5: Graceful shutdown for multiple consumers

Your shutdown logic now becomes trivial and powerful:

```go
func (c *Container) Shutdown(ctx context.Context) error {
	for _, consumer := range c.Consumers {
		_ = consumer.Shutdown(ctx)
	}

	if c.RMQConn != nil {
		_ = c.RMQConn.Close()
	}

	if c.DB != nil {
		c.DB.Close()
	}

	return nil
}
```

Shutdown order:

1. Stop all consumers
2. Wait for all handlers
3. Close connection
4. Close DB

Exactly what you wanted.

---

# 🔟 Mental model (THIS IS THE CLICK)

Think like this:

* **Connection** = highway
* **Channel** = lane
* **Consumer** = truck
* **Queue** = warehouse

One highway → many lanes → many trucks → different warehouses

---

# 1️⃣1️⃣ Scaling further (future-proof)

With this design, you can easily:

* Add 10 more queues
* Change worker count per queue
* Disable one consumer
* Run consumers in a separate binary
* Move consumers to another service

No refactor required.

---

# 1️⃣2️⃣ Common mistakes (avoid these)

❌ One consumer reading multiple queues
❌ One handler switching on queue name
❌ One WaitGroup for all consumers
❌ Sharing channels across consumers

You avoided all of these naturally 👏

---

# Final verdict

✅ Your current consumer design **already supports multiple queues**
✅ Just create multiple consumer instances
✅ Track them as a slice
✅ Start & stop them as a group

This is **exactly how real production systems do it**.

---

If you want next, we can:

* Add per-consumer readiness probes
* Add DLQ & retry strategy per queue
* Split consumers into a separate service
* Discuss connection-per-service vs per-consumer

Just tell me 👌
