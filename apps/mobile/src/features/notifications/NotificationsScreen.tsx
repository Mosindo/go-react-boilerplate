import React, { useMemo } from "react";
import { FlatList, Pressable, RefreshControl, StyleSheet, View } from "react-native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { matchesApi, notificationsApi } from "../../api/endpoints";
import type { AppNotification } from "../../api/types";
import { keys, useNotifications } from "../../hooks/queries";
import { formatRelative } from "../../lib/format";
import type { RootParamList } from "../../navigation/types";
import { spacing, useTheme } from "../../theme/theme";
import { Button } from "../../ui/Button";
import { EmptyState, ErrorState, LoadingView } from "../../ui/Feedback";
import { Avatar } from "../../ui/Photo";
import { Screen } from "../../ui/Screen";
import { Text } from "../../ui/Text";

type Props = NativeStackScreenProps<RootParamList, "Notifications">;

function describe(n: AppNotification): string {
  const name = n.actor?.firstName ?? "Quelqu'un";
  return n.type === "match" ? `Tu as un nouveau match avec ${name} !` : `${name} t'a écrit.`;
}

export function NotificationsScreen({ navigation }: Props) {
  const { colors } = useTheme();
  const queryClient = useQueryClient();
  const query = useNotifications();
  const items = useMemo(() => query.data?.pages.flatMap((p) => p.notifications) ?? [], [query.data]);
  const unread = query.data?.pages[0]?.unreadCount ?? 0;

  const refresh = () => void queryClient.invalidateQueries({ queryKey: keys.notifications });
  const readOne = useMutation({ mutationFn: notificationsApi.markRead, onSuccess: refresh });
  const readAll = useMutation({ mutationFn: notificationsApi.markAllRead, onSuccess: refresh });

  const open = (n: AppNotification) => {
    if (!n.readAt) readOne.mutate(n.id);
    if (!n.refId || !n.actor) return;
    // refId is a match id (type "match") or a conversation id (type "message").
    void queryClient.fetchQuery({ queryKey: keys.matches, queryFn: () => matchesApi.list() }).then((page) => {
      const match = page.matches.find((m) =>
        n.type === "match" ? m.id === n.refId : m.conversationId === n.refId
      );
      if (match)
        navigation.navigate("Chat", {
          conversationId: match.conversationId,
          matchId: match.id,
          peer: match.user
        });
    });
  };

  if (query.isPending)
    return (
      <Screen>
        <LoadingView />
      </Screen>
    );
  if (query.isError)
    return (
      <Screen>
        <ErrorState message={query.error.message} onRetry={() => void query.refetch()} />
      </Screen>
    );

  return (
    <Screen padded={false} edges={["bottom"]}>
      {unread > 0 ? (
        <View style={styles.toolbar}>
          <Button
            title="Tout marquer comme lu"
            variant="ghost"
            onPress={() => readAll.mutate()}
            loading={readAll.isPending}
          />
        </View>
      ) : null}
      <FlatList
        data={items}
        keyExtractor={(n) => n.id}
        renderItem={({ item }) => (
          <Pressable
            accessibilityRole="button"
            accessibilityLabel={describe(item)}
            onPress={() => open(item)}
            style={({ pressed }) => [
              styles.row,
              {
                backgroundColor: pressed
                  ? colors.surfaceMuted
                  : item.readAt
                    ? "transparent"
                    : colors.primarySoft
              }
            ]}
          >
            <Avatar path={item.actor?.photoUrl} name={item.actor?.firstName ?? "?"} size={48} />
            <View style={styles.flex}>
              <Text style={item.readAt ? undefined : styles.bold}>{describe(item)}</Text>
              <Text variant="caption" tone="textSubtle">
                {formatRelative(item.createdAt)}
              </Text>
            </View>
          </Pressable>
        )}
        ListEmptyComponent={
          <EmptyState
            icon="notifications-off-outline"
            title="Rien de nouveau"
            message="Tes matchs et nouveaux messages apparaîtront ici."
          />
        }
        onEndReached={() => {
          if (query.hasNextPage && !query.isFetchingNextPage) void query.fetchNextPage();
        }}
        refreshControl={
          <RefreshControl refreshing={query.isRefetching} onRefresh={() => void query.refetch()} />
        }
        contentContainerStyle={items.length === 0 ? styles.grow : undefined}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  grow: { flexGrow: 1 },
  bold: { fontWeight: "600" },
  toolbar: { alignItems: "flex-end", paddingHorizontal: spacing.md },
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.sm,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm
  }
});
