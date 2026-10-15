-- Notifications such as "new message" are coalesced per conversation while unread.
ALTER TABLE notifications
  ADD COLUMN IF NOT EXISTS dedupe_key TEXT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_notifications_unread_dedupe
  ON notifications (user_id, dedupe_key)
  WHERE is_read = FALSE AND dedupe_key IS NOT NULL;
