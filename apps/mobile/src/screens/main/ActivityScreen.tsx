import React from "react";
import { FlatList, Pressable, RefreshControl, StyleSheet, View } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import { notificationsApi } from "../../api/endpoints";
import type { AppNotification } from "../../api/types";
import { useNotifications } from "../../hooks/queries";
import { queryKeys } from "../../hooks/queryClient";
import { useRealtime } from "../../hooks/useRealtime";
import type { TabScreenProps } from "../../navigation/types";
import { layout, spacing, useTheme } from "../../theme";
import { Button, EmptyState, ErrorState, Header, Icon, LoadingState, Screen, Text } from "../../ui";
import { formatShortDate } from "../../utils/dates";

export default function ActivityScreen({ navigation }: TabScreenProps<"Activity">) {
  const { colors } = useTheme();
  const qc = useQueryClient();
  const { connected } = useRealtime();
  const query = useNotifications(connected ? false : 20_000);

  const markAll = async () => {
    await notificationsApi.markAllRead();
    await qc.invalidateQueries({ queryKey: queryKeys.notifications });
  };

  const open = async (n: AppNotification) => {
    if (!n.isRead) {
      void notificationsApi.markRead(n.id).then(() => qc.invalidateQueries({ queryKey: queryKeys.notifications }));
    }
    const userId = n.data.userId;
    if (!userId) {
      return;
    }
    navigation.navigate("Chat", { userId, firstName: n.data.firstName ?? "Votre match" });
  };

  const renderItem = ({ item }: { item: AppNotification }) => (
    <Pressable
      accessibilityRole="button"
      onPress={() => void open(item)}
      style={({ pressed }) => [
        styles.row,
        { backgroundColor: pressed ? colors.surfaceAlt : item.isRead ? "transparent" : colors.accentSoft }
      ]}
      testID={`notification-${item.id}`}
    >
      <View style={[styles.icon, { backgroundColor: colors.surface }]}>
        <Icon color="accent" name={item.type === "match.created" ? "heart" : "chatbubble-ellipses"} size={20} />
      </View>
      <View style={styles.text}>
        <View style={styles.top}>
          <Text variant="bodyStrong">{item.title}</Text>
          <Text color="textSubtle" variant="caption">
            {formatShortDate(item.createdAt)}
          </Text>
        </View>
        <Text color="textMuted">{item.body}</Text>
      </View>
      {!item.isRead ? <View accessibilityLabel="Non lu" style={[styles.dot, { backgroundColor: colors.accent }]} /> : null}
    </Pressable>
  );

  const unread = query.data?.unreadCount ?? 0;

  return (
    <Screen testID="activity-screen">
      <Header
        right={
          unread > 0 ? (
            <Button
              fullWidth={false}
              label="Tout lire"
              onPress={() => void markAll()}
              size="sm"
              testID="activity-read-all"
              variant="ghost"
            />
          ) : undefined
        }
        title="Activité"
      />
      {query.isLoading ? (
        <LoadingState />
      ) : query.isError ? (
        <ErrorState message="Impossible de charger vos notifications." onRetry={() => void query.refetch()} />
      ) : (query.data?.notifications.length ?? 0) === 0 ? (
        <EmptyState icon="notifications-outline" message="Vos nouveaux matchs et messages s’afficheront ici." title="Rien de neuf" />
      ) : (
        <FlatList
          contentContainerStyle={styles.list}
          data={query.data?.notifications}
          keyExtractor={(n) => n.id}
          refreshControl={
            <RefreshControl onRefresh={() => void query.refetch()} refreshing={query.isRefetching} tintColor={colors.accent} />
          }
          renderItem={renderItem}
        />
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  list: { gap: spacing.xs, paddingBottom: 110, width: "100%", maxWidth: layout.maxContentWidth, alignSelf: "center" },
  row: { flexDirection: "row", alignItems: "center", gap: spacing.md, padding: spacing.sm, borderRadius: 16 },
  icon: { width: 44, height: 44, borderRadius: 22, alignItems: "center", justifyContent: "center" },
  text: { flex: 1, gap: 2 },
  top: { flexDirection: "row", justifyContent: "space-between", gap: spacing.sm },
  dot: { width: 10, height: 10, borderRadius: 5 }
});
