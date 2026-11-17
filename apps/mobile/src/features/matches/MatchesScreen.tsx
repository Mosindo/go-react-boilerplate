import React, { useMemo } from "react";
import { FlatList, Pressable, RefreshControl, ScrollView, StyleSheet, View } from "react-native";
import { useInfiniteQuery } from "@tanstack/react-query";
import { chatApi } from "../../api/endpoints";
import { queryKeys } from "../../api/queryKeys";
import type { Conversation } from "../../api/types";
import { useCurrentUser } from "../../auth/AuthProvider";
import type { TabScreenProps } from "../../navigation/types";
import { colors, radii, spacing } from "../../theme";
import { Avatar, EmptyState, ErrorState, LoadingState, Screen, Text } from "../../ui";
import { shortTime } from "../../utils/format";

export function MatchesScreen({ navigation }: TabScreenProps<"Matches">) {
  const me = useCurrentUser();
  const query = useInfiniteQuery({
    queryKey: queryKeys.conversations,
    queryFn: ({ pageParam }) => chatApi.conversations(pageParam),
    initialPageParam: null as string | null,
    getNextPageParam: (last) => last.nextCursor
  });

  const all = useMemo(() => query.data?.pages.flatMap((p) => p.conversations) ?? [], [query.data]);
  const fresh = all.filter((c) => !c.lastMessage);
  const threads = all.filter((c) => c.lastMessage);
  const open = (c: Conversation) => navigation.navigate("Conversation", { conversationId: c.id });

  if (query.isLoading)
    return (
      <Screen>
        <LoadingState />
      </Screen>
    );
  if (query.isError)
    return (
      <Screen>
        <ErrorState onRetry={() => void query.refetch()} />
      </Screen>
    );

  return (
    <Screen testID="matches-screen">
      <Text variant="title" style={styles.title}>
        Matchs
      </Text>
      <FlatList
        data={threads}
        keyExtractor={(c) => c.id}
        refreshControl={<RefreshControl refreshing={query.isRefetching} onRefresh={() => void query.refetch()} tintColor={colors.accent} />}
        onEndReached={() => query.hasNextPage && !query.isFetchingNextPage && void query.fetchNextPage()}
        onEndReachedThreshold={0.4}
        contentContainerStyle={styles.list}
        ListHeaderComponent={
          fresh.length > 0 ? (
            <View style={styles.freshBlock}>
              <Text variant="overline" tone="subtle">
                Nouveaux matchs
              </Text>
              <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerStyle={styles.fresh}>
                {fresh.map((c) => (
                  <Pressable
                    key={c.id}
                    testID={`new-match-${c.profile.userId}`}
                    accessibilityRole="button"
                    accessibilityLabel={`Écrire à ${c.profile.firstName}`}
                    onPress={() => open(c)}
                    style={styles.freshItem}
                  >
                    <Avatar url={c.profile.photos[0]?.url} name={c.profile.firstName} size={72} ring />
                    <Text variant="label" numberOfLines={1}>
                      {c.profile.firstName}
                    </Text>
                  </Pressable>
                ))}
              </ScrollView>
              {threads.length > 0 ? (
                <Text variant="overline" tone="subtle" style={styles.sectionLabel}>
                  Messages
                </Text>
              ) : null}
            </View>
          ) : threads.length > 0 ? (
            <Text variant="overline" tone="subtle" style={styles.sectionLabel}>
              Messages
            </Text>
          ) : null
        }
        ListEmptyComponent={
          fresh.length === 0 ? (
            <EmptyState
              icon="heart-outline"
              title="Pas encore de match"
              message="Quand une personne que tu aimes t'aime aussi, elle apparaîtra ici."
              actionLabel="Découvrir des profils"
              onAction={() => navigation.navigate("Discover")}
            />
          ) : (
            <Text tone="subtle" center style={styles.hint}>
              Lance la conversation avec tes nouveaux matchs ✨
            </Text>
          )
        }
        renderItem={({ item }) => {
          const unread = item.unreadCount > 0;
          const mine = item.lastMessage?.senderId === me.id;
          return (
            <Pressable
              testID={`conversation-${item.profile.userId}`}
              accessibilityRole="button"
              accessibilityLabel={`Conversation avec ${item.profile.firstName}${unread ? `, ${item.unreadCount} non lus` : ""}`}
              onPress={() => open(item)}
              style={({ pressed }) => [styles.row, pressed ? styles.pressed : null]}
            >
              <Avatar url={item.profile.photos[0]?.url} name={item.profile.firstName} size={56} />
              <View style={styles.rowText}>
                <View style={styles.rowTop}>
                  <Text variant="bodyStrong" numberOfLines={1} style={styles.flex}>
                    {item.profile.firstName}
                  </Text>
                  {item.lastMessage ? (
                    <Text variant="caption" tone={unread ? "accent" : "subtle"}>
                      {shortTime(item.lastMessage.createdAt)}
                    </Text>
                  ) : null}
                </View>
                <View style={styles.rowTop}>
                  <Text tone={unread ? "default" : "muted"} variant={unread ? "bodyStrong" : "body"} numberOfLines={1} style={styles.flex}>
                    {mine ? "Toi : " : ""}
                    {item.lastMessage?.content}
                  </Text>
                  {unread ? (
                    <View style={styles.badge}>
                      <Text variant="caption" tone="inverse">
                        {item.unreadCount}
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
  title: {
    marginBottom: spacing.md
  },
  list: {
    flexGrow: 1,
    paddingBottom: spacing.xl
  },
  freshBlock: {
    gap: spacing.sm
  },
  fresh: {
    gap: spacing.lg,
    paddingVertical: spacing.sm
  },
  freshItem: {
    alignItems: "center",
    gap: spacing.xs,
    width: 80
  },
  sectionLabel: {
    marginTop: spacing.md,
    marginBottom: spacing.xs
  },
  hint: {
    marginTop: spacing.xl
  },
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    paddingVertical: spacing.md,
    borderRadius: radii.md
  },
  pressed: {
    backgroundColor: colors.surface
  },
  rowText: {
    flex: 1,
    gap: 2
  },
  rowTop: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.sm
  },
  flex: {
    flex: 1
  },
  badge: {
    minWidth: 22,
    height: 22,
    borderRadius: 11,
    paddingHorizontal: 6,
    backgroundColor: colors.accent,
    alignItems: "center",
    justifyContent: "center"
  }
});
