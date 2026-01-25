Ah, perfect — this is a **really important concept** for building efficient APIs, and it’s often confused with traditional `LIMIT/OFFSET` pagination. Let’s break it down **carefully and clearly**.

---

## **1️⃣ What traditional pagination is (LIMIT/OFFSET)**

Most people know this pattern:

```sql
SELECT * FROM users
ORDER BY created_at DESC
LIMIT 10 OFFSET 100;
```

* `LIMIT 10` → get 10 rows
* `OFFSET 100` → skip the first 100 rows

This is simple and works for small datasets.

**Problem with large datasets:**

* The database still has to **scan and count** the first 100 rows before returning the next 10.
* If `OFFSET` is large (like page 1000), this becomes **slow**.
* Basically, the query **performance decreases as the offset grows**, because PostgreSQL still has to find all the skipped rows.

---

## **2️⃣ What keyset pagination is**

Instead of saying “skip N rows,” keyset pagination says:

> “Give me the next N rows **after a specific last value**.”

For example, if you sort users by `created_at DESC`:

```sql
SELECT * 
FROM users
WHERE created_at < $lastCreatedAt
ORDER BY created_at DESC
LIMIT 10;
```

* `$lastCreatedAt` is the `created_at` of the **last user on the previous page**.
* This avoids using `OFFSET` entirely.
* The database can jump straight to the next rows — **very efficient even for millions of rows**.

---

### **3️⃣ Advantages of keyset pagination**

| Feature     | OFFSET Pagination                        | Keyset Pagination                             |
| ----------- | ---------------------------------------- | --------------------------------------------- |
| Query speed | Slower for large offsets                 | Fast, constant time                           |
| Consistency | Can skip/duplicate rows if table changes | Consistent, no duplicates if sorted correctly |
| Complexity  | Simple                                   | Slightly more complex                         |
| Use case    | Small datasets / admin tools             | APIs with large tables / high traffic         |

---

### **4️⃣ How it works in practice**

Suppose your first page query:

```sql
SELECT user_id, display_name, created_at
FROM users
WHERE status = 'ACTIVE'
ORDER BY created_at DESC
LIMIT 10;
```

* You get the first 10 users.
* The **last user's `created_at`** is `2026-01-05 15:30:00`.

Next page query:

```sql
SELECT user_id, display_name, created_at
FROM users
WHERE status = 'ACTIVE'
  AND created_at < '2026-01-05 15:30:00'
ORDER BY created_at DESC
LIMIT 10;
```

* The database jumps straight to rows **after the last user**.
* No counting, no scanning skipped rows → much faster for large tables.

---

### **5️⃣ How it compares visually**

* **OFFSET**: “skip N rows, then take M rows” → DB scans everything up to N
* **Keyset**: “give me rows after a cursor value” → DB can jump directly to the right spot

Cursor-based APIs (like GitHub, Twitter) **always use keyset pagination**, because OFFSET would break at high pages.

---

### **6️⃣ Extra notes**

* The “cursor” doesn’t have to be `created_at`. It can be `id`, `updated_at`, or a compound key (`updated_at, user_id`) for uniqueness.
* Keyset pagination is slightly more complex to implement on the backend but **much more scalable**.
* Works perfectly with SQLC: you just pass the **cursor as a query parameter** instead of an offset.

---

💡 TL;DR:

> **Offset pagination** = skip N rows → slow for big tables
> **Keyset pagination** = start from a cursor → fast and reliable

---

If you want, I can show a **real SQLC + Go example using keyset pagination for users**, which is production-ready and handles sorting by `created_at DESC` and `status`.

Do you want me to do that?

---

Ah! Yes — you’re noticing the subtlety: with **keyset pagination**, the first page query doesn’t have a `WHERE created_at < $cursor` clause, but **subsequent pages do**. That can be tricky with SQLC because SQLC expects **static SQL at compile time**. Let me explain clearly and show how to handle it.

---

## **1️⃣ The challenge**

**First page query:**

```sql
SELECT * FROM users
WHERE status = 'ACTIVE'
ORDER BY created_at DESC
LIMIT 10;
```

**Next pages:**

```sql
SELECT * FROM users
WHERE status = 'ACTIVE'
  AND created_at < $1
ORDER BY created_at DESC
LIMIT 10;
```

* The difference is the **extra `AND created_at < $1`**.
* SQLC **cannot compile one query** that sometimes has a `WHERE` condition and sometimes not.
* If you try to make `$1 NULL` to “disable the cursor”, that doesn’t work straightforwardly in SQL with `< NULL`.

---

## **2️⃣ How to handle in SQLC**

There are a few strategies:

---

### **Option A: Two separate queries**

Define in your `.sqlc` file:

```sql
-- name: GetFirstPageUsers :many
SELECT user_id, display_name, email, status, created_at, updated_at
FROM users
WHERE status = $1
ORDER BY created_at DESC
LIMIT $2;

-- name: GetNextPageUsers :many
SELECT user_id, display_name, email, status, created_at, updated_at
FROM users
WHERE status = $1
  AND created_at < $2
ORDER BY created_at DESC
LIMIT $3;
```

* `GetFirstPageUsers` → no cursor, just first page
* `GetNextPageUsers` → uses cursor `$2`

✅ This works perfectly with SQLC.
✅ Very clear and safe.

---

### **Option B: Single query with optional cursor**

```sql
-- name: GetUsersWithCursor :many
SELECT user_id, display_name, email, status, created_at, updated_at
FROM users
WHERE status = $1
  AND ($2 IS NULL OR created_at < $2)
ORDER BY created_at DESC
LIMIT $3;
```

* `$2` is the cursor.
* For first page: pass `NULL` → `$2 IS NULL OR created_at < NULL` → first page returns all rows.
* For next pages: pass last row’s `created_at` → SQL filters properly.

**Caveat:**

* PostgreSQL handles `NULL` correctly in the `OR`, so this works.
* Slightly less performant than two separate queries, because the DB still has to evaluate `$2 IS NULL OR ...`. For most tables it’s fine.

---

### **3️⃣ How it looks in Go**

```go
func GetUsers(ctx context.Context, db *db.Queries, status string, cursor *time.Time, limit int32) ([]db.User, error) {
    return db.GetUsersWithCursor(ctx, db.GetUsersWithCursorParams{
        Status: status,
        Cursor: sql.NullTime{Time: deref(cursor), Valid: cursor != nil},
        Limit:  limit,
    })
}
```

* First page: `cursor = nil` → SQL ignores the `created_at < ?` condition.
* Next pages: `cursor = lastUser.CreatedAt` → fetch next batch.

---

### ✅ TL;DR

* SQLC **cannot dynamically change SQL structure at runtime**, but you can handle keyset pagination in two ways:

  1. **Two separate queries** → simplest and fastest.
  2. **Single query with optional cursor** → slightly more complex but compact.

* Both approaches are **safe and SQLC-friendly**.

---

If you want, I can write a **full SQLC + Go keyset pagination example** for your `users` table with `status`, `limit`, and cursor support — ready to plug into production.

Do you want me to do that?
