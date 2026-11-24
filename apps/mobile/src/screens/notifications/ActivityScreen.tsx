import React, { useMemo } from "react";
import { FlatList, Pressable, RefreshControl, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import { useQueryClient } from "@tanstack/react-query";
import Ionicons from "@expo/vector-icons/Ionicons";
import { notificationsApi } from "../../api/endpoints";
import type { AppNotification } from "../../api/types";
import { AppText } from "../../components/AppText";
import { Button } from "../../components/Button";
import { Header } from "../../components/Header";
import { Screen } from "../../components/Screen";
import { EmptyState, ErrorState, LoadingState } from "../../components/States";
import { useNotifications } from "../../hooks/queries";
import { queryKeys } from "../../queryClient";
import { colors, radii, spacing } from "../../theme/tokens";
import { errorMessage } from "../../utils/errors";
import { relativeTime } from "../../utils/format";

export default function ActivityScreen() {
  const navigation = useNavigation();
  const client = useQueryClient();
  const query = useNotifications();
  const items = useMemo(() => query.data?.pages.flatMap((p) => p.notifications) ?? [], [query.data]);
  const unread = query.data?.pages[0]?.unreadCount ?? 0;

  const refreshCounts = () => {
    void client.invalidateQueries({ queryKey: queryKeys.notifications });
  };

  const open = async (n: AppNotification) => {
    if (!n.isRead) {
      notificationsApi.markRead(n.id).then(refreshCounts).catch(() => undefined);
    }
    const conversationId = typeof n.data.conversationId === "string" ? n.data.conversationId : null;
    const matchId = typeof n.data.matchId === "string" ? n.data.matchId : null;
    const userId = typeof n.data.userId === "string" ? n.data.userId : null;
    const firstName = typeof n.data.firstName === "string" ? n.data.firstName : "";
    if (n.type === "match" && conversationId && matchId && userId) {
      navigation.navigate("Chat", { conversationId, matchId, userId, name: firstName, photoUrl: null });
    }
  };

  const markAll = async () => {
    await notificationsApi.markAllRead().catch(() => undefined);
    refreshCounts();
  };

  return (
    <Screen testID="activity-screen">
      <Header title="Activité" right={unread > 0 ? <Button label="Tout lire" variant="ghost" compact onPress={markAll} testID="activity-read-all" /> : undefined} />
      {query.isPending ? (
        <LoadingState />
      ) : query.isError ? (
        <ErrorState message={errorMessage(query.error)} onRetry={() => void query.refetch()} />
      ) : (
        <FlatList
          data={items}
          keyExtractor={(n) => n.id}
          contentContainerStyle={styles.list}
          ListEmptyComponent={<EmptyState icon="notifications-outline" title="Rien de neuf" message="Tes nouveaux matchs apparaîtront ici." />}
          renderItem={({ item }) => (
            <Pressable onPress={() => void open(item)} style={({ pressed }) => [styles.row, !item.isRead && styles.unread, pressed && styles.pressed]} accessibilityRole="button" testID={`notification-${item.type}`}>
              <View style={styles.icon}>
                <Ionicons name={item.type === "match" ? "sparkles" : "notifications"} size={20} color={colors.primary} />
              </View>
              <View style={styles.text}>
                <AppText variant="bodyStrong">{item.title}</AppText>
                <AppText color={colors.textMuted}>{item.body}</AppText>
                <AppText variant="caption" color={colors.textSubtle}>
                  {relativeTime(item.createdAt)}
                </AppText>
              </View>
              {!item.isRead ? <View style={styles.dot} accessibilityLabel="Non lu" /> : null}
            </Pressable>
          )}
          onEndReached={() => {
            if (query.hasNextPage && !query.isFetchingNextPage) void query.fetchNextPage();
          }}
          refreshControl={<RefreshControl refreshing={query.isRefetching} onRefresh={() => void query.refetch()} tintColor={colors.primary} />}
        />
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  list: { padding: spacing.lg, gap: spacing.sm, flexGrow: 1 },
  row: { flexDirection: "row", gap: spacing.md, padding: spacing.md, borderRadius: radii.md, backgroundColor: colors.surface, borderWidth: 1, borderColor: colors.border, alignItems: "flex-start" },
  unread: { borderColor: colors.primary },
  pressed: { backgroundColor: colors.surfacePressed },
  icon: { width: 40, height: 40, borderRadius: 20, backgroundColor: colors.primarySoft, alignItems: "center", justifyContent: "center" },
  text: { flex: 1, gap: 2 },
  dot: { width: 10, height: 10, borderRadius: 5, backgroundColor: colors.primary, marginTop: spacing.xs }
});
