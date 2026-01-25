# 🧭 Roadmap (What we’ll do)

**Step 1️⃣** Understand how Goose works
**Step 2️⃣** Install Goose CLI
**Step 3️⃣** Create & understand migration files
**Step 4️⃣** Run migrations using Goose CLI
**Step 5️⃣** Verify migrations in the database
**Step 6️⃣** Run Goose programmatically from Go
**Step 7️⃣** Decide when to auto-migrate (best practices)

We’ll go **one step at a time**.

---

# 🧩 Step 1️⃣ — How Goose works (Concepts)

### Goose uses:

- **SQL files** (or Go files)
- **Versioned migrations**
- A table called **`goose_db_version`**

### Example migration filename:

```text
20240101120000_create_users_table.sql
```

- Timestamp ensures order
- Goose runs migrations **in ascending order**
- Once applied, Goose records it in the DB

📌 Goose will **never run the same migration twice**

---

## Migration file structure (SQL)

Every SQL migration has two parts:

```sql
-- +goose Up
-- SQL statements for applying the migration

-- +goose Down
-- SQL statements for rolling back
```

---

# 🛠 Step 2️⃣ — Install Goose CLI

### Run this **once**:

```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
```

Verify installation:

```bash
goose --version
```

If this fails:

- Ensure `$GOPATH/bin` is in your `PATH`

---

# 📂 Step 3️⃣ — Create migrations folder

From your project root:

```bash
mkdir migrations
```

Your project should now look like:

```text
.
├── config/
├── migrations/
├── main.go
├── go.mod
```

---

## Create your first migration

Goose provides a helper command:

```bash
goose -dir migrations create create_users_table sql
```

This creates something like:

```text
migrations/
└── 20260110123045_create_users_table.sql
```

---

### Open the file and edit it

```sql
-- +goose Up
CREATE TABLE users (
    id UUID PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    password TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE users;
```

📌 **Important**

- `Up` = apply
- `Down` = rollback

---

# ▶️ Step 4️⃣ — Run migrations using Goose CLI

### You need:

- DB running
- `DB_URL` (same one used by your app)

Example:

```bash
export DB_URL="postgres://user:password@localhost:5432/app_db?sslmode=disable"
```

Now run:

```bash
goose -dir migrations postgres "$DB_URL" up
```

### What happens:

- Goose connects to DB
- Creates `goose_db_version` table
- Applies all pending migrations

Expected output:

```text
OK   20260110123045_create_users_table.sql
```

---

## If something fails ❌

- Syntax error → fix SQL → re-run
- Partial migration → use `down` or fix manually

---

# 🔍 Step 5️⃣ — Verify in Database

Connect to Postgres:

```bash
psql "$DB_URL"
```

Check tables:

```sql
\dt
```

You should see:

```text
users
goose_db_version
```

Check applied migrations:

```sql
SELECT * FROM goose_db_version;
```

---

## Neon DB Problem

Good catch — this is a **very common issue with Neon Postgres**, and you didn’t do anything wrong 👍  
Let’s slow down and understand **exactly why this happens** and **how to fix it properly**.

---

# 🧠 What this error means

Error:

```
FATAL: prepared statement name is already in use (SQLSTATE 08P01)
```

### In simple terms

- **Neon uses a connection pooler** (PgBouncer-like)
  
- Goose (via `pgx`) uses **prepared statements**
  
- The pooler **reuses connections**
  
- Prepared statements are **connection-specific**
  
- Result → name collision → 💥 error
  

This is **not a Goose bug**  
This is **not your SQL’s fault**

It’s a **known Neon + pgx + prepared statements issue**

---

# 🧪 Why it happens specifically with Neon

Neon’s pooled endpoint:

```
...-pooler.ap-southeast-1.aws.neon.tech
```

This endpoint:

- Reuses connections aggressively
  
- Does **not allow prepared statements safely**
  
- Requires **simple protocol mode**
  

---

# ✅ The Correct Fix (Recommended by Neon)

You must **disable prepared statements** by adding this to your DB URL:

```
prefer_simple_protocol=true
```

---

# 🛠 Corrected command (THIS WILL WORK)

```bash
goose postgres "postgresql://neondb_owner:npg_XBlsthg0PCY7@ep-late-water-a1hlpg8x-pooler.ap-southeast-1.aws.neon.tech/userdb?sslmode=require&channel_binding=require&prefer_simple_protocol=true" up
```

📌 That **one flag** fixes everything.

---

# 🧩 Why this works

| Setting | Effect |
| --- | --- |
| `prefer_simple_protocol=true` | Disables prepared statements |
| Simple protocol | Safe with poolers |
| Goose migrations | Run cleanly |
| Neon | Fully compatible |

---

# 🔐 Important security note (PLEASE DO THIS)

You just pasted **real database credentials** 😬  
You should **rotate this password immediately** in Neon.

For learning: okay  
For safety: **rotate now**

---

# 🔄 Optional: Make this permanent (Recommended)

Since you already use Viper, update your config:

### `env.yaml`

```yaml
db_url: "postgresql://neondb_owner:REDACTED@ep-late-water-a1hlpg8x-pooler.ap-southeast-1.aws.neon.tech/userdb?sslmode=require&prefer_simple_protocol=true"
```

Now:

- Goose works
  
- Your app works
  
- No more surprises
  

---

Perfect 🎉  
Now we move to **Step 6️⃣ — Running Goose inside your Go application**, and we’ll do it **carefully**, **Neon-safe**, and **production-aware**.

We’ll go **slow and practical** again.

---

# 🧠 Step 6.1 — When should an app auto-migrate?

### ✅ Good for

- Local development
  
- Single-instance services
  
- Early-stage projects
  

### ⚠️ Be careful in

- Production
  
- Multiple replicas (Kubernetes, ECS)
  
- Shared databases
  

👉 **Rule of thumb**

> Auto-migrate only in `development` or when explicitly enabled.

We’ll enforce this rule in code.

---

# 📦 Step 6.2 — Add Goose dependency to your project

From project root:

```bash
go get github.com/pressly/goose/v3
```

This allows you to call Goose **programmatically**.

---

# 🔌 Step 6.3 — Open DB connection (Neon-safe)

Goose needs a `*sql.DB`.

### IMPORTANT (Neon + pgx)

We must disable prepared statements **again**.

---

### Create `internal/db/db.go` (or similar)

```go
package db

import (
    "database/sql"
    "log"

    _ "github.com/jackc/pgx/v5/stdlib"
)

func NewPostgres(dbURL string) *sql.DB {
    db, err := sql.Open("pgx", dbURL)
    if err != nil {
        log.Fatalf("failed to open db: %v", err)
    }

    if err := db.Ping(); err != nil {
        log.Fatalf("failed to ping db: %v", err)
    }

    return db
}
```

📌 Your `db_url` **must include**:

```
prefer_simple_protocol=true
```

---

# 🧩 Step 6.4 — Run Goose migrations in code

### Create `internal/db/migrate.go`

```go
package db

import (
    "database/sql"
    "log"

    "github.com/pressly/goose/v3"
)

func RunMigrations(db *sql.DB) {
    if err := goose.SetDialect("postgres"); err != nil {
        log.Fatalf("failed to set goose dialect: %v", err)
    }

    if err := goose.Up(db, "migrations"); err != nil {
        log.Fatalf("failed to run migrations: %v", err)
    }
}
```

---

# 🚀 Step 6.5 — Wire it into `main.go`

### Example `main.go`

```go
package main

import (
    "cosmic-user-service/config"
    "cosmic-user-service/internal/db"
    "log"
)

func main() {
    cfg := config.LoadConfig("env.yaml")

    dbConn := db.NewPostgres(cfg.DBURL)

    // ✅ Only auto-migrate in development
    if cfg.Env == "development" {
        log.Println("Running database migrations...")
        db.RunMigrations(dbConn)
    }

    log.Println("Starting server...")
    // start HTTP server here
}
```

---

# 🧪 Step 6.6 — Test it locally

1. Drop the table manually (optional):
  
  ```sql
  DROP TABLE users;
  ```
  
2. Run your app:
  
  ```bash
  go run main.go
  ```
  
3. Expected output:
  
  ```text
  Running database migrations...
  OK   20260110123045_create_users_table.sql
  Starting server...
  ```
  

---

# 🛑 VERY IMPORTANT — Production safety

### ❌ Do NOT do this blindly:

```go
goose.Up(db, "migrations")
```

### ✅ Better controls (choose one later):

- `cfg.Env == "development"`
  
- `ENABLE_MIGRATIONS=true`
  
- CI/CD pipeline
  
- Kubernetes init container
  

We’ll design this cleanly in the next step.

---

# 🧭 Step 7️⃣ — Production-grade migration strategy (NO foot-guns)

## The core problem we must solve

In production you often have:

- Multiple app replicas
  
- Auto-scaling
  
- Restarts
  
- Rolling deployments
  

❌ If **every instance runs `goose.Up()`**, you can get:

- Race conditions
  
- Deadlocks
  
- Failed deployments
  
- Corrupted state (worst case)
  

So we need **controlled migration execution**.

---

# ✅ The 4 safe strategies (from simplest to best)

I’ll explain all four, then tell you **which one you should use**.

---

## Strategy 1️⃣ — Manual CLI (Simple & Safe)

### How it works

You run migrations manually before deployment:

```bash
goose -dir migrations postgres "$DB_URL" up
```

### When to use

- Small teams
  
- Early stage
  
- You control deploys
  

### Pros

✅ Zero risk  
✅ Easy  
✅ Very common

### Cons

❌ Human step  
❌ Easy to forget

---

## Strategy 2️⃣ — CI/CD Pipeline (Recommended ⭐)

### How it works

Migrations run automatically in CI/CD **before app deploys**.

Example pipeline step:

```bash
goose -dir migrations postgres "$DB_URL" up
```

### When to use

- Production
  
- Staging
  
- Multiple replicas
  

### Pros

✅ Fully automated  
✅ No race conditions  
✅ Industry standard

### Cons

❌ Needs CI config

---

## Strategy 3️⃣ — App-controlled via ENV flag

### How it works

Only run migrations when explicitly enabled:

```env
RUN_MIGRATIONS=true
```

```go
if os.Getenv("RUN_MIGRATIONS") == "true" {
    db.RunMigrations(migrationDB)
}
```

### When to use

- Temporary setups
  
- One-off tasks
  

### Pros

✅ Simple

### Cons

❌ Dangerous if misused

---

## Strategy 4️⃣ — Kubernetes Init Container (Best for K8s)

### How it works

- A **single init container** runs migrations
  
- App containers start only after success
  

### Pros

✅ Rock solid  
✅ Zero race conditions

### Cons

❌ Kubernetes-only  
❌ More YAML

---

# 🎯 What YOU should use

Based on your setup (Go + Viper + Goose + Neon):

### ✅ **Use Strategy 2 (CI/CD) + Strategy 1 (local dev)**

| Environment | How migrations run |
| --- | --- |
| Local | App auto-migrate |
| CI  | `goose up` |
| Production | CI pipeline |
| App | ❌ no auto-migration |

---

# 🧩 Final recommended setup (clean & safe)

## 1️⃣ Disable auto-migration in production

```go
if cfg.Env == "development" {
    db.RunMigrations(migrationDB)
}
```

Nothing changes here.

---

## 2️⃣ Create a dedicated migration command (OPTIONAL but clean)

Create:

```text
cmd/migrate/main.go
```

```go
package main

import (
    "cosmic-user-service/config"
    "cosmic-user-service/internal/db"
)

func main() {
    cfg := config.LoadConfig("env.yaml")
    migrationDB := db.NewMigrationDB(cfg.DBURL)
    db.RunMigrations(migrationDB)
}
```

Run it like:

```bash
go run cmd/migrate/main.go
```

This is **very CI-friendly**.

---

## 3️⃣ CI/CD example (generic)

```yaml
- name: Run DB migrations
  run: goose -dir migrations postgres "$DB_URL" up
```

or

```yaml
- name: Run DB migrations
  run: go run cmd/migrate/main.go
```

---

# 🔒 Extra safety (advanced, optional)

If you ever want **absolute safety**, Goose supports:

```sql
-- +goose StatementBegin
LOCK TABLE goose_db_version IN EXCLUSIVE MODE;
-- +goose StatementEnd
```

But in 99% of cases:

> **CI-controlled migrations are enough**