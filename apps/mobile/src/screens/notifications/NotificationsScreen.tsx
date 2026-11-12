import React from "react";
import { FlatList, Pressable, RefreshControl, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { notificationsApi } from "../../api/endpoints";
import type { AppNotification } from "../../api/types";
import { keys, useMarkNotificationsRead, useNotifications } from "../../hooks/queries";
import type { AppStackParamList } from "../../navigation/types";
import { spacing, useTheme } from "../../theme";
import { Button, EmptyState, ErrorState, Icon, Loading, Screen, Text } from "../../ui";
import { relativeTime } from "../../utils/format";

export default function NotificationsScreen() {
  const { colors } = useTheme();
  const navigation = useNavigation<NativeStackNavigationProp<AppStackParamList>>();
  const qc = useQueryClient();
  const query = useNotifications();
  const markAll = useMarkNotificationsRead();

  const open = async (n: AppNotification) => {
    if (!n.isRead) {
      await notificationsApi.markRead(n.id).catch(() => undefined);
      void qc.invalidateQueries({ queryKey: keys.notifications });
    }
    const userId = n.data.userId;
    if (userId) {
      // Match and message notifications both lead to the conversation.
      navigation.navigate("Chat", { userId, firstName: n.type === "message" ? n.title.replace(/^Nouveau message de /, "") : "" });
    }
  };

  if (query.isLoading) return <Loading />;
  if (query.error) {
    return (
      <Screen>
        <ErrorState message={errorMessage(query.error)} onRetry={() => void query.refetch()} />
      </Screen>
    );
  }
  const items = query.data?.notifications ?? [];
  const unread = query.data?.unreadCount ?? 0;

  return (
    <Screen padded={false}>
      <View style={styles.header}>
        <Text variant="title" serif accessibilityRole="header">
          Activité
        </Text>
        {unread > 0 ? <Button label="Tout marquer comme lu" variant="ghost" compact onPress={() => markAll.mutate()} loading={markAll.isPending} /> : null}
      </View>
      {items.length === 0 ? (
        <EmptyState icon="notifications-outline" title="Rien de neuf pour l'instant" message="Vos nouveaux matchs et messages apparaîtront ici." />
      ) : (
        <FlatList
          data={items}
          keyExtractor={(n) => n.id}
          refreshControl={<RefreshControl refreshing={query.isRefetching} onRefresh={() => void query.refetch()} tintColor={colors.primary} />}
          renderItem={({ item }) => (
            <Pressable
              accessibilityRole="button"
              accessibilityLabel={`${item.title}. ${item.body}${item.isRead ? "" : ". Non lu"}`}
              onPress={() => void open(item)}
              style={({ pressed }) => [styles.row, { backgroundColor: pressed ? colors.surfaceMuted : item.isRead ? "transparent" : colors.primarySoft }]}
            >
              <View style={[styles.icon, { backgroundColor: item.type === "match" ? colors.primary : colors.accentSoft }]}>
                <Icon
                  name={item.type === "match" ? "heart" : "chatbubble-ellipses"}
                  size={18}
                  color={item.type === "match" ? colors.onPrimary : colors.accent}
                />
              </View>
              <View style={styles.text}>
                <Text variant="bodyStrong">{item.title}</Text>
                <Text tone="muted" numberOfLines={2}>
                  {item.body}
                </Text>
              </View>
              <Text variant="caption" tone="subtle">
                {relativeTime(item.createdAt)}
              </Text>
            </Pressable>
          )}
        />
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  header: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.xs,
    minHeight: 52
  },
  row: { flexDirection: "row", gap: spacing.sm, alignItems: "flex-start", paddingHorizontal: spacing.lg, paddingVertical: spacing.md },
  icon: { width: 36, height: 36, borderRadius: 18, alignItems: "center", justifyContent: "center" },
  text: { flex: 1, gap: 2 }
});
