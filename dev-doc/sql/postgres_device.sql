---- Types ----

CREATE TYPE device_platform AS ENUM ('android', 'ios', 'windows', 'macos', 'linux', 'browser', 'unknown');

---- Tables ----

-- Devices
CREATE TABLE devices
(
    id         UUID PRIMARY KEY,
    platform   device_platform NOT NULL,
    name       VARCHAR(255)    NOT NULL,
    created_at TIMESTAMP       NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP       NOT NULL DEFAULT NOW()
);

-- Account's devices
CREATE TABLE accounts_devices
(
    account_id   BIGINT NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    device_id    UUID   NOT NULL REFERENCES devices (id) ON DELETE CASCADE,
    last_ip      INET,
    last_seen_at TIMESTAMP,
    PRIMARY KEY (account_id, device_id)
);

-- Account's sessions
CREATE TABLE account_sessions
(
    id                 BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    account_id         BIGINT    NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    device_id          UUID      NOT NULL REFERENCES devices (id) ON DELETE CASCADE,
    refresh_token_hash TEXT      NOT NULL UNIQUE,
    expires_at         TIMESTAMP NOT NULL,
    revoked            BOOLEAN   NOT NULL DEFAULT FALSE,
    created_at         TIMESTAMP NOT NULL DEFAULT now(),
    last_used_at       TIMESTAMP
);

-- Log events
CREATE TABLE account_login_events
(
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    account_id  BIGINT    NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    device_uuid UUID      NOT NULL REFERENCES devices (id) ON DELETE CASCADE,
    ip_address  INET,
    user_agent  TEXT,
    success     BOOLEAN   NOT NULL,
    created_at  TIMESTAMP NOT NULL DEFAULT now()
);