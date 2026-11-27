import React, { useMemo } from "react";
import { FlatList, Pressable, RefreshControl, StyleSheet, View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { notificationsApi } from "../../api/endpoints";
import type { AppNotification } from "../../api/types";
import { useNotifications } from "../../hooks/queries";
import { keys, queryClient } from "../../hooks/queryClient";
import { useTheme } from "../../theme/ThemeProvider";
import { spacing } from "../../theme/palette";
import { Button, EmptyState, ErrorState, Header, Icon, Loading, Screen, Text } from "../../ui";
import { formatRelativeShort } from "../../utils/format";
import type { TabScreenProps } from "../../navigation/types";

function refresh() {
  void queryClient.invalidateQueries({ queryKey: keys.notifications });
}

export function NotificationsScreen({ navigation }: TabScreenProps<"Notifications">) {
  const { colors } = useTheme();
  const query = useNotifications();
  const items = useMemo(() => query.data?.pages.flatMap((p) => p.notifications) ?? [], [query.data]);
  const markAll = useMutation({ mutationFn: notificationsApi.markAllRead, onSuccess: refresh });
  const markOne = useMutation({ mutationFn: notificationsApi.markRead, onSuccess: refresh });

  const open = (n: AppNotification) => {
    if (!n.isRead) {
      markOne.mutate(n.id);
    }
    if (n.type === "match" && n.data.matchId) {
      navigation.navigate("Chat", { matchId: n.data.matchId });
    }
  };

  const hasUnread = items.some((n) => !n.isRead);

  return (
    <Screen padded={false} testID="notifications-screen">
      <View style={styles.pad}>
        <Header
          title="Activité"
          large
          right={hasUnread ? <Button label="Tout lire" variant="ghost" size="sm" loading={markAll.isPending} onPress={() => markAll.mutate()} /> : null}
        />
      </View>
      {query.isLoading ? (
        <Loading />
      ) : query.error && items.length === 0 ? (
        <ErrorState message={errorMessage(query.error)} onRetry={() => void query.refetch()} />
      ) : items.length === 0 ? (
        <EmptyState icon="notifications-outline" title="Rien de neuf" message="Tes nouveaux matchs apparaîtront ici." />
      ) : (
        <FlatList
          data={items}
          keyExtractor={(n) => n.id}
          contentContainerStyle={styles.list}
          refreshControl={<RefreshControl refreshing={query.isRefetching} onRefresh={() => void query.refetch()} tintColor={colors.primary} />}
          onEndReachedThreshold={0.4}
          onEndReached={() => {
            if (query.hasNextPage && !query.isFetchingNextPage) void query.fetchNextPage();
          }}
          renderItem={({ item }) => (
            <Pressable
              onPress={() => open(item)}
              accessibilityRole="button"
              accessibilityLabel={`${item.title}. ${item.body}${item.isRead ? "" : ". Non lue"}`}
              style={({ pressed }) => [styles.row, { backgroundColor: pressed ? colors.surfaceMuted : item.isRead ? "transparent" : colors.primarySoft }]}
              testID={`notification-${item.type}`}
            >
              <View style={[styles.icon, { backgroundColor: colors.surface }]}>
                <Icon name={item.type === "match" ? "heart" : "notifications"} size={20} color={colors.primary} />
              </View>
              <View style={styles.flex}>
                <Text variant="bodyStrong">{item.title}</Text>
                <Text variant="caption" tone="muted">
                  {item.body}
                </Text>
              </View>
              <Text variant="caption" tone="subtle">
                {formatRelativeShort(item.createdAt)}
              </Text>
            </Pressable>
          )}
        />
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  pad: { paddingHorizontal: spacing.lg },
  flex: { flex: 1 },
  list: { paddingBottom: spacing.xxl, maxWidth: 720, width: "100%", alignSelf: "center" },
  row: { flexDirection: "row", alignItems: "center", gap: spacing.sm, paddingHorizontal: spacing.lg, paddingVertical: spacing.md },
  icon: { width: 40, height: 40, borderRadius: 20, alignItems: "center", justifyContent: "center" }
});
