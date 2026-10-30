import React from "react";
import { FlatList, Pressable, RefreshControl, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useQueryClient } from "@tanstack/react-query";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { notificationsApi } from "../api/endpoints";
import type { AppNotification } from "../api/types";
import { errorMessage, keys, useConversations, useNotifications } from "../hooks/queries";
import { timeLabel } from "../lib/format";
import type { MainStackParamList } from "../navigation/types";
import { Button, EmptyView, ErrorView, LoadingView, Screen, Text, radii, spacing, useTheme, useToast } from "../ui";

type Nav = NativeStackNavigationProp<MainStackParamList>;

export default function NotificationsScreen() {
  const { colors } = useTheme();
  const navigation = useNavigation<Nav>();
  const toast = useToast();
  const queryClient = useQueryClient();
  const query = useNotifications();
  const conversations = useConversations();

  const refresh = () => queryClient.invalidateQueries({ queryKey: keys.notifications });

  const open = async (n: AppNotification) => {
    try {
      if (!n.isRead) {
        await notificationsApi.markRead(n.id);
        void refresh();
      }
    } catch (error) {
      toast.show(errorMessage(error), "error");
    }
    if (n.data.conversationId && n.data.userId) {
      navigation.navigate("Chat", {
        conversationId: n.data.conversationId,
        userId: n.data.userId,
        name: conversations.data?.find((c) => c.id === n.data.conversationId)?.user.displayName ?? "Conversation"
      });
    }
  };

  const markAll = async () => {
    try {
      await notificationsApi.markAllRead();
      await refresh();
    } catch (error) {
      toast.show(errorMessage(error), "error");
    }
  };

  if (query.isLoading) return <LoadingView />;
  if (query.isError) return <ErrorView message={errorMessage(query.error)} onRetry={() => void query.refetch()} />;

  const items = query.data?.notifications ?? [];
  const unread = query.data?.unreadCount ?? 0;

  return (
    <Screen topInset testID="notifications-screen">
      <View style={styles.header}>
        <Text variant="title">Notifications</Text>
        {unread > 0 ? (
          <Button label="Tout lire" onPress={markAll} small testID="notifications-read-all" variant="ghost" />
        ) : null}
      </View>
      {items.length === 0 ? (
        <EmptyView
          icon="notifications-outline"
          message="Vos matchs et nouveaux messages apparaîtront ici."
          title="Rien de nouveau"
          testID="notifications-empty"
        />
      ) : (
        <FlatList
          data={items}
          keyExtractor={(n) => n.id}
          refreshControl={
            <RefreshControl
              onRefresh={() => void query.refetch()}
              refreshing={query.isRefetching}
              tintColor={colors.primary}
            />
          }
          renderItem={({ item }) => (
            <Pressable
              accessibilityRole="button"
              onPress={() => void open(item)}
              style={[styles.row, { backgroundColor: item.isRead ? "transparent" : colors.primarySoft }]}
              testID="notification-row"
            >
              <View style={[styles.icon, { backgroundColor: colors.surface }]}>
                <Ionicons color={colors.primary} name={item.type === "match" ? "heart" : "chatbubble"} size={20} />
              </View>
              <View style={styles.text}>
                <Text variant="label">{item.title}</Text>
                <Text tone="muted">{item.body}</Text>
              </View>
              <Text tone="muted" variant="caption">
                {timeLabel(item.createdAt)}
              </Text>
            </Pressable>
          )}
        />
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  header: { flexDirection: "row", alignItems: "center", justifyContent: "space-between", marginBottom: spacing.md },
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    padding: spacing.md,
    borderRadius: radii.md,
    marginBottom: spacing.xs,
    minHeight: 64
  },
  icon: { width: 40, height: 40, borderRadius: 20, alignItems: "center", justifyContent: "center" },
  text: { flex: 1 }
});
