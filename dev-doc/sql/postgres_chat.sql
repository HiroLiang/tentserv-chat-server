---- Types ----

CREATE TYPE chat_group_type AS ENUM ('direct', 'group', 'channel', 'bot');

CREATE TYPE participant_type AS ENUM ('user', 'agent', 'system');

CREATE TYPE chat_message_type AS ENUM ('text', 'image', 'file', 'system');

CREATE TYPE chat_member_role AS ENUM ('owner', 'admin', 'member', 'guest');

---- Tables ----

-- Chat groups
CREATE TABLE IF NOT EXISTS goat.public.chat_groups
(
    id          BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    name        TEXT,
    description TEXT,
    avatar_url  TEXT,
    type        chat_group_type NOT NULL DEFAULT 'group',
    max_members INTEGER         NOT NULL DEFAULT 2,
    is_deleted  BOOLEAN         NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMP       NOT NULL DEFAULT now(),
    updated_at  TIMESTAMP       NOT NULL DEFAULT now(),
    created_by  BIGINT          REFERENCES users (id) ON DELETE SET NULL
);

-- Participants of chat groups
CREATE TABLE IF NOT EXISTS goat.public.participants
(
    id           BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    type         participant_type NOT NULL,
    user_id      BIGINT REFERENCES users (id) ON DELETE CASCADE,
    agent_id     BIGINT REFERENCES agents (id) ON DELETE CASCADE,
    display_name TEXT,
    avatar_url   TEXT,
    created_at   TIMESTAMP        NOT NULL DEFAULT now(),

    -- NOTE: uniqueness is enforced by the partial indexes below (UNIQUE constraint
    -- with NULLs would not prevent duplicate USER/AGENT/SYSTEM rows)
    CONSTRAINT chk_participant CHECK (
        (type = 'user' AND user_id IS NOT NULL AND agent_id IS NULL) OR
        (type = 'agent' AND agent_id IS NOT NULL AND user_id IS NULL) OR
        (type = 'system' AND user_id IS NULL AND agent_id IS NULL)
        )
);

-- Add indexes for searching by participant with type
CREATE UNIQUE INDEX idx_participants_user ON participants (user_id) WHERE type = 'user';
CREATE UNIQUE INDEX idx_participants_agent ON participants (agent_id) WHERE type = 'agent';
CREATE UNIQUE INDEX idx_participants_system ON participants (type) WHERE type = 'system';

-- Chat group members
CREATE TABLE IF NOT EXISTS goat.public.chat_group_members
(
    id             BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    group_id       BIGINT           NOT NULL REFERENCES chat_groups (id) ON DELETE CASCADE,
    participant_id BIGINT           NOT NULL REFERENCES participants (id) ON DELETE CASCADE,
    role           chat_member_role NOT NULL DEFAULT 'member',
    joined_at      TIMESTAMP        NOT NULL DEFAULT now(),
    is_archived    BOOLEAN          NOT NULL DEFAULT FALSE,
    is_muted       BOOLEAN          NOT NULL DEFAULT FALSE,
    is_pinned      BOOLEAN          NOT NULL DEFAULT FALSE,
    last_read_at   TIMESTAMP,
    updated_at     TIMESTAMP        NOT NULL DEFAULT now(),

    UNIQUE (group_id, participant_id)
);

-- Add indexes for searching by participant
CREATE INDEX idx_chat_group_members_participant ON chat_group_members (participant_id);

-- Chat records
CREATE TABLE IF NOT EXISTS goat.public.chat_records
(
    id           BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    group_id     BIGINT            NOT NULL REFERENCES chat_groups (id) ON DELETE CASCADE,
    sender_id    BIGINT            NOT NULL REFERENCES participants (id) ON DELETE CASCADE,
    content      TEXT              NOT NULL,
    message_type chat_message_type NOT NULL DEFAULT 'text',
    reply_to_id  BIGINT            REFERENCES chat_records (id) ON DELETE SET NULL,
    is_edited    BOOLEAN           NOT NULL DEFAULT FALSE,
    is_deleted   BOOLEAN           NOT NULL DEFAULT FALSE,
    created_at   TIMESTAMP         NOT NULL DEFAULT now(),
    updated_at   TIMESTAMP         NOT NULL DEFAULT now()
);

CREATE INDEX idx_chat_records_group_id ON chat_records (group_id);
CREATE INDEX idx_chat_records_group_created ON chat_records (group_id, created_at DESC);
CREATE INDEX idx_chat_records_sender ON chat_records (sender_id);
