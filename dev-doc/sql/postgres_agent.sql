---- Types ----

-- Agent status
CREATE TYPE agent_status AS ENUM ('available', 'maintaining', 'discontinued', 'error');

-- Agent types
CREATE TYPE agent_types AS ENUM ('local', 'remote', 'hybrid');

-- Agent engine
CREATE TYPE agent_engine AS ENUM ('gguf', 'onnx', 'api', 'cloud', 'mlc', 'webGPU');

---- Tables ----

-- Agents Table
CREATE TABLE IF NOT EXISTS public.agents
(
    id         BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    name       TEXT         NOT NULL,
    type       agent_types  NOT NULL,
    status     agent_status NOT NULL DEFAULT 'maintaining',
    engine     agent_engine NOT NULL,
    created_at TIMESTAMP    NOT NULL DEFAULT now(),
    created_by BIGINT REFERENCES users (id) ON DELETE CASCADE,
    updated_at TIMESTAMP    NOT NULL DEFAULT now(),
    updated_by BIGINT REFERENCES users (id) ON DELETE CASCADE
);
