-- ============================================================
-- [EN] File: 05_participant.sql  |  Execution order: 5 of 10
--      Creates the participant abstraction layer: participant_type
--      enum, participant_system_types lookup, participants root table,
--      and the three subtype tables (users / agents / systems).
--      Seeds built-in system participants (alert + notification).
--
-- [中] 檔案：05_participant.sql  |  執行順序：第 5 支（共 10 支）
--      建立參與者抽象層：participant_type 列舉、participant_system_types
--      查找表、participants 根表，及三個子類型表（users / agents / systems）。
--      插入內建系統參與者（alert + notification）。
--
-- [日] ファイル：05_participant.sql  |  実行順：5/10
--      参加者抽象化層を作成する：participant_type 列挙、
--      participant_system_types ルックアップ、participants ルートテーブル、
--      サブタイプテーブル（users / agents / systems）。
--      組み込みシステム参加者（alert + notification）を挿入する。
--
-- Dependencies | 依賴 | 依存: 01_account.sql (users), 04_agent.sql (agents)
-- Creates      | 建立 | 作成:
--   TYPE  participant_type
--   TABLE participant_system_types, participants
--   TABLE participant_users, participant_agents, participant_systems
--   INDEX idx_participants_type
-- ============================================================


-- ============================================================
-- SECTION 1: ENUM TYPES
-- [EN] Discriminator for the participants table — identifies whether
--      the participant is a human user, an AI agent, or an internal
--      system sender. To add a new variant: ALTER TYPE participant_type ADD VALUE 'bot';
-- [中] participants 表的鑑別器 — 識別參與者為人類使用者、AI Agent 或系統發送者。
--      新增變體：ALTER TYPE participant_type ADD VALUE 'bot';
-- [日] participants テーブルの識別子 — 参加者がユーザー・AI エージェント・
--      システム送信者のいずれかを識別する。
--      新しい値を追加する場合: ALTER TYPE participant_type ADD VALUE 'bot';
-- ============================================================

CREATE TYPE participant_type AS ENUM ('user', 'agent', 'system');


-- ============================================================
-- SECTION 2: TABLES
-- [EN] participant_system_types is a code-lookup for system participant
--      variants (e.g. 'alert', 'notification'). participants is the
--      polymorphic root; each row has exactly one subtype row.
-- [中] participant_system_types 為系統參與者變體的代碼查找表
--      （例如 'alert'、'notification'）。participants 是多型根表，
--      每筆紀錄對應恰好一列子類型紀錄。
-- [日] participant_system_types はシステム参加者の種類コードルックアップ
--      （例：'alert'、'notification'）。participants はポリモーフィックルートテーブルで、
--      各行にはちょうど 1 つのサブタイプ行が対応する。
-- ============================================================

-- System participant type codes | 系統參與者類型代碼 | システム参加者タイプコード
CREATE TABLE public.participant_system_types
(
    code        TEXT PRIMARY KEY,
    description TEXT
);

-- Polymorphic participant root | 多型參與者根表 | ポリモーフィック参加者ルート
CREATE TABLE IF NOT EXISTS public.participants
(
    id         BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    type       participant_type NOT NULL,
    created_at TIMESTAMP        NOT NULL DEFAULT now()
);

-- Subtype: human users | 子類型：人類使用者 | サブタイプ：人間ユーザー
CREATE TABLE IF NOT EXISTS public.participant_users
(
    participant_id BIGINT PRIMARY KEY REFERENCES public.participants (id) ON DELETE CASCADE,
    user_id        BIGINT NOT NULL UNIQUE REFERENCES public.users (id) ON DELETE CASCADE
);

-- Subtype: AI agents | 子類型：AI Agent | サブタイプ：AI エージェント
CREATE TABLE IF NOT EXISTS public.participant_agents
(
    participant_id BIGINT PRIMARY KEY REFERENCES public.participants (id) ON DELETE CASCADE,
    agent_id       BIGINT NOT NULL UNIQUE REFERENCES public.agents (id) ON DELETE CASCADE
);

-- Subtype: system senders | 子類型：系統發送者 | サブタイプ：システム送信者
CREATE TABLE IF NOT EXISTS public.participant_systems
(
    participant_id BIGINT PRIMARY KEY REFERENCES public.participants (id) ON DELETE CASCADE,
    system_type    TEXT NOT NULL REFERENCES participant_system_types (code)
);


-- ============================================================
-- SECTION 3: INDEXES
-- [EN] Index on type for fast participant-type filtering queries.
-- [中] 在 type 上建立索引，加速依參與者類型過濾的查詢。
-- [日] type への高速フィルタリングクエリのためにインデックスを作成する。
-- ============================================================

CREATE INDEX IF NOT EXISTS idx_participants_type ON public.participants (type);


-- ============================================================
-- SECTION 4: DEFAULT DATA
-- [EN] Two built-in system participants: 'alert' and 'notification'.
--      These are pre-assigned IDs 1 and 2 by identity generation order.
-- [中] 兩個內建系統參與者：'alert' 與 'notification'。
--      依身份生成順序預先指派 ID 1 和 2。
-- [日] 2 つの組み込みシステム参加者：'alert' と 'notification'。
--      ID は生成順序により 1 と 2 が割り当てられる。
-- ============================================================

INSERT INTO public.participant_system_types (code, description)
VALUES ('alert',        'Alert message sender'),
       ('notification', 'Normal system notifier');

INSERT INTO public.participants (type)
VALUES ('system'),
       ('system');

INSERT INTO public.participant_systems (participant_id, system_type)
VALUES (1, 'alert'),
       (2, 'notification');
