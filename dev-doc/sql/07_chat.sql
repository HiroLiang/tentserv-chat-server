-- ============================================================
-- [EN] File: 07_chat.sql  |  Execution order: 7 of 10
--      Creates the chat core layer: chat_room_type, chat_member_role,
--      chat_message_type enums; chat_rooms, chat_members, chat_records
--      tables with their indexes; and the per-room member-count trigger.
--
-- [中] 檔案：07_chat.sql  |  執行順序：第 7 支（共 10 支）
--      建立聊天核心層：chat_room_type、chat_member_role、chat_message_type
--      列舉；chat_rooms、chat_members、chat_records 資料表及其索引；
--      以及每房間成員數量限制觸發器。
--
-- [日] ファイル：07_chat.sql  |  実行順：7/10
--      チャットコア層を作成する：chat_room_type、chat_member_role、
--      chat_message_type 列挙；chat_rooms、chat_members、chat_records
--      テーブルとそのインデックス；ルームごとのメンバー数制限トリガー。
--
-- Dependencies | 依賴 | 依存: 05_participant.sql (participants)
-- Creates      | 建立 | 作成:
--   TYPE  chat_room_type, chat_member_role, chat_message_type
--   TABLE chat_rooms, chat_members, chat_records
--   INDEX idx_chat_group_members_room_participant, idx_chat_group_members_participant
--   INDEX idx_chat_records_room_created, idx_chat_records_room_id, idx_chat_records_room_sender
--   FUNCTION limit_member_of_room()
--   TRIGGER trg_limit_member ON chat_members
-- ============================================================


-- ============================================================
-- SECTION 1: ENUM TYPES
-- [EN] chat_room_type distinguishes room topology (direct DM, group,
--      broadcast channel, or bot conversation).
--      chat_member_role is the permission tier inside a room.
--      chat_message_type describes the payload carried in chat_records.
-- [中] chat_room_type 區分房間拓撲（直接私訊、群組、廣播頻道或機器人對話）。
--      chat_member_role 是房間內的權限層級。
--      chat_message_type 說明 chat_records 中的訊息負載類型。
-- [日] chat_room_type はルームトポロジーを区別する（ダイレクト、グループ、
--      ブロードキャストチャンネル、ボット会話）。
--      chat_member_role はルーム内の権限レベル。
--      chat_message_type は chat_records のペイロード種別を表す。
-- ============================================================

-- Room topology | 房間拓撲 | ルームトポロジー
CREATE TYPE chat_room_type AS ENUM ('direct', 'group', 'channel', 'bot');

-- In-room permission tier | 房間內權限層級 | ルーム内権限レベル
CREATE TYPE chat_member_role AS ENUM ('owner', 'admin', 'member', 'guest');

-- Message payload type | 訊息負載類型 | メッセージペイロード種別
CREATE TYPE chat_message_type AS ENUM ('text', 'image', 'file', 'icon', 'system');


-- ============================================================
-- SECTION 2: TABLES
-- [EN] chat_rooms is the root. chat_members links a participant to
--      a room with role/mute/soft-delete semantics. chat_records
--      stores every message with an optional reply chain.
-- [中] chat_rooms 為根表。chat_members 將參與者連結至房間，
--      包含角色/靜音/軟刪除語意。chat_records 儲存所有訊息及可選的回覆鏈。
-- [日] chat_rooms がルート。chat_members は参加者をルームにリンクし、
--      ロール・ミュート・論理削除のセマンティクスを持つ。
--      chat_records はすべてのメッセージとオプションの返信チェーンを格納する。
-- ============================================================

-- Chat rooms | 聊天室 | チャットルーム
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

-- Chat members | 聊天室成員 | チャットメンバー
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

-- Chat records (messages) | 聊天訊息 | チャットレコード（メッセージ）
CREATE TABLE IF NOT EXISTS public.chat_records
(
    id                 BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    room_id            BIGINT            NOT NULL REFERENCES chat_rooms (id) ON DELETE CASCADE,
    sender_id          BIGINT            NOT NULL REFERENCES chat_members (id) ON DELETE NO ACTION,
    sender_device_id   UUID              NOT NULL REFERENCES devices (id) ON DELETE NO ACTION,
    sender_key_version BIGINT            NOT NULL,
    content            TEXT              NOT NULL,
    message_type       chat_message_type NOT NULL DEFAULT 'text',
    reply_to_id        BIGINT            REFERENCES chat_records (id) ON DELETE SET NULL,
    is_edited          BOOLEAN           NOT NULL DEFAULT FALSE,
    is_deleted         BOOLEAN           NOT NULL DEFAULT FALSE,
    created_at         TIMESTAMPTZ       NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ       NOT NULL DEFAULT now()
);


-- ============================================================
-- SECTION 3: INDEXES
-- [EN] chat_members: composite index for room-level membership
--      queries; secondary index for participant-level queries.
--      chat_records: three indexes covering the most common access
--      patterns (recent messages, page by id, per-sender queries).
-- [中] chat_members：房間層級成員查詢的複合索引；參與者層級查詢的次級索引。
--      chat_records：三個索引覆蓋最常見存取模式（最新訊息、ID 分頁、
--      按發送者查詢）。
-- [日] chat_members：ルームレベルのメンバーシップクエリ用複合インデックス；
--      参加者レベルクエリ用の二次インデックス。
--      chat_records：最も一般的なアクセスパターン（最新メッセージ、
--      ID ページング、送信者別クエリ）をカバーする 3 つのインデックス。
-- ============================================================

-- Members: room+participant lookup | 成員：房間+參與者查詢 | メンバー：ルーム+参加者検索
CREATE INDEX idx_chat_group_members_room_participant ON public.chat_members (room_id, participant_id);
CREATE INDEX idx_chat_group_members_participant      ON public.chat_members (participant_id);

-- Records: ordered by time | 訊息：按時間排序 | レコード：時間順
CREATE INDEX idx_chat_records_room_created ON public.chat_records (room_id, created_at DESC);
-- Records: paging by id | 訊息：ID 分頁 | レコード：ID ページング
CREATE INDEX idx_chat_records_room_id      ON public.chat_records (room_id, id DESC);
-- Records: per-sender queries | 訊息：按發送者查詢 | レコード：送信者別クエリ
CREATE INDEX idx_chat_records_room_sender  ON public.chat_records (room_id, sender_id);


-- ============================================================
-- SECTION 4: FUNCTION + TRIGGER
-- [EN] Enforce chat_rooms.max_members at the DB level.
--      The count excludes soft-deleted members and system participants.
--      Fires BEFORE INSERT on public.chat_members.
-- [中] 在資料庫層面強制執行 chat_rooms.max_members 限制。
--      計數排除已軟刪除成員與系統參與者。
--      在 public.chat_members 的 INSERT 前觸發。
-- [日] DB レベルで chat_rooms.max_members を強制する。
--      カウントは論理削除済みメンバーとシステム参加者を除外する。
--      public.chat_members の INSERT 前に発火する。
-- ============================================================

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

    IF max_limit > 0 AND current_count >= max_limit THEN
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
