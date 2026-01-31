CREATE TABLE gateway_logs (
    id BIGSERIAL PRIMARY KEY,
    method TEXT,
    path TEXT,
    query TEXT,
    target_url TEXT,
    status_code INT,
    request_body TEXT,
    response_body TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_gateway_logs_created_at
ON gateway_logs(created_at DESC);

CREATE INDEX idx_gateway_logs_path
ON gateway_logs(path);
