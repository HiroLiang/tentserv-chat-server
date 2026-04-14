-- ============================================================
-- [EN] File: 00_drop_all.sql
--      Drops ALL database objects created by files 01–10 in reverse
--      dependency order. Safe to run even when objects do not exist
--      (every statement uses IF EXISTS). Run this before re-applying
--      the schema from scratch, or to reset a development database.
--
-- [中] 檔案：00_drop_all.sql
--      以反向依賴順序刪除 01～10 號檔案所建立的所有資料庫物件。
--      所有陳述式均使用 IF EXISTS，物件不存在時不會報錯。
--      用於從頭重建 schema，或重置開發資料庫。
--
-- [日] ファイル：00_drop_all.sql
--      01〜10 のファイルが作成したすべてのデータベースオブジェクトを
--      依存関係の逆順で削除します。すべての文に IF EXISTS を使用し、
--      オブジェクトが存在しなくてもエラーになりません。
--      スキーマを最初から再適用する前、または開発 DB をリセットする際に実行します。
--
-- Drop order | 刪除順序 | 削除順序:
--   1. Triggers       (Trigger / 觸發器 / トリガー)
--   2. Indexes        (Index / 索引 / インデックス)
--   3. Tables         (Table / 資料表 / テーブル)  ← reverse dependency
--   4. Functions      (Function / 函數 / 関数)
--   5. ENUMs / Types  (Enum / 列舉類型 / 列挙型)
-- ============================================================


-- ============================================================
-- SECTION 1: TRIGGERS
-- [EN] Drop triggers before dropping the functions they reference.
-- [中] 在刪除 function 之前先刪除觸發器，避免依賴衝突。
-- [日] 参照している関数を削除する前にトリガーを先に削除します。
-- ============================================================

-- chat_members member-count guard | 聊天室成員數量限制觸發器 | チャットメンバー数制限トリガー
DROP TRIGGER IF EXISTS trg_limit_member ON public.chat_members;

-- users per-account user-count guard | 每帳號使用者數量限制觸發器 | アカウントごとのユーザー数制限トリガー
DROP TRIGGER IF EXISTS trg_limit_user ON public.users;


-- ============================================================
-- SECTION 2: INDEXES
-- [EN] Drop indexes explicitly. Tables are dropped with CASCADE below,
--      but being explicit avoids hidden surprises and documents intent.
-- [中] 明確刪除索引。下方刪除表時雖會 CASCADE，但明確列出有助於文件追蹤。
-- [日] インデックスを明示的に削除します。テーブルを CASCADE で削除しても
--      消えますが、意図を明確にするために個別に列挙します。
-- ============================================================

-- E2EE: sender_key_requests | 發送方金鑰請求 | 送信者鍵リクエスト
DROP INDEX IF EXISTS idx_sender_key_requests_provider;
DROP INDEX IF EXISTS idx_self_sender_key_sync_distributions_requester;
DROP INDEX IF EXISTS idx_participant_self_sender_key_syncs_participant;
DROP INDEX IF EXISTS idx_sender_key_receipts_sender;
DROP INDEX IF EXISTS idx_sender_key_receipts_receiver;

-- E2EE: sender_key_distributions | 發送方金鑰分發確認 | 送信者鍵配布確認
DROP INDEX IF EXISTS idx_sender_key_distributions_sender;
DROP INDEX IF EXISTS idx_sender_key_distributions_receiver;

-- E2EE: participant_self_sender_key_syncs | 自有送信者鍵同步 | 自己送信者鍵同期
DROP INDEX IF EXISTS idx_participant_self_sender_key_syncs_status;

-- E2EE: member_sender_keys | 成員發送方金鑰 | メンバー送信者鍵
DROP INDEX IF EXISTS idx_member_sender_keys_member;

-- E2EE: user_signed_pre_keys | 已簽名預金鑰 | 署名済みプリキー
DROP INDEX IF EXISTS idx_user_signed_pre_keys_active;

-- E2EE: user_one_time_pre_keys | 一次性預金鑰 | ワンタイムプリキー
DROP INDEX IF EXISTS idx_user_one_time_pre_keys_lookup;

-- Delivery queue | 投遞佇列 | 配信キュー
DROP INDEX IF EXISTS idx_delivery_queue_user_status;
DROP INDEX IF EXISTS idx_delivery_queue_status_create_at;

-- Chat invitations | 聊天邀請 | チャット招待
DROP INDEX IF EXISTS idx_chat_invitation_room_id_invite_id;
DROP INDEX IF EXISTS idx_chat_invitation_room_id_inviter_id;

-- Chat records | 聊天訊息 | チャットレコード
DROP INDEX IF EXISTS idx_chat_records_room_sender;
DROP INDEX IF EXISTS idx_chat_records_room_created;
DROP INDEX IF EXISTS idx_chat_records_room_id;

-- Chat members | 聊天室成員 | チャットメンバー
DROP INDEX IF EXISTS idx_chat_group_members_room_participant;
DROP INDEX IF EXISTS idx_chat_group_members_participant;

-- Participants | 參與者 | 参加者
DROP INDEX IF EXISTS idx_participants_type;

-- Friendships | 好友關係 | フレンドシップ
DROP INDEX IF EXISTS idx_user_friendships_user_id;
DROP INDEX IF EXISTS idx_user_friendships_friend_id;


-- ============================================================
-- SECTION 3: TABLES  (reverse dependency order)
-- [EN] Drop tables from most-dependent to least-dependent.
--      CASCADE is included as a safety net for any cross-references
--      not explicitly listed here.
-- [中] 從最依賴到最基礎的順序刪除表。
--      加上 CASCADE 作為保險，以防尚有未列出的交叉引用。
-- [日] 最も依存度の高いテーブルから基底テーブルの順に削除します。
--      ここに列挙されていない相互参照に備えて CASCADE を付けます。
-- ============================================================

-- E2EE: sender key layer | 發送方金鑰層 | 送信者鍵レイヤー
DROP TABLE IF EXISTS public.sender_key_requests CASCADE;
DROP TABLE IF EXISTS public.sender_key_distributions CASCADE;
DROP TABLE IF EXISTS public.sender_key_receipts CASCADE;
DROP TABLE IF EXISTS public.self_sender_key_sync_distributions CASCADE;
DROP TABLE IF EXISTS public.participant_self_sender_key_syncs CASCADE;
DROP TABLE IF EXISTS public.member_sender_keys CASCADE;
DROP TABLE IF EXISTS public.user_one_time_pre_keys CASCADE;
DROP TABLE IF EXISTS public.user_signed_pre_keys CASCADE;
DROP TABLE IF EXISTS public.user_identity_keys CASCADE;

-- Delivery queue | 投遞佇列 | 配信キュー
DROP TABLE IF EXISTS public.delivery_queue CASCADE;

-- Chat invitations | 聊天邀請 | チャット招待
DROP TABLE IF EXISTS public.chat_invitations CASCADE;

-- Chat core | 聊天核心 | チャットコア
DROP TABLE IF EXISTS public.chat_records CASCADE;
DROP TABLE IF EXISTS public.chat_members CASCADE;
DROP TABLE IF EXISTS public.chat_rooms CASCADE;

-- Participant subtypes | 參與者子類型 | 参加者サブタイプ
DROP TABLE IF EXISTS public.participant_systems CASCADE;
DROP TABLE IF EXISTS public.participant_agents CASCADE;
DROP TABLE IF EXISTS public.participant_users CASCADE;
DROP TABLE IF EXISTS public.participants CASCADE;
DROP TABLE IF EXISTS public.participant_system_types CASCADE;

-- Friendships | 好友關係 | フレンドシップ
DROP TABLE IF EXISTS public.user_friendships CASCADE;

-- Agents | AI Agent | AI エージェント
DROP TABLE IF EXISTS public.agents CASCADE;

-- Email logs | 電子郵件日誌 | メールログ
DROP TABLE IF EXISTS public.email_logs CASCADE;

-- Device & session layer | 設備與會話層 | デバイス・セッション層
DROP TABLE IF EXISTS public.account_login_events CASCADE;
DROP TABLE IF EXISTS public.account_sessions CASCADE;
DROP TABLE IF EXISTS public.accounts_devices CASCADE;
DROP TABLE IF EXISTS public.devices CASCADE;

-- Account & user core | 帳號與使用者核心 | アカウント・ユーザーコア
DROP TABLE IF EXISTS public.users_roles CASCADE;
DROP TABLE IF EXISTS public.roles CASCADE;
DROP TABLE IF EXISTS public.users CASCADE;
DROP TABLE IF EXISTS public.accounts CASCADE;


-- ============================================================
-- SECTION 4: FUNCTIONS
-- [EN] Drop functions after triggers (which called them) are gone.
-- [中] 在觸發器（呼叫這些函數的）刪除後再刪除函數。
-- [日] これらの関数を呼び出していたトリガーが削除された後に関数を削除します。
-- ============================================================

-- Room member-count guard | 聊天室成員數量限制函數 | チャットメンバー数制限関数
DROP FUNCTION IF EXISTS limit_member_of_room();

-- Account user-count guard | 帳號使用者數量限制函數 | アカウントユーザー数制限関数
DROP FUNCTION IF EXISTS limit_user_per_account();


-- ============================================================
-- SECTION 5: ENUM TYPES
-- [EN] Drop enum types last; tables referencing them are already gone.
-- [中] 最後刪除列舉類型；引用它們的表已先被刪除。
-- [日] 列挙型を最後に削除します。参照していたテーブルはすでに削除済みです。
-- ============================================================

-- Chat invitations enums | 聊天邀請列舉 | チャット招待列挙
DROP TYPE IF EXISTS invitation_type;
DROP TYPE IF EXISTS invitation_status;

-- Friendship enum | 好友關係列舉 | フレンドシップ列挙
DROP TYPE IF EXISTS friendship_status;

-- Chat enums | 聊天列舉 | チャット列挙
DROP TYPE IF EXISTS chat_message_type;
DROP TYPE IF EXISTS chat_member_role;
DROP TYPE IF EXISTS chat_room_type;

-- Participant enum | 參與者列舉 | 参加者列挙
DROP TYPE IF EXISTS participant_type;

-- Agent enums | Agent 列舉 | エージェント列挙
DROP TYPE IF EXISTS agent_status;
DROP TYPE IF EXISTS agent_types;
DROP TYPE IF EXISTS agent_engine;

-- Email enum | 郵件列舉 | メール列挙
DROP TYPE IF EXISTS email_status;

-- Device enum | 設備列舉 | デバイス列挙
DROP TYPE IF EXISTS device_platform;
DROP TYPE IF EXISTS account_device_status;

-- Account enum | 帳號列舉 | アカウント列挙
DROP TYPE IF EXISTS account_status;
