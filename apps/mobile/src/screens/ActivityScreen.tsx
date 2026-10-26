import React from "react";
import { FlatList, Pressable, RefreshControl, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { notificationsApi, type AppNotification } from "../api";
import { Button } from "../components/Button";
import { Avatar } from "../components/Photo";
import { Screen } from "../components/Screen";
import { EmptyState, ErrorState, LoadingState } from "../components/States";
import { Text } from "../components/Text";
import { useNotifications } from "../hooks/useQueries";
import { timeLabel } from "../lib/format";
import { queryKeys } from "../lib/queryKeys";
import type { RootStackParamList } from "../navigation/types";
import { radius, spacing, useTheme } from "../theme";

export function ActivityScreen() {
  const theme = useTheme();
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>();
  const queryClient = useQueryClient();
  const query = useNotifications();
  const items = query.data?.pages.flatMap((p) => p.notifications) ?? [];
  const unread = query.data?.pages[0]?.unreadCount ?? 0;

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
    void queryClient.invalidateQueries({ queryKey: queryKeys.unread });
  };
  const readOne = useMutation({ mutationFn: notificationsApi.markRead, onSuccess: invalidate });
  const readAll = useMutation({ mutationFn: notificationsApi.markAllRead, onSuccess: invalidate });

  const open = (n: AppNotification) => {
    if (!n.readAt) readOne.mutate(n.id);
    if (n.matchId && n.actor) {
      navigation.navigate("Tabs", { screen: "Messages" });
    }
  };

  if (query.isPending)
    return (
      <Screen>
        <LoadingState />
      </Screen>
    );
  if (query.isError)
    return (
      <Screen>
        <ErrorState message={query.error.message} onRetry={() => void query.refetch()} />
      </Screen>
    );

  return (
    <Screen>
      <View style={styles.header}>
        <Text variant="title">Activité</Text>
        {unread > 0 ? (
          <Button
            label="Tout marquer comme lu"
            variant="ghost"
            small
            onPress={() => readAll.mutate()}
            testID="read-all"
          />
        ) : null}
      </View>
      {items.length === 0 ? (
        <EmptyState
          emoji="🔔"
          title="Rien de nouveau"
          message="Vos nouveaux matchs et messages apparaîtront ici."
        />
      ) : (
        <FlatList
          data={items}
          keyExtractor={(n) => n.id}
          refreshControl={
            <RefreshControl
              refreshing={query.isRefetching && !query.isFetchingNextPage}
              onRefresh={() => void query.refetch()}
            />
          }
          onEndReached={() => query.hasNextPage && !query.isFetchingNextPage && void query.fetchNextPage()}
          renderItem={({ item }) => {
            const name = item.actor?.firstName ?? "Quelqu'un";
            const text =
              item.type === "match"
                ? `Vous avez un nouveau match avec ${name} !`
                : `${name} vous a envoyé un message.`;
            return (
              <Pressable
                onPress={() => open(item)}
                accessibilityRole="button"
                accessibilityLabel={`${text}${item.readAt ? "" : " Non lu."}`}
                style={({ pressed }) => [
                  styles.row,
                  {
                    backgroundColor: item.readAt ? "transparent" : theme.primarySoft,
                    opacity: pressed ? 0.8 : 1
                  }
                ]}
                testID={`notification-${item.type}`}
              >
                <Avatar url={item.actor?.photoUrl} name={name} size={48} />
                <View style={styles.text}>
                  <Text variant={item.readAt ? "body" : "bodyStrong"}>{text}</Text>
                  <Text variant="caption" muted>
                    {timeLabel(item.createdAt)}
                  </Text>
                </View>
              </Pressable>
            );
          }}
        />
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  header: { flexDirection: "row", alignItems: "center", justifyContent: "space-between" },
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    padding: spacing.md,
    borderRadius: radius.md,
    marginTop: spacing.xs
  },
  text: { flex: 1, gap: 2 }
});
