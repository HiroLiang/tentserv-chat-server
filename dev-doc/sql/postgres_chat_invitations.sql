CREATE TYPE invitation_status AS ENUM ('pending', 'accepted', 'rejected', 'blocked');
CREATE TYPE invitation_type   AS ENUM ('join_request', 'invitation');

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

CREATE INDEX idx_chat_invitation_room_id_invite_id   ON public.chat_invitations (room_id, invitee_id);
CREATE INDEX idx_chat_invitation_room_id_inviter_id  ON public.chat_invitations (room_id, inviter_id);
