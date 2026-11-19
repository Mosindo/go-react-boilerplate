import React, { useMemo } from "react";
import { FlatList, Pressable, ScrollView, StyleSheet, View } from "react-native";
import { useInfiniteQuery } from "@tanstack/react-query";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { errorMessage } from "../../api/client";
import { chatApi } from "../../api/endpoints";
import { queryKeys } from "../../api/queryKeys";
import type { Conversation } from "../../api/types";
import { formatListDate } from "../../lib/format";
import type { AppStackParamList } from "../../navigation/types";
import { useTheme } from "../../theme/ThemeProvider";
import { radii, spacing } from "../../theme/tokens";
import { Avatar, EmptyState, ErrorState, Header, LoadingState, Screen, SectionTitle, Text } from "../../ui";

type Nav = NativeStackNavigationProp<AppStackParamList>;

export function useConversations() {
  return useInfiniteQuery({
    queryKey: queryKeys.conversations,
    queryFn: ({ pageParam }) => chatApi.list(pageParam),
    initialPageParam: null as string | null,
    getNextPageParam: (last) => last.nextCursor
  });
}

export function ConversationsScreen() {
  const navigation = useNavigation<Nav>();
  const { colors } = useTheme();
  const query = useConversations();
  const all = useMemo(() => query.data?.pages.flatMap((p) => p.conversations) ?? [], [query.data]);
  const fresh = all.filter((c) => !c.lastMessage);
  const threads = all.filter((c) => c.lastMessage);

  const open = (c: Conversation) => navigation.navigate("Chat", { conversationId: c.id, name: c.partner.firstName });

  if (query.isLoading) {
    return (
      <Screen>
        <Header title="Messages" large />
        <LoadingState />
      </Screen>
    );
  }
  if (query.error) {
    return (
      <Screen>
        <Header title="Messages" large />
        <ErrorState message={errorMessage(query.error)} onRetry={() => void query.refetch()} />
      </Screen>
    );
  }

  return (
    <Screen>
      <Header title="Messages" large />
      <FlatList
        data={threads}
        keyExtractor={(c) => c.id}
        refreshing={query.isRefetching}
        onRefresh={() => void query.refetch()}
        onEndReached={() => query.hasNextPage && !query.isFetchingNextPage && void query.fetchNextPage()}
        onEndReachedThreshold={0.4}
        contentContainerStyle={styles.list}
        ListHeaderComponent={
          fresh.length ? (
            <View>
              <SectionTitle>Nouveaux matchs</SectionTitle>
              <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerStyle={styles.freshRow}>
                {fresh.map((c) => (
                  <Pressable
                    key={c.id}
                    accessibilityRole="button"
                    accessibilityLabel={`Nouveau match avec ${c.partner.firstName}. Écrire un message`}
                    onPress={() => open(c)}
                    style={styles.fresh}
                    testID={`new-match-${c.partner.userId}`}
                  >
                    <Avatar path={c.partner.photoUrl} name={c.partner.firstName} size={68} ring />
                    <Text variant="caption" numberOfLines={1}>
                      {c.partner.firstName}
                    </Text>
                  </Pressable>
                ))}
              </ScrollView>
              {threads.length ? <SectionTitle>Conversations</SectionTitle> : null}
            </View>
          ) : null
        }
        ListEmptyComponent={
          fresh.length ? null : (
            <EmptyState
              icon="chatbubbles-outline"
              title="Pas encore de conversation"
              message="Quand un like est réciproque, vous pourrez discuter ici."
            />
          )
        }
        renderItem={({ item }) => {
          const last = item.lastMessage;
          const unread = item.unreadCount > 0;
          return (
            <Pressable
              accessibilityRole="button"
              accessibilityLabel={`Conversation avec ${item.partner.firstName}${unread ? `, ${item.unreadCount} non lus` : ""}`}
              onPress={() => open(item)}
              style={({ pressed }) => [styles.row, { backgroundColor: pressed ? colors.surfaceMuted : "transparent" }]}
              testID={`conversation-${item.partner.userId}`}
            >
              <Avatar path={item.partner.photoUrl} name={item.partner.firstName} size={56} />
              <View style={styles.rowBody}>
                <View style={styles.rowTop}>
                  <Text variant="label" style={styles.flex} numberOfLines={1}>
                    {item.partner.firstName}
                  </Text>
                  {last ? (
                    <Text variant="caption" tone={unread ? "primary" : "subtle"}>
                      {formatListDate(last.createdAt)}
                    </Text>
                  ) : null}
                </View>
                <View style={styles.rowTop}>
                  <Text tone={unread ? "default" : "muted"} weight={unread ? "600" : "400"} numberOfLines={1} style={styles.flex}>
                    {last ? `${last.isMine ? "Vous : " : ""}${last.body}` : ""}
                  </Text>
                  {unread ? (
                    <View style={[styles.badge, { backgroundColor: colors.primary }]}>
                      <Text variant="caption" tone="onPrimary">
                        {item.unreadCount > 99 ? "99+" : item.unreadCount}
                      </Text>
                    </View>
                  ) : null}
                </View>
              </View>
            </Pressable>
          );
        }}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  list: { flexGrow: 1, paddingBottom: spacing.xl },
  freshRow: { gap: spacing.md, paddingVertical: spacing.xs },
  fresh: { alignItems: "center", gap: 6, width: 76 },
  row: { flexDirection: "row", alignItems: "center", gap: spacing.sm, paddingVertical: spacing.sm, borderRadius: radii.md },
  rowBody: { flex: 1, gap: 2 },
  rowTop: { flexDirection: "row", alignItems: "center", gap: spacing.xs },
  flex: { flex: 1 },
  badge: { minWidth: 22, height: 22, borderRadius: 11, paddingHorizontal: 6, alignItems: "center", justifyContent: "center" }
});
