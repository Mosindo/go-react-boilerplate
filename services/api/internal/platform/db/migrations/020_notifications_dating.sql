-- Notifications for the dating app: keyset pagination and unread message coalescing lookups.
CREATE INDEX IF NOT EXISTS idx_notifications_user_created_id
  ON notifications (user_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_notifications_unread_message_conversation
  ON notifications (user_id, (data->>'conversationId'))
  WHERE is_read = FALSE AND type = 'message';
