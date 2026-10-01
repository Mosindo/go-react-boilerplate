import React from "react";
import { FlatList, RefreshControl, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { notificationApi } from "../api/platform";
import { queryKeys } from "../api/queryClient";
import type { AppNotification } from "../api/types";
import { useNotifications } from "../hooks/useData";
import type { RootStackParamList } from "../navigation/types";
import { EmptyView, ErrorView, LoadingView } from "../shared/feedback";
import { SafeAreaLayout } from "../shared/layout";
import { Button, NotificationItem, Text, spacing } from "../shared/ui";
import { formatRelativeTime } from "../utils/format";

type Nav = NativeStackNavigationProp<RootStackParamList>;

export default function NotificationsScreen() {
  const navigation = useNavigation<Nav>();
  const queryClient = useQueryClient();
  const { data, isLoading, isError, isRefetching, refetch } = useNotifications();

  const markRead = useMutation({
    mutationFn: (id: string) => notificationApi.markRead(id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.notifications })
  });
  const markAll = useMutation({
    mutationFn: notificationApi.markAllRead,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.notifications })
  });

  const open = (n: AppNotification) => {
    if (!n.isRead) {
      markRead.mutate(n.id);
    }
    if ((n.type === "match" || n.type === "message") && n.data.conversationId) {
      navigation.navigate("Conversation", { conversationId: n.data.conversationId, userId: n.data.userId });
    }
  };

  if (isLoading) {
    return (
      <SafeAreaLayout edges={["top"]}>
        <LoadingView label="Chargement…" />
      </SafeAreaLayout>
    );
  }
  if (isError || !data) {
    return (
      <SafeAreaLayout edges={["top"]}>
        <ErrorView message="Impossible de charger vos notifications." onAction={() => void refetch()} />
      </SafeAreaLayout>
    );
  }

  return (
    <SafeAreaLayout edges={["top"]}>
      <FlatList
        testID="notifications-screen"
        ItemSeparatorComponent={() => <View style={styles.gap} />}
        ListEmptyComponent={
          <EmptyView icon="notifications-outline" message="Vos matchs et nouveaux messages apparaîtront ici." title="Rien de neuf" />
        }
        ListHeaderComponent={
          <View style={styles.header}>
            <Text accessibilityRole="header" variant="title">
              Notifications
            </Text>
            {data.unreadCount > 0 ? (
              <Button
                fullWidth={false}
                label="Tout marquer comme lu"
                loading={markAll.isPending}
                onPress={() => markAll.mutate()}
                size="sm"
                variant="outline"
              />
            ) : null}
          </View>
        }
        contentContainerStyle={styles.content}
        data={data.notifications}
        keyExtractor={(n) => n.id}
        refreshControl={<RefreshControl onRefresh={() => void refetch()} refreshing={isRefetching && !isLoading} />}
        renderItem={({ item }) => (
          <NotificationItem
            body={item.body}
            kind={item.type}
            onPress={() => open(item)}
            time={formatRelativeTime(item.createdAt)}
            title={item.title}
            unread={!item.isRead}
          />
        )}
      />
    </SafeAreaLayout>
  );
}

const styles = StyleSheet.create({
  content: { padding: spacing.lg, flexGrow: 1 },
  header: { gap: spacing.sm, paddingBottom: spacing.md, alignItems: "flex-start" },
  gap: { height: spacing.sm }
});
