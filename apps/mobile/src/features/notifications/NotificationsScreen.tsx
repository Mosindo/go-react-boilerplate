import React, { useMemo } from "react";
import { FlatList, Pressable, RefreshControl, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { Ionicons } from "@expo/vector-icons";
import { useQueryClient } from "@tanstack/react-query";
import { notificationsApi } from "../../api/endpoints";
import type { AppNotification } from "../../api/types";
import { keys, useNotifications } from "../../hooks/queries";
import type { MainStackParamList } from "../../navigation/types";
import { useTheme } from "../../theme/ThemeProvider";
import { EmptyState, ErrorState, Header, IconButton, LoadingState, Screen, Text } from "../../ui";
import { formatRelative } from "../../utils/format";

export function NotificationsScreen() {
  const theme = useTheme();
  const qc = useQueryClient();
  const navigation = useNavigation<NativeStackNavigationProp<MainStackParamList>>();
  const query = useNotifications();
  const items = useMemo(() => query.data?.pages.flatMap((p) => p.items) ?? [], [query.data]);
  const unread = query.data?.pages[0]?.unreadCount ?? 0;

  const refresh = () => qc.invalidateQueries({ queryKey: keys.notifications });

  const open = async (n: AppNotification) => {
    if (!n.readAt) {
      await notificationsApi.markRead(n.id).catch(() => undefined);
      void refresh();
    }
    if (n.matchId) navigation.navigate("Chat", { matchId: n.matchId, name: "" });
  };

  const markAll = async () => {
    await notificationsApi.markAllRead().catch(() => undefined);
    void refresh();
  };

  return (
    <Screen padded={false}>
      <View style={styles.pad}>
        <Header large right={unread ? <IconButton icon="checkmark-done-outline" label="Tout marquer comme lu" onPress={markAll} /> : null} title="Activité" />
      </View>
      {query.isLoading ? (
        <LoadingState />
      ) : query.error ? (
        <ErrorState onRetry={() => query.refetch()} />
      ) : items.length === 0 ? (
        <EmptyState icon="notifications-outline" message="Vos nouveaux matchs et messages apparaîtront ici." title="Rien de neuf" />
      ) : (
        <FlatList
          data={items}
          keyExtractor={(n) => n.id}
          onEndReached={() => query.hasNextPage && !query.isFetchingNextPage && query.fetchNextPage()}
          refreshControl={<RefreshControl onRefresh={() => query.refetch()} refreshing={query.isRefetching} tintColor={theme.colors.primary} />}
          renderItem={({ item }) => (
            <Pressable
              accessibilityLabel={`${item.body}${item.readAt ? "" : ", non lue"}`}
              onPress={() => open(item)}
              style={({ pressed }) => [styles.row, !item.readAt && { backgroundColor: theme.colors.primarySoft }, pressed && { opacity: 0.7 }]}
              testID={`notification-${item.type}`}
            >
              <View style={[styles.icon, { backgroundColor: item.type === "match" ? theme.colors.primary : theme.colors.accent }]}>
                <Ionicons color="#FFFFFF" name={item.type === "match" ? "heart" : "chatbubble"} size={18} />
              </View>
              <View style={styles.body}>
                <Text variant="small" weight={item.readAt ? undefined : "600"}>
                  {item.body}
                </Text>
                <Text tone="subtle" variant="caption">
                  {formatRelative(item.createdAt)}
                </Text>
              </View>
            </Pressable>
          )}
        />
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  pad: { paddingHorizontal: 20 },
  row: { flexDirection: "row", gap: 14, alignItems: "center", paddingHorizontal: 20, paddingVertical: 14 },
  icon: { width: 40, height: 40, borderRadius: 20, alignItems: "center", justifyContent: "center" },
  body: { flex: 1, gap: 2 }
});
