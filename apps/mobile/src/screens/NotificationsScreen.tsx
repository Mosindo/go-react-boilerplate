import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useQueryClient } from "@tanstack/react-query";
import React from "react";
import { FlatList, Pressable, RefreshControl, StyleSheet, View } from "react-native";
import { errorMessage } from "../api/client";
import { datingApi } from "../api/dating";
import type { AppNotification } from "../api/types";
import { keys, useNotifications } from "../hooks/queries";
import { formatListTime } from "../lib/format";
import type { MainStackParamList } from "../navigation/types";
import { spacing, useTheme } from "../theme";
import { Avatar, Button, EmptyState, ErrorState, LoadingState, Screen, Text, useToast } from "../ui";

export default function NotificationsScreen() {
  const { colors } = useTheme();
  const navigation = useNavigation<NativeStackNavigationProp<MainStackParamList>>();
  const client = useQueryClient();
  const toast = useToast();
  const { data, isLoading, isError, error, refetch, isRefetching } = useNotifications();

  const open = async (n: AppNotification) => {
    try {
      if (!n.readAt) {
        await datingApi.markNotificationRead(n.id);
        void client.invalidateQueries({ queryKey: keys.notifications });
      }
    } catch (e) {
      toast.show(errorMessage(e), "error");
    }
    if (n.matchId && n.actor) {
      navigation.navigate("Chat", { matchId: n.matchId, userId: n.actor.userId, name: n.actor.firstName });
    }
  };

  const readAll = async () => {
    try {
      await datingApi.markAllNotificationsRead();
      await client.invalidateQueries({ queryKey: keys.notifications });
    } catch (e) {
      toast.show(errorMessage(e), "error");
    }
  };

  if (isLoading) {
    return (
      <Screen>
        <LoadingState />
      </Screen>
    );
  }
  if (isError || !data) {
    return (
      <Screen>
        <ErrorState message={errorMessage(error)} onRetry={() => void refetch()} />
      </Screen>
    );
  }

  return (
    <Screen padded={false}>
      <FlatList
        data={data.items}
        keyExtractor={(n) => n.id}
        refreshControl={<RefreshControl refreshing={isRefetching} onRefresh={() => void refetch()} tintColor={colors.primary} />}
        ListHeaderComponent={
          <View style={styles.header}>
            <Text variant="title" style={styles.flex}>
              Notifications
            </Text>
            {data.unreadCount > 0 ? <Button label="Tout lire" variant="ghost" onPress={() => void readAll()} /> : null}
          </View>
        }
        ListEmptyComponent={
          <EmptyState icon="notifications-outline" title="Rien de nouveau" message="Vos nouveaux matchs et messages apparaîtront ici." />
        }
        renderItem={({ item }) => {
          const unread = !item.readAt;
          const name = item.actor?.firstName ?? "Quelqu'un";
          const label = item.kind === "match" ? `Nouveau match avec ${name} 🎉` : `${name} vous a envoyé un message`;
          return (
            <Pressable
              accessibilityRole="button"
              accessibilityLabel={`${label}${unread ? ", non lu" : ""}`}
              onPress={() => void open(item)}
              style={({ pressed }) => [
                styles.row,
                { backgroundColor: pressed ? colors.surfaceMuted : unread ? colors.primarySoft : "transparent" }
              ]}
            >
              <Avatar uri={item.actor?.thumbUrl} name={name} size={48} />
              <View style={styles.flex}>
                <Text style={unread ? styles.bold : undefined}>{label}</Text>
                <Text variant="caption" tone="muted">
                  {formatListTime(item.createdAt)}
                </Text>
              </View>
              {unread ? <View style={[styles.dot, { backgroundColor: colors.primary }]} /> : null}
            </Pressable>
          );
        }}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  header: { flexDirection: "row", alignItems: "center", padding: spacing.lg },
  row: { flexDirection: "row", alignItems: "center", gap: spacing.md, paddingHorizontal: spacing.lg, paddingVertical: spacing.md },
  bold: { fontWeight: "700" },
  dot: { width: 10, height: 10, borderRadius: 5 }
});
