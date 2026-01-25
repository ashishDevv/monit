monitors (
  id UUID PRIMARY KEY,
  user_id UUID,
  url TEXT,
  interval_seconds INT,
  timeout_seconds INT,
  latency_threshold_ms INT,
  expected_status INT,
  enabled BOOLEAN,
  created_at TIMESTAMP
)

check_results (
  id BIGSERIAL,
  monitor_id UUID,
  status INT,
  latency_ms INT,
  success BOOLEAN,
  checked_at TIMESTAMP
)

incidents (
  id UUID,
  monitor_id UUID,
  start_time TIMESTAMP,
  end_time TIMESTAMP NULL,
  success BOOLEAN,
  status INT,
  latency_ms INT
)

users (
  id UUID
  email TEXT UNIQUE
  password TEXT
  monitor_count INT   - max 10 can create
)