-- ============================================================
-- [EN] File: 06_friendship.sql  |  Execution order: 6 of 10
--      Creates the friendship layer: friendship_status enum,
--      user_friendships table, and lookup indexes.
--
-- [中] 檔案：06_friendship.sql  |  執行順序：第 6 支（共 10 支）
--      建立好友關係層：friendship_status 列舉、user_friendships 資料表
--      及查詢索引。
--
-- [日] ファイル：06_friendship.sql  |  実行順：6/10
--      フレンドシップ層を作成する：friendship_status 列挙、
--      user_friendships テーブル、検索インデックス。
--
-- Dependencies | 依賴 | 依存: 01_account.sql (users)
-- Creates      | 建立 | 作成:
--   TYPE  friendship_status
--   TABLE user_friendships
--   INDEX idx_user_friendships_user_id, idx_user_friendships_friend_id
-- ============================================================


-- ============================================================
-- SECTION 1: ENUM TYPES
-- [EN] States of a friendship relationship: pending request,
--      accepted, or blocked.
-- [中] 好友關係狀態：待接受請求、已接受或已封鎖。
-- [日] フレンドシップの状態：申請中・承認済み・ブロック済み。
-- ============================================================

CREATE TYPE friendship_status AS ENUM ('pending', 'accepted', 'blocked');


-- ============================================================
-- SECTION 2: TABLES
-- [EN] user_friendships is a directed edge: user_id → friend_id.
--      A mutual friendship is represented by two rows.
--      UNIQUE (user_id, friend_id) prevents duplicate edges.
-- [中] user_friendships 是有向邊：user_id → friend_id。
--      互相好友關係由兩列表示。UNIQUE 防止重複邊。
-- [日] user_friendships は有向エッジ：user_id → friend_id。
--      相互フレンドシップは 2 行で表現する。UNIQUE で重複エッジを防ぐ。
-- ============================================================

-- User friendships | 好友關係 | フレンドシップ
CREATE TABLE IF NOT EXISTS public.user_friendships
(
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id    BIGINT            REFERENCES users (id) ON DELETE CASCADE,
    friend_id  BIGINT            REFERENCES users (id) ON DELETE CASCADE,
    status     friendship_status NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP         NOT NULL DEFAULT now(),
    updated_at TIMESTAMP         NOT NULL DEFAULT now(),
    UNIQUE (user_id, friend_id)
);


-- ============================================================
-- SECTION 3: INDEXES
-- [EN] Separate indexes on each side of the directed edge so
--      "all my friendships" and "all requests sent to me" are
--      both index-range scans.
-- [中] 為有向邊的兩端各建立索引，讓「我的所有好友關係」與
--      「所有發給我的好友請求」都能走索引範圍掃描。
-- [日] 有向エッジの両端にインデックスを作成し、「自分のすべての
--      フレンドシップ」と「自分宛のすべてのリクエスト」が
--      インデックス範囲スキャンになるようにする。
-- ============================================================

CREATE INDEX IF NOT EXISTS idx_user_friendships_user_id   ON public.user_friendships (user_id);
CREATE INDEX IF NOT EXISTS idx_user_friendships_friend_id ON public.user_friendships (friend_id);
