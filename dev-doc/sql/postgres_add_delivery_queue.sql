CREATE TABLE public.delivery_queue
(
    id           BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    user_id      BIGINT    NOT NULL,
    payload_type TEXT      NOT NULL,
    payload      BYTEA     NOT NULL,
    status       TEXT      NOT NULL DEFAULT 'pending',
    created_at   TIMESTAMP NOT NULL DEFAULT now(),
    delivered_at TIMESTAMP
);

CREATE INDEX idx_delivery_queue_user_status ON delivery_queue (user_id, status);
CREATE INDEX idx_delivery_queue_status_create_at ON delivery_queue (status, created_at);
