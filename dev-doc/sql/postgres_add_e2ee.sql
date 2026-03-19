-- E2EE Key Management Infrastructure
-- Run this migration after init_postgres.sql

-- User identity key (one per user per device, long-lived)
CREATE TABLE IF NOT EXISTS public.user_identity_keys
(
    id          BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    user_id     BIGINT    NOT NULL REFERENCES public.users (id) ON DELETE CASCADE,
    device_id   UUID      NOT NULL REFERENCES public.devices (id) ON DELETE CASCADE,
    public_key  BYTEA     NOT NULL, -- 32-byte Curve25519 public key
    fingerprint TEXT      NOT NULL, -- hex(SHA-256(public_key)) for display
    uploaded_at TIMESTAMP NOT NULL DEFAULT now(),
    UNIQUE (user_id, device_id)
);

-- Signed pre-key (rotated medium-term, signed by an identity key)
CREATE TABLE IF NOT EXISTS public.user_signed_pre_keys
(
    id         BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    user_id    BIGINT    NOT NULL REFERENCES public.users (id) ON DELETE CASCADE,
    device_id  UUID      NOT NULL REFERENCES public.devices (id) ON DELETE CASCADE,
    key_id     INT       NOT NULL, -- client-assigned uint32 identifier
    public_key BYTEA     NOT NULL, -- 32-byte Curve25519 public key
    signature  BYTEA     NOT NULL, -- 64-byte Ed25519 signature over public_key
    is_active  BOOLEAN   NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    expires_at TIMESTAMP,
    UNIQUE (user_id, device_id, key_id)
);
CREATE INDEX IF NOT EXISTS idx_user_signed_pre_keys_active
    ON public.user_signed_pre_keys (user_id, device_id, is_active);

-- One-time pre-keys (consumed atomically on X3DH initiation)
CREATE TABLE IF NOT EXISTS public.user_one_time_pre_keys
(
    id          BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    user_id     BIGINT    NOT NULL REFERENCES public.users (id) ON DELETE CASCADE,
    device_id   UUID      NOT NULL REFERENCES public.devices (id) ON DELETE CASCADE,
    key_id      INT       NOT NULL, -- client-assigned uint32 identifier
    public_key  BYTEA     NOT NULL, -- 32-byte Curve25519 public key
    uploaded_at TIMESTAMP NOT NULL DEFAULT now(),
    UNIQUE (user_id, device_id, key_id)
);
CREATE INDEX IF NOT EXISTS idx_user_one_time_pre_keys_lookup
    ON public.user_one_time_pre_keys (user_id, device_id);

-- Member sender key (per chat_member per room, for group E2EE)
-- distribution_message is an opaque sealed blob the server stores and
-- delivers to members who join later; server never parses it.
CREATE TABLE IF NOT EXISTS public.member_sender_keys
(
    id                   BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    chat_member_id       BIGINT    NOT NULL REFERENCES public.chat_members (id) ON DELETE CASCADE,
    chain_id             INT       NOT NULL DEFAULT 1, -- incremented on a re-key
    sender_key_public    BYTEA     NOT NULL,           -- public part for verification
    distribution_message BYTEA     NOT NULL,           -- sealed SenderKeyDistributionMessage
    created_at           TIMESTAMP NOT NULL DEFAULT now(),
    UNIQUE (chat_member_id, chain_id)
);
CREATE INDEX IF NOT EXISTS idx_member_sender_keys_member
    ON public.member_sender_keys (chat_member_id);
