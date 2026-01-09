-- Migration: Copy existing users to new User table format
-- Run this after the API server creates the new User table

-- First, rename old table to backup
ALTER TABLE User RENAME TO User_backup;

-- The API server will create the new User table automatically
-- After that, run this to copy the users:

INSERT INTO User (user_id, user_name, email, password_hash, language, active, created_at, last_login)
SELECT
    username as user_id,
    full_name as user_name,
    '' as email,  -- Empty email, can be updated later
    password_hash,
    language,
    active,
    datetime('now') as created_at,
    datetime('now') as last_login
FROM User_backup;

-- Verify migration
SELECT 'Migrated users:' as info;
SELECT user_id, user_name, language, active FROM User;

-- After verifying, you can drop the backup:
-- DROP TABLE User_backup;
