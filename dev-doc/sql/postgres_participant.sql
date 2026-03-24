---- Types ----

-- If needed to add type
-- ALTER TYPE participant_type ADD VALUE 'bot';
CREATE TYPE participant_type AS ENUM ('user', 'agent', 'system');

---- Tables ----

-- Participants system types
CREATE TABLE public.participant_system_types
(
    code        TEXT PRIMARY KEY,
    description TEXT
);

-- Participants of chat
CREATE TABLE IF NOT EXISTS public.participants
(
    id         BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    type       participant_type NOT NULL,
    created_at TIMESTAMP        NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS public.participant_users
(
    participant_id BIGINT PRIMARY KEY REFERENCES public.participants (id) ON DELETE CASCADE,
    user_id        BIGINT NOT NULL UNIQUE REFERENCES public.users (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS public.participant_agents
(
    participant_id BIGINT PRIMARY KEY REFERENCES public.participants (id) ON DELETE CASCADE,
    agent_id       BIGINT NOT NULL UNIQUE REFERENCES public.agents (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS public.participant_systems
(
    participant_id BIGINT PRIMARY KEY REFERENCES public.participants (id) ON DELETE CASCADE,
    system_type    TEXT NOT NULL REFERENCES participant_system_types (code)
);

---- Indexes ----
CREATE INDEX IF NOT EXISTS idx_participants_type ON public.participants (type);

---- Default data ----
INSERT INTO public.participant_system_types (code, description)
VALUES ('alert', 'Alert message sender'),
       ('notification', 'Normal system notifier');

INSERT INTO public.participants (type)
VALUES ('system'),
       ('system');

INSERT INTO public.participant_systems (participant_id, system_type)
VALUES (1, 'alert'),
       (2, 'notification');