---- Types ----

CREATE TYPE device_platform AS ENUM ('android', 'ios', 'windows', 'macos', 'linux', 'unknown');

---- Tables ----

-- Devices
CREATE TABLE devices
(
    id         VARCHAR(127)    NOT NULL,
    platform   device_platform NOT NULL,
    name       VARCHAR(255)    NOT NULL,
    created_at TIMESTAMP       NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP       NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id)
);

-- Device user
CREATE TABLE device_user
(
    device_id VARCHAR(127) NOT NULL,
    user_id   BIGINT       NOT NULL,
    PRIMARY KEY (device_id, user_id)
);