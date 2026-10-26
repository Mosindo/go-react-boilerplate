import React from "react";
import { FlatList, Pressable, RefreshControl, ScrollView, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import type { Conversation } from "../api";
import { Avatar } from "../components/Photo";
import { Screen } from "../components/Screen";
import { EmptyState, ErrorState, LoadingState } from "../components/States";
import { Text } from "../components/Text";
import { useConversations } from "../hooks/useQueries";
import { timeLabel } from "../lib/format";
import type { RootStackParamList } from "../navigation/types";
import { radius, spacing, useTheme } from "../theme";

export function MessagesScreen() {
  const theme = useTheme();
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>();
  const query = useConversations();
  const conversations = query.data?.pages.flatMap((p) => p.conversations) ?? [];
  const fresh = conversations.filter((c) => !c.lastMessage);
  const talking = conversations.filter((c) => c.lastMessage);

  const open = (c: Conversation) => navigation.navigate("Chat", { matchId: c.matchId, profile: c.profile });

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
      <Text variant="title">Messages</Text>
      {conversations.length === 0 ? (
        <EmptyState
          emoji="💬"
          title="Pas encore de match"
          message="Quand vous et une autre personne vous plaisez mutuellement, la conversation s'ouvre ici."
        />
      ) : (
        <FlatList
          data={talking}
          keyExtractor={(c) => c.matchId}
          refreshControl={
            <RefreshControl
              refreshing={query.isRefetching && !query.isFetchingNextPage}
              onRefresh={() => void query.refetch()}
            />
          }
          onEndReached={() => query.hasNextPage && !query.isFetchingNextPage && void query.fetchNextPage()}
          onEndReachedThreshold={0.4}
          ListHeaderComponent={
            fresh.length > 0 ? (
              <View style={styles.rail}>
                <Text variant="label" muted>
                  NOUVEAUX MATCHS
                </Text>
                <ScrollView
                  horizontal
                  showsHorizontalScrollIndicator={false}
                  contentContainerStyle={styles.railRow}
                >
                  {fresh.map((c) => (
                    <Pressable
                      key={c.matchId}
                      onPress={() => open(c)}
                      accessibilityRole="button"
                      accessibilityLabel={`Nouveau match : ${c.profile.firstName}`}
                      style={styles.railItem}
                      testID={`new-match-${c.profile.firstName}`}
                    >
                      <View style={[styles.ring, { borderColor: theme.primary }]}>
                        <Avatar url={c.profile.photos[0]?.url} name={c.profile.firstName} size={64} />
                      </View>
                      <Text variant="caption">{c.profile.firstName}</Text>
                    </Pressable>
                  ))}
                </ScrollView>
              </View>
            ) : null
          }
          ListFooterComponent={talking.length > 0 ? <View style={{ height: spacing.xl }} /> : null}
          renderItem={({ item }) => (
            <Pressable
              onPress={() => open(item)}
              accessibilityRole="button"
              accessibilityLabel={`Conversation avec ${item.profile.firstName}${item.unreadCount ? `, ${item.unreadCount} non lu` : ""}`}
              style={({ pressed }) => [
                styles.row,
                { backgroundColor: pressed ? theme.surfaceAlt : "transparent" }
              ]}
              testID={`conversation-${item.profile.firstName}`}
            >
              <Avatar url={item.profile.photos[0]?.url} name={item.profile.firstName} size={56} />
              <View style={styles.rowText}>
                <View style={styles.rowTop}>
                  <Text variant="bodyStrong">{item.profile.firstName}</Text>
                  <Text variant="caption" muted>
                    {item.lastMessage ? timeLabel(item.lastMessage.createdAt) : ""}
                  </Text>
                </View>
                <Text
                  numberOfLines={1}
                  muted={item.unreadCount === 0}
                  variant={item.unreadCount ? "bodyStrong" : "body"}
                >
                  {item.lastMessage?.body ?? ""}
                </Text>
              </View>
              {item.unreadCount > 0 ? (
                <View style={[styles.badge, { backgroundColor: theme.primary }]}>
                  <Text variant="caption" color={theme.onPrimary}>
                    {item.unreadCount > 9 ? "9+" : item.unreadCount}
                  </Text>
                </View>
              ) : null}
            </Pressable>
          )}
        />
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  rail: { gap: spacing.sm, paddingVertical: spacing.md },
  railRow: { gap: spacing.lg },
  railItem: { alignItems: "center", gap: spacing.xs },
  ring: { borderWidth: 2.5, borderRadius: 999, padding: 2 },
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    paddingVertical: spacing.md,
    borderRadius: radius.md
  },
  rowText: { flex: 1, gap: 2 },
  rowTop: { flexDirection: "row", justifyContent: "space-between" },
  badge: {
    minWidth: 24,
    height: 24,
    borderRadius: 12,
    alignItems: "center",
    justifyContent: "center",
    paddingHorizontal: 6
  }
});
