import React from "react";
import { Alert, FlatList, Pressable, StyleSheet, View } from "react-native";
import type { CompositeScreenProps } from "@react-navigation/native";
import type { BottomTabScreenProps } from "@react-navigation/bottom-tabs";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useInfiniteQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { fetchConversations, hideConversation } from "../../api/social";
import { queryKeys } from "../../api/keys";
import { formatTime } from "../../lib/format";
import { flattenPages } from "../../lib/pagination";
import type { MainStackParamList, MainTabParamList } from "../../navigation/types";
import { radii, spacing, useTheme } from "../../theme";
import { Avatar, EmptyState, ErrorState, LoadingState, Screen, Text, useToast } from "../../ui";

type Props = CompositeScreenProps<
  BottomTabScreenProps<MainTabParamList, "Chats">,
  NativeStackScreenProps<MainStackParamList>
>;

export function useConversations() {
  return useInfiniteQuery({
    queryKey: queryKeys.conversations,
    queryFn: ({ pageParam }) => fetchConversations(pageParam),
    initialPageParam: null as string | null,
    getNextPageParam: (last) => last.nextCursor
  });
}

export function ChatsScreen({ navigation }: Props) {
  const { colors } = useTheme();
  const toast = useToast();
  const qc = useQueryClient();
  const query = useConversations();
  const items = flattenPages(
    query.data?.pages.map((p) => ({ items: p.conversations, nextCursor: p.nextCursor }))
  );

  const hide = useMutation({
    mutationFn: (id: string) => hideConversation(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.conversations }),
    onError: (e) => toast.show(e instanceof Error ? e.message : "Could not remove the conversation.", "error")
  });

  return (
    <Screen>
      <Text variant="title">Messages</Text>
      {query.isLoading ? (
        <LoadingState />
      ) : query.isError ? (
        <ErrorState error={query.error} onRetry={() => void query.refetch()} />
      ) : items.length === 0 ? (
        <EmptyState title="No conversations yet" message="Match with someone and say hello." />
      ) : (
        <FlatList
          data={items}
          keyExtractor={(c) => c.id}
          onEndReached={() => query.hasNextPage && void query.fetchNextPage()}
          refreshing={query.isRefetching}
          onRefresh={() => void query.refetch()}
          renderItem={({ item }) => (
            <Pressable
              accessibilityRole="button"
              accessibilityLabel={`Conversation with ${item.user.firstName}${item.unreadCount ? `, ${item.unreadCount} unread` : ""}`}
              style={[styles.row, { borderBottomColor: colors.border }]}
              onPress={() =>
                navigation.navigate("Thread", {
                  conversationId: item.id,
                  userId: item.user.userId,
                  name: item.user.firstName
                })
              }
              onLongPress={() =>
                Alert.alert(
                  "Remove conversation?",
                  "It is hidden for you only and returns if they write again.",
                  [
                    { text: "Cancel", style: "cancel" },
                    { text: "Remove", style: "destructive", onPress: () => hide.mutate(item.id) }
                  ]
                )
              }
            >
              <Avatar url={item.user.photoUrl} name={item.user.firstName} />
              <View style={styles.flex}>
                <Text variant="label">{item.user.firstName}</Text>
                <Text
                  tone="muted"
                  numberOfLines={1}
                  style={item.unreadCount ? { fontWeight: "700", color: colors.text } : undefined}
                >
                  {item.lastMessage?.body ?? "You matched. Say hello!"}
                </Text>
              </View>
              <View style={styles.meta}>
                <Text variant="caption" tone="muted">
                  {formatTime(item.lastMessage?.createdAt ?? item.updatedAt)}
                </Text>
                {item.unreadCount > 0 ? (
                  <View style={[styles.badge, { backgroundColor: colors.accent }]}>
                    <Text variant="caption" tone="onPrimary">
                      {item.unreadCount > 9 ? "9+" : item.unreadCount}
                    </Text>
                  </View>
                ) : null}
              </View>
            </Pressable>
          )}
        />
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  row: {
    flexDirection: "row",
    alignItems: "center",
    paddingVertical: spacing.md,
    borderBottomWidth: StyleSheet.hairlineWidth
  },
  flex: { flex: 1, marginHorizontal: spacing.md },
  meta: { alignItems: "flex-end" },
  badge: {
    marginTop: spacing.xs,
    minWidth: 22,
    height: 22,
    borderRadius: radii.pill,
    alignItems: "center",
    justifyContent: "center",
    paddingHorizontal: 5
  }
});
