---- Drop Trigger Query ----

-- Chats
DROP TRIGGER IF EXISTS trg_limit_member ON goat.public.chat_members;

-- Accounts
DROP TRIGGER IF EXISTS trg_limit_user ON goat.public.users;

---- Drop Indexes Query ----

-- Friendship
DROP TRIGGER IF EXISTS idx_user_friendships_friend_id ON goat.public.chat_members;
DROP TRIGGER IF EXISTS idx_user_friendships_user_id ON goat.public.chat_members;

-- X3DH
DROP INDEX IF EXISTS idx_member_sender_keys_member;
DROP INDEX IF EXISTS idx_user_one_time_pre_keys_lookup;
DROP INDEX IF EXISTS idx_user_signed_pre_keys_active;

-- Delivery queue
DROP INDEX IF EXISTS idx_delivery_queue_user_status;
DROP INDEX IF EXISTS idx_delivery_queue_status_create_at;

-- Chat invitations
DROP INDEX IF EXISTS idx_chat_invitation_room_id_invite_id;

-- Chats
DROP INDEX IF EXISTS idx_chat_records_room_sender;
DROP INDEX IF EXISTS idx_chat_records_room_created;
DROP INDEX IF EXISTS idx_chat_group_members_participant;
DROP INDEX IF EXISTS idx_chat_group_members_room_participant;

-- Participants
DROP INDEX IF EXISTS idx_participants_type;

---- Drop Tables Query ----

-- Friendship
DROP TABLE IF EXISTS goat.public.user_friendships CASCADE;

-- X3DH
DROP TABLE IF EXISTS goat.public.user_identity_keys CASCADE;
DROP TABLE IF EXISTS goat.public.user_signed_pre_keys CASCADE;
DROP TABLE IF EXISTS goat.public.user_one_time_pre_keys CASCADE;
DROP TABLE IF EXISTS goat.public.member_sender_keys CASCADE;

-- Delivery queue
DROP TABLE IF EXISTS goat.public.delivery_queue CASCADE;

-- Chat invitations
DROP TABLE IF EXISTS goat.public.chat_invitations CASCADE;

-- Chats
DROP TABLE IF EXISTS goat.public.chat_records CASCADE;
DROP TABLE IF EXISTS goat.public.chat_members CASCADE;
DROP TABLE IF EXISTS goat.public.chat_rooms CASCADE;

-- Participants
DROP TABLE IF EXISTS goat.public.participant_systems CASCADE;
DROP TABLE IF EXISTS goat.public.participant_agents CASCADE;
DROP TABLE IF EXISTS goat.public.participant_users CASCADE;
DROP TABLE IF EXISTS goat.public.participants CASCADE;
DROP TABLE IF EXISTS goat.public.participant_system_types CASCADE;

-- Agents
DROP TABLE IF EXISTS goat.public.agents CASCADE;

-- Emails
DROP TABLE IF EXISTS goat.public.email_logs CASCADE;

-- Devices
DROP TABLE IF EXISTS goat.public.account_login_events CASCADE;
DROP TABLE IF EXISTS goat.public.account_sessions CASCADE;
DROP TABLE IF EXISTS goat.public.accounts_devices CASCADE;
DROP TABLE IF EXISTS goat.public.devices CASCADE;

-- Accounts
DROP TABLE IF EXISTS goat.public.users_roles CASCADE;
DROP TABLE IF EXISTS goat.public.roles CASCADE;
DROP TABLE IF EXISTS goat.public.users CASCADE;
DROP TABLE IF EXISTS goat.public.accounts CASCADE;

---- Drop Enums Query ----

-- Friendship
DROP TYPE IF EXISTS friendship_status;

-- Chat invitations
DROP TYPE IF EXISTS invitation_status;

-- Chats
DROP TYPE IF EXISTS chat_message_type;
DROP TYPE IF EXISTS chat_member_role;
DROP TYPE IF EXISTS chat_room_type;

-- Participants
DROP TYPE IF EXISTS participant_type;

-- Agents
DROP TYPE IF EXISTS agent_status;
DROP TYPE IF EXISTS agent_types;
DROP TYPE IF EXISTS agent_engine;

-- Emails
DROP TYPE IF EXISTS email_status;

-- Devices
DROP TYPE IF EXISTS device_platform;

-- Accounts
DROP TYPE IF EXISTS account_status;
