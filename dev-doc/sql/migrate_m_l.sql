-- M-1: Rename chat_members.is_delete → is_deleted for consistency with
--       chat_rooms.is_deleted and chat_records.is_deleted.
ALTER TABLE public.chat_members RENAME COLUMN is_delete TO is_deleted;

-- M-4: Add invitation_type to distinguish self-join requests from admin invitations.
--       Enables type guards in ApproveJoinRequest (join_request only) and
--       RespondToInvitation (invitation only).
CREATE TYPE invitation_type AS ENUM ('join_request', 'invitation');
ALTER TABLE public.chat_invitations
    ADD COLUMN invitation_type invitation_type NOT NULL DEFAULT 'invitation';

-- Back-fill: rows where inviter = invitee are join requests.
UPDATE public.chat_invitations SET invitation_type = 'join_request' WHERE inviter_id = invitee_id;

-- M-5: Add expires_at to allow invitation expiry.
ALTER TABLE public.chat_invitations
    ADD COLUMN expires_at TIMESTAMPTZ;

-- L-1: Convert chat_invitations PK from BIGSERIAL to GENERATED ALWAYS AS IDENTITY
--      to match the convention used by all other tables.
ALTER TABLE public.chat_invitations ALTER COLUMN id DROP DEFAULT;
DROP SEQUENCE IF EXISTS public.chat_invitations_id_seq CASCADE;
ALTER TABLE public.chat_invitations ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY;

-- L-2: Index to support FindPendingByRoomAndInviter queries.
CREATE INDEX IF NOT EXISTS idx_chat_invitation_room_id_inviter_id
    ON public.chat_invitations (room_id, inviter_id);
