-- ============================================================
-- [EN] File: 08_chat_invitations.sql  |  Execution order: 8 of 10
--      Creates the chat invitation layer: invitation_status and
--      invitation_type enums, chat_invitations table, and its indexes.
--
-- [中] 檔案：08_chat_invitations.sql  |  執行順序：第 8 支（共 10 支）
--      建立聊天邀請層：invitation_status、invitation_type 列舉、
--      chat_invitations 資料表及其索引。
--
-- [日] ファイル：08_chat_invitations.sql  |  実行順：8/10
--      チャット招待層を作成する：invitation_status、invitation_type 列挙、
--      chat_invitations テーブルとそのインデックス。
--
-- Dependencies | 依賴 | 依存: 07_chat.sql (chat_rooms), 05_participant.sql (participants)
-- Creates      | 建立 | 作成:
--   TYPE  invitation_status, invitation_type
--   TABLE chat_invitations
--   INDEX idx_chat_invitation_room_id_invite_id, idx_chat_invitation_room_id_inviter_id
-- ============================================================


-- ============================================================
-- SECTION 1: ENUM TYPES
-- [EN] invitation_status tracks the lifecycle of an invite
--      (pending → accepted / rejected / blocked).
--      invitation_type distinguishes whether the row was created
--      by a room member inviting someone ('invitation') or by
--      an outsider requesting to join ('join_request').
-- [中] invitation_status 追蹤邀請生命週期（pending → accepted / rejected / blocked）。
--      invitation_type 區分該列是由房間成員邀請他人（'invitation'）
--      還是外部人員申請加入（'join_request'）所建立。
-- [日] invitation_status は招待のライフサイクルを追跡する
--      （pending → accepted / rejected / blocked）。
--      invitation_type は、ルームメンバーが招待した ('invitation') か、
--      外部ユーザーが参加申請した ('join_request') かを区別する。
-- ============================================================

-- Invitation lifecycle status | 邀請生命週期狀態 | 招待ライフサイクルステータス
CREATE TYPE invitation_status AS ENUM ('pending', 'accepted', 'rejected', 'blocked');

-- Invitation origin type | 邀請來源類型 | 招待の発生元タイプ
CREATE TYPE invitation_type   AS ENUM ('join_request', 'invitation');


-- ============================================================
-- SECTION 2: TABLES
-- [EN] chat_invitations links a room to an inviter (participant
--      already in the room) and an invitee (target participant).
--      expires_at is optional — NULL means no expiry.
-- [中] chat_invitations 連結房間、邀請者（已在房間內的參與者）
--      及被邀請者（目標參與者）。
--      expires_at 為可選欄位，NULL 表示無到期時間。
-- [日] chat_invitations はルームと招待者（既存メンバー）、
--      被招待者（対象参加者）をリンクする。
--      expires_at はオプション — NULL は期限なしを意味する。
-- ============================================================

-- Chat invitations | 聊天邀請 | チャット招待
CREATE TABLE IF NOT EXISTS public.chat_invitations
(
    id              BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    room_id         BIGINT            NOT NULL REFERENCES public.chat_rooms (id),
    inviter_id      BIGINT            NOT NULL REFERENCES public.participants (id),
    invitee_id      BIGINT            NOT NULL REFERENCES public.participants (id),
    status          invitation_status NOT NULL DEFAULT 'pending',
    invitation_type invitation_type   NOT NULL DEFAULT 'invitation',
    expires_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ       NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ       NOT NULL DEFAULT now()
);


-- ============================================================
-- SECTION 3: INDEXES
-- [EN] Two composite indexes: one for "show all invites to this
--      invitee in this room", one for "show all invites sent by
--      this inviter in this room".
-- [中] 兩個複合索引：一個用於「查詢此房間中發給此被邀請者的所有邀請」，
--      一個用於「查詢此房間中此邀請者發送的所有邀請」。
-- [日] 2 つの複合インデックス：「このルームのこの被招待者への招待をすべて表示」と
--      「このルームでこの招待者が送った招待をすべて表示」のため。
-- ============================================================

-- By invitee | 按被邀請者 | 被招待者別
CREATE INDEX idx_chat_invitation_room_id_invite_id  ON public.chat_invitations (room_id, invitee_id);
-- By inviter | 按邀請者 | 招待者別
CREATE INDEX idx_chat_invitation_room_id_inviter_id ON public.chat_invitations (room_id, inviter_id);
