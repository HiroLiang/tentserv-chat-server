---- Drop Trigger Query ----

-- Chats
DROP TRIGGER IF EXISTS trg_limit_member ON goat.public.chat_members;

-- Accounts
DROP TRIGGER IF EXISTS trg_limit_user ON goat.public.users;

---- Drop Indexes Query ----

-- Chats
DROP INDEX IF EXISTS idx_chat_records_room_sender;
DROP INDEX IF EXISTS idx_chat_records_room_created;
DROP INDEX IF EXISTS idx_chat_group_members_participant;
DROP INDEX IF EXISTS idx_chat_group_members_room_participant;

-- Participants
DROP INDEX IF EXISTS idx_participants_type;

---- Drop Tables Query ----

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

-- Devices
DROP TYPE IF EXISTS device_platform;

-- Accounts
DROP TYPE IF EXISTS account_status;
