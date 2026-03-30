---- Types ----

CREATE TYPE chat_room_type AS ENUM ('direct', 'group', 'channel', 'bot');

CREATE TYPE chat_member_role AS ENUM ('owner', 'admin', 'member', 'guest');

CREATE TYPE chat_message_type AS ENUM ('text', 'image', 'file', 'icon', 'system');

---- Tables ----

-- Chat rooms
CREATE TABLE IF NOT EXISTS public.chat_rooms
(
    id          BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    name        TEXT,
    description TEXT,
    avatar_name TEXT,
    type        chat_room_type NOT NULL DEFAULT 'direct',
    max_members INTEGER        NOT NULL DEFAULT 2,
    allow_agent BOOLEAN        NOT NULL DEFAULT false,
    is_deleted  BOOLEAN        NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ    NOT NULL DEFAULT now()
);

-- Chat members
CREATE TABLE IF NOT EXISTS public.chat_members
(
    id             BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    room_id        BIGINT           NOT NULL REFERENCES public.chat_rooms (id) ON DELETE NO ACTION,
    participant_id BIGINT           NOT NULL REFERENCES public.participants (id) ON DELETE NO ACTION,
    role           chat_member_role NOT NULL DEFAULT 'member',
    is_muted       BOOLEAN          NOT NULL DEFAULT false,
    is_deleted     BOOLEAN          NOT NULL DEFAULT false,
    last_read_at   TIMESTAMPTZ,
    joined_at      TIMESTAMPTZ      NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ      NOT NULL DEFAULT now(),
    deleted_at     TIMESTAMPTZ,

    UNIQUE (room_id, participant_id)
);

CREATE INDEX idx_chat_group_members_room_participant ON public.chat_members (room_id, participant_id);
CREATE INDEX idx_chat_group_members_participant ON public.chat_members (participant_id);

-- Chat records
CREATE TABLE IF NOT EXISTS public.chat_records
(
    id           BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    room_id      BIGINT            NOT NULL REFERENCES chat_rooms (id) ON DELETE CASCADE,
    sender_id    BIGINT            NOT NULL REFERENCES chat_members (id) ON DELETE NO ACTION,
    content      TEXT              NOT NULL,
    message_type chat_message_type NOT NULL DEFAULT 'text',
    reply_to_id  BIGINT            REFERENCES chat_records (id) ON DELETE SET NULL,
    is_edited    BOOLEAN           NOT NULL DEFAULT FALSE,
    is_deleted   BOOLEAN           NOT NULL DEFAULT FALSE,
    created_at   TIMESTAMPTZ       NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ       NOT NULL DEFAULT now()
);

CREATE INDEX idx_chat_records_room_created ON public.chat_records (room_id, created_at DESC);
CREATE INDEX idx_chat_records_room_id ON public.chat_records (room_id, id DESC);
CREATE INDEX idx_chat_records_room_sender ON public.chat_records (room_id, sender_id);

---- Trigger ----

-- Limit member number of room
CREATE OR REPLACE FUNCTION limit_member_of_room()
    RETURNS trigger AS
$$
DECLARE
    current_count INTEGER;
    max_limit     INTEGER;
BEGIN
    SELECT r.max_members
    INTO max_limit
    FROM public.chat_rooms r
    WHERE r.id = NEW.room_id
        FOR UPDATE;

    SELECT count(1)
    INTO current_count
    FROM public.chat_members m
             JOIN public.participants p ON p.id = m.participant_id
    WHERE m.room_id = NEW.room_id
      AND m.is_deleted = false
      AND p.type != 'system';

    IF current_count >= max_limit THEN
        RAISE EXCEPTION 'User limit exceeded for this account';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_limit_member
    BEFORE INSERT
    ON public.chat_members
    FOR EACH ROW
EXECUTE FUNCTION limit_member_of_room();
