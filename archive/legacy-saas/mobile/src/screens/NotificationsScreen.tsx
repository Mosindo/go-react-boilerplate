import React, { useCallback, useEffect, useMemo, useState } from "react";
import { FlatList, StyleSheet, View } from "react-native";
import {
  listNotifications,
  markNotificationRead,
  type Notification
} from "../api/platform";
import { Header, ScreenContainer } from "../shared/layout";
import { EmptyView, ErrorView, LoadingView } from "../shared/feedback";
import { Button, NotificationItem, spacing } from "../shared/ui";

type NotificationsScreenProps = {
  token: string;
};

function formatNotificationDate(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "Unknown date";
  }
  return date.toLocaleString();
}

export default function NotificationsScreen({ token }: NotificationsScreenProps) {
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const nextNotifications = await listNotifications(token);
      setNotifications(nextNotifications);
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : "could not load notifications");
    } finally {
      setLoading(false);
    }
  }, [token]);

  useEffect(() => {
    load();
  }, [load]);

  async function onMarkRead(item: Notification) {
    if (item.isRead) {
      return;
    }

    try {
      const updated = await markNotificationRead(token, item.id);
      setNotifications((prev) => prev.map((current) => (current.id === updated.id ? updated : current)));
    } catch (markError) {
      setError(markError instanceof Error ? markError.message : "could not update notification");
    }
  }

  const unreadCount = useMemo(
    () => notifications.filter((item) => !item.isRead).length,
    [notifications]
  );

  if (loading && notifications.length === 0) {
    return (
      <ScreenContainer testID="notifications-screen">
        <LoadingView fullScreen label="Loading notifications..." />
      </ScreenContainer>
    );
  }

  return (
    <ScreenContainer contentMaxWidth={820} testID="notifications-screen">
      <Header
        action={
          <View style={styles.actions}>
            <Button
              disabled={loading}
              label="Reload"
              onPress={load}
              size="sm"
              testID="notifications-reload-button"
              variant="outline"
            />
          </View>
        }
        eyebrow="Inbox"
        subtitle={`${unreadCount} unread`}
        style={styles.headerShell}
        title="Notifications"
      />

      {error ? (
        <ErrorView actionLabel="Retry" message={error} onAction={() => void load()} style={styles.error} />
      ) : null}

      <FlatList
        contentContainerStyle={styles.list}
        data={notifications}
        keyExtractor={(item) => item.id}
        ListEmptyComponent={
          <EmptyView
            message="Notifications from messages, posts, and account activity will appear here."
            title="No notifications yet"
          />
        }
        renderItem={({ item }) => (
          <NotificationItem
            body={item.body}
            createdAtLabel={formatNotificationDate(item.createdAt)}
            isRead={item.isRead}
            onPress={() => void onMarkRead(item)}
            title={item.title}
            type={item.type}
          />
        )}
      />
    </ScreenContainer>
  );
}

const styles = StyleSheet.create({
  headerShell: {
    marginBottom: spacing.lg,
  },
  actions: {
    flexDirection: "row",
    gap: spacing.sm,
    paddingTop: spacing.md
  },
  error: {
    marginBottom: spacing.sm
  },
  list: {
    paddingBottom: spacing.xxxl
  }
});
