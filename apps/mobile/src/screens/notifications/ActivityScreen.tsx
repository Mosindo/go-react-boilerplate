import React, { useMemo } from "react";
import { FlatList, Pressable, RefreshControl, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useQueryClient } from "@tanstack/react-query";
import { Ionicons } from "@expo/vector-icons";
import { notificationsApi } from "../../api/social";
import type { AppNotification } from "../../api/types";
import { keys, useNotifications } from "../../hooks/queries";
import type { AppStackParamList } from "../../navigation/types";
import { colors, radii, spacing } from "../../theme/tokens";
import { Button, EmptyState, ErrorState, Header, LoadingState, Screen, Text } from "../../ui";
import { errorMessage } from "../../utils/errors";
import { relativeTime } from "../../utils/format";

type Nav = NativeStackNavigationProp<AppStackParamList>;

export default function ActivityScreen() {
  const navigation = useNavigation<Nav>();
  const qc = useQueryClient();
  const query = useNotifications();
  const items = useMemo(() => query.data?.pages.flatMap((p) => p.notifications) ?? [], [query.data]);
  const unread = query.data?.pages[0]?.unreadCount ?? 0;

  const open = async (n: AppNotification) => {
    if (!n.isRead) {
      await notificationsApi.markRead(n.id).catch(() => undefined);
      void qc.invalidateQueries({ queryKey: keys.notifications });
    }
    if (n.matchId) {
      navigation.navigate("Conversation", { matchId: n.matchId });
    }
  };

  const readAll = async () => {
    await notificationsApi.markAllRead().catch(() => undefined);
    void qc.invalidateQueries({ queryKey: keys.notifications });
  };

  const header = (
    <Header
      title="Activité"
      large
      right={unread > 0 ? <Button label="Tout marquer lu" variant="ghost" compact onPress={readAll} testID="activity-read-all" /> : undefined}
    />
  );

  if (query.isLoading) {
    return (
      <Screen>
        {header}
        <LoadingState />
      </Screen>
    );
  }
  if (query.error && !items.length) {
    return (
      <Screen>
        {header}
        <ErrorState message={errorMessage(query.error)} onRetry={() => query.refetch()} />
      </Screen>
    );
  }

  return (
    <Screen padded={false} testID="activity-screen">
      <FlatList
        data={items}
        keyExtractor={(n) => n.id}
        contentContainerStyle={styles.list}
        ListHeaderComponent={header}
        refreshControl={<RefreshControl refreshing={query.isRefetching} onRefresh={() => query.refetch()} tintColor={colors.accent} />}
        onEndReached={() => query.hasNextPage && !query.isFetchingNextPage && query.fetchNextPage()}
        ListEmptyComponent={<EmptyState icon="notifications-outline" title="Rien de neuf" message="Tes nouveaux matchs et messages apparaîtront ici." />}
        renderItem={({ item }) => (
          <Pressable
            onPress={() => open(item)}
            style={({ pressed }) => [styles.row, !item.isRead && styles.unread, pressed && { opacity: 0.75 }]}
            accessibilityRole="button"
            testID={`notification-${item.type}`}
          >
            <View style={[styles.icon, { backgroundColor: item.type === "match" ? colors.roseSoft : colors.accentSoft }]}>
              <Ionicons name={item.type === "match" ? "heart" : "chatbubble"} size={18} color={item.type === "match" ? colors.rose : colors.accent} />
            </View>
            <View style={styles.texts}>
              <Text variant="bodyStrong">{item.title}</Text>
              <Text variant="caption" tone="muted">
                {item.body}
              </Text>
            </View>
            <View style={styles.side}>
              <Text variant="caption" tone="subtle">
                {relativeTime(item.updatedAt)}
              </Text>
              {!item.isRead ? <View style={styles.dot} accessibilityLabel="Non lu" /> : null}
            </View>
          </Pressable>
        )}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  list: {
    paddingHorizontal: spacing.xl,
    paddingTop: spacing.lg,
    paddingBottom: 120,
    gap: spacing.sm,
    maxWidth: 600,
    width: "100%",
    alignSelf: "center",
    flexGrow: 1
  },
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    padding: spacing.md,
    borderRadius: radii.lg,
    backgroundColor: colors.surface,
    borderWidth: 1,
    borderColor: colors.border
  },
  unread: { borderColor: "rgba(240,138,93,0.4)" },
  icon: { width: 40, height: 40, borderRadius: 20, alignItems: "center", justifyContent: "center" },
  texts: { flex: 1, gap: 2 },
  side: { alignItems: "flex-end", gap: 6 },
  dot: { width: 8, height: 8, borderRadius: 4, backgroundColor: colors.accent }
});
