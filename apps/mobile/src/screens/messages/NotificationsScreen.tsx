import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import React from "react";
import { FlatList, RefreshControl, StyleSheet, View } from "react-native";
import { markNotificationsRead } from "../../api/notifications";
import type { AppNotification } from "../../api/types";
import { keys, useNotifications } from "../../hooks/queries";
import type { RootStackParamList } from "../../navigation/types";
import { EmptyView, ErrorView, LoadingView } from "../../shared/feedback";
import { SafeAreaLayout } from "../../shared/layout";
import { Button, NotificationItem } from "../../shared/ui";
import { spacing } from "../../theme";

type Nav = NativeStackNavigationProp<RootStackParamList>;

export function NotificationsScreen() {
  const navigation = useNavigation<Nav>();
  const queryClient = useQueryClient();
  const query = useNotifications();
  const markRead = useMutation({
    mutationFn: (ids?: string[]) => markNotificationsRead(ids),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: keys.notifications })
  });

  const open = (n: AppNotification) => {
    if (!n.readAt) markRead.mutate([n.id]);
    if (n.conversationId) navigation.navigate("Chat", { conversationId: n.conversationId, userId: n.actor.id, name: n.actor.firstName });
  };

  const items = query.data?.items ?? [];
  let body: React.ReactNode;
  if (query.isPending) body = <LoadingView />;
  else if (query.isError) body = <ErrorView message={query.error.message} onRetry={() => void query.refetch()} />;
  else if (items.length === 0)
    body = (
      <EmptyView
        testID="notifications-empty"
        icon="notifications-outline"
        title="Rien de nouveau"
        message="Vos nouveaux matchs et messages apparaîtront ici."
      />
    );
  else {
    body = (
      <FlatList
        data={items}
        keyExtractor={(n) => n.id}
        refreshControl={<RefreshControl refreshing={query.isRefetching} onRefresh={() => void query.refetch()} />}
        renderItem={({ item }) => (
          <NotificationItem
            testID={`notification-${item.type}`}
            title={item.type === "match" ? `Nouveau match avec ${item.actor.firstName}` : `Nouveau message de ${item.actor.firstName}`}
            subtitle={item.type === "match" ? "Dites bonjour !" : "Touchez pour répondre"}
            actorName={item.actor.firstName}
            actorThumb={item.actor.thumbUrl}
            createdAt={item.createdAt}
            unread={!item.readAt}
            onPress={() => open(item)}
          />
        )}
      />
    );
  }

  return (
    <SafeAreaLayout edges={["bottom", "left", "right"]}>
      <View style={styles.container}>
        {(query.data?.unreadCount ?? 0) > 0 ? (
          <Button
            testID="notifications-mark-all"
            label="Tout marquer comme lu"
            variant="secondary"
            onPress={() => markRead.mutate(undefined)}
            loading={markRead.isPending}
          />
        ) : null}
        <View style={styles.flex}>{body}</View>
      </View>
    </SafeAreaLayout>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, width: "100%", maxWidth: 640, alignSelf: "center", padding: spacing.lg, gap: spacing.md },
  flex: { flex: 1 }
});
