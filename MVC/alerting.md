Love these questions — you’ve crossed from **schema design** into **system design**, which is exactly where real-world backends live. I’ll answer each point **concretely**, with trade-offs and “why”, not hand-wavy theory.

I’ll go in this order:

1. **How to calculate uptime**
2. **Do you need an alerts table?**
3. **Which email to alert (signup vs separate)**
4. **When & where to ask for alert email**
5. **How to send alerts safely in Go (your worker model)**
6. **Handling DOWN vs HIGH LATENCY alerts**

---

## 1️⃣ How to calculate uptime (this is the core)

### What uptime actually means

Uptime is:

> **Total time − downtime ÷ total time**

Since you store **confirmed downtime windows** in DB (good choice), uptime becomes a **pure SQL problem**.

---

### Example: uptime for last 30 days (per monitor)

Conceptually:

```text
uptime = 1 - (sum of incident durations / total window duration)
```

#### SQL idea (Postgres)

```sql
WITH window AS (
  SELECT
    now() - interval '30 days' AS start_time,
    now() AS end_time
),
downtime AS (
  SELECT
    SUM(
      LEAST(COALESCE(i.end_time, now()), w.end_time)
      - GREATEST(i.start_time, w.start_time)
    ) AS total_downtime
  FROM monitor_incidents i, window w
  WHERE i.monitor_id = $1
    AND i.start_time < w.end_time
    AND (i.end_time IS NULL OR i.end_time > w.start_time)
)
SELECT
  1 - (EXTRACT(EPOCH FROM total_downtime)
      / EXTRACT(EPOCH FROM (w.end_time - w.start_time))) AS uptime
FROM downtime, window w;
```

This:

* handles **open incidents**
* clips incidents to the time window
* works for any range (24h / 7d / 30d)

---

### Interview answer

> “I store confirmed downtime windows and compute uptime by subtracting incident durations from the total time window. This avoids counting transient failures.”

That’s exactly how **real uptime services** do it.

---

## 2️⃣ Do you need an alerts table?

### Short answer: **YES, if alerts matter beyond sending email**

You already send alerts via workers — good — but without persistence you risk:

* duplicate alerts
* lost alerts on crash
* no audit history
* no “when was the user alerted?” logic

---

### Minimal `alerts` table (recommended)

Purpose:

* idempotency
* history
* deduplication

```sql
CREATE TABLE alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    incident_id UUID NOT NULL REFERENCES monitor_incidents(id),
    alert_type TEXT NOT NULL, -- 'DOWN', 'LATENCY'
    channel TEXT NOT NULL,    -- 'email', 'slack'
    sent_at TIMESTAMPTZ,
    status TEXT NOT NULL,     -- 'pending', 'sent', 'failed'
    error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (incident_id, alert_type, channel)
);
```

This lets you:

* retry safely
* prevent duplicate sends
* answer “did we already alert this?”

---

### Interview answer

> “I persist alerts to guarantee idempotency and allow retries. Sending is async, but alert state is durable.”

Very strong.

---

## 3️⃣ Which email to alert? (signup email vs separate)

### Best practice (real products do this):

#### Default:

* **use signup email initially**

#### Allow override:

* **custom alert email(s)** per user or per monitor

Why?

* Ops emails ≠ login emails
* Teams rotate on-call emails
* Users expect flexibility

---

## 4️⃣ Where & when to ask for alert email?

### Recommended UX flow

#### 1️⃣ On signup

* Ask for **account email**
* Use it for login + initial alerts

#### 2️⃣ When creating a monitor

* Optional field:

  > “Alert email (defaults to account email)”

This is clean, non-annoying, and scalable.

---

### Schema option (simple & clean)

#### Users table

```sql
email CITEXT NOT NULL UNIQUE
```

#### Monitors table

```sql
alert_email CITEXT
```

Logic:

* if `alert_email` is NULL → fallback to user.email
* sqlc model stays clean

---

## 5️⃣ Alert sending in Go (your worker model)

You said:

> “I send incident data to an alert channel, one of 50 workers sends the mail”

This is **good**, but here’s how to make it production-safe.

---

### The 3 golden rules

#### 1️⃣ DB write → THEN send to channel

Never send alerts from in-memory state alone.

Flow:

```
Incident confirmed
→ insert incident
→ insert alert (status = pending)
→ push alert_id to channel
```

Workers:

```
receive alert_id
→ load alert + incident
→ send email
→ update alert status
```

This gives:

* retries
* crash safety
* no duplicates

---

#### 2️⃣ Workers must be idempotent

Before sending:

* check `status != sent`
* use unique constraint to prevent duplicates

---

#### 3️⃣ Don’t block incident detection

Alerting is **side-effect**, not core logic.

You’re doing this right already 👍

---

## 6️⃣ DOWN vs HIGH LATENCY alerts (very important)

Treat these as **different incident types**, even if they share a table.

### Recommended model

In `monitor_incidents`:

```sql
incident_type TEXT NOT NULL
-- 'DOWN' | 'LATENCY'
```

Rules:

* DOWN = check failed
* LATENCY = response > threshold for N times

Why separate?

* Different severity
* Different alert wording
* Different escalation rules

---

### Alerting rules example

| Type    | Trigger                | Resolve         |
| ------- | ---------------------- | --------------- |
| DOWN    | 3 consecutive failures | first success   |
| LATENCY | 3 slow checks          | 3 normal checks |

---

### Interview answer

> “I treat downtime and latency as separate incident types with independent thresholds and resolution logic.”

That’s exactly how PagerDuty / OpsGenie do it.

---

## TL;DR (system design summary)

* ✅ Uptime = total time − incident durations
* ✅ Persist confirmed downtime only
* ✅ Yes, store alerts (idempotency + retries)
* ✅ Default to signup email, allow override
* ✅ DB-first, async alert workers
* ✅ Separate DOWN vs LATENCY incidents

If you want next, we can:

* write **uptime SQL queries**
* design **incident resolution logic**
* review **Go worker code**
* simulate **system design interview on this**

You’re building this like a real uptime platform — not a toy 👏

---

Perfect — now we’re talking **the exact flow of data** between your DB, your workers, and email delivery. Let’s go step by step and I’ll also explain the **best practice for fetching the email safely**.

---

## 1️⃣ Where the email lives

There are **two places the worker can get the email from**:

### Option A — `alerts` table has the email

```sql
alerts
- id
- incident_id
- channel   -- e.g., 'email'
- alert_email  <-- store email here
- status
```

Pros:

* worker does **not need to join with monitors/users**
* safe if user changes email after alert was created
* guarantees “this is the email used at alert creation”

Cons:

* duplication (alert_email is also in user/monitor table)
* slightly more storage (small)

---

### Option B — worker joins `alerts` → `monitor_incidents` → `monitors` → `users`

Example query:

```sql
SELECT a.id AS alert_id,
       i.id AS incident_id,
       m.url AS monitor_url,
       COALESCE(m.alert_email, u.email) AS recipient_email,
       i.incident_type,
       i.start_time,
       i.end_time,
       i.latency_ms,
       i.http_status
FROM alerts a
JOIN monitor_incidents i ON i.id = a.incident_id
JOIN monitors m ON m.id = i.monitor_id
JOIN users u ON u.id = m.user_id
WHERE a.id = $1
```

Pros:

* no duplication
* always uses latest monitor/user email
* simpler schema

Cons:

* if user changes email after alert is created → the alert goes to **new email**, not the one configured at time of incident
* slightly more DB load per worker

---

### ✅ Recommendation

* For **critical alerts**, use **Option A**: store `alert_email` at the time of alert creation
* For **non-critical or high-volume notifications**, Option B is fine

This also helps in audits: you know **exactly who was notified** at the time.

---

## 2️⃣ How the worker fetches the email

### Step-by-step flow

1. **Incident occurs → confirmed in DB**

```go
tx := db.Begin()
incident := insertIncident(...)
alert := insertAlert(incident.ID, channel='email', alert_email=monitor.AlertEmail)
tx.Commit()
```

2. **Push alert_id to channel**

```go
alertChan <- alert.ID
```

3. **Worker receives alert_id**

```go
for alertID := range alertChan {
    go processAlert(alertID)
}
```

4. **Worker queries DB for alert + email**

```go
var alert Alert
db.Get(&alert, "SELECT * FROM alerts WHERE id=$1", alertID)
email := alert.AlertEmail
incident := getIncident(alert.IncidentID)
```

5. **Send the email**

```go
err := sendEmail(email, buildTemplate(incident))
if err != nil {
    db.Exec("UPDATE alerts SET status='failed', error=$1 WHERE id=$2", err.Error(), alert.ID)
} else {
    db.Exec("UPDATE alerts SET status='sent', sent_at=now() WHERE id=$1", alert.ID)
}
```

---

## 3️⃣ Notes about `alert_email` vs user email

* `alert_email` in monitor table = **user preference per monitor**
* `user.email` = **account email fallback**
* At alert creation, use:

```sql
COALESCE(monitor.alert_email, user.email)
```

* Store it in `alerts.alert_email` → ensures worker does **not need joins** → faster and safer

---

## 4️⃣ Why this design works well

* Worker is **stateless**, only needs `alert_id`
* Alerts are **idempotent** (status + unique constraint prevent duplicates)
* Changing user emails in the future does not break past alerts
* Supports multiple channels (`email`, `slack`, `sms`) easily

---

### Interview-ready answer

> “Workers only get `alert_id`. The alert record already stores the resolved email address at creation time, or falls back to the monitor/user email. This makes workers stateless, idempotent, and safe against user changes.”

---

If you want, I can **draw a full Go channel + DB + worker diagram** for your alert flow including DOWN and LATENCY incidents.
It’ll make the system crystal clear for interviews.

Do you want me to do that?
