---- Types ----

CREATE TYPE email_status AS ENUM ('sending', 'sent', 'failed');

---- Tables ----

-- Email Logs
CREATE TABLE IF NOT EXISTS goat.public.email_logs
(
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    sender      VARCHAR(255)  NOT NULL,
    recipients  TEXT          NOT NULL,
    subject     TEXT          NOT NULL,
    status      email_status  NOT NULL,
    external_id TEXT          NULL,
    created_at  TIMESTAMP     NOT NULL DEFAULT now()
);
