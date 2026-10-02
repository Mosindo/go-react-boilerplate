import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useInfiniteQuery } from "@tanstack/react-query";
import React from "react";
import { FlatList, Pressable, RefreshControl, ScrollView, StyleSheet, View } from "react-native";

import { matchesApi } from "../../api/endpoints";
import type { MatchItem } from "../../api/types";
import { formatShortTime } from "../../lib/format";
import type { MainStackParamList } from "../../navigation/types";
import { keys } from "../../realtime/RealtimeProvider";
import { radii, spacing, useTheme } from "../../theme";
import { PhotoImage } from "../../ui/PhotoImage";
import { Screen } from "../../ui/Screen";
import { EmptyState, ErrorState, Loading } from "../../ui/States";
import { Text } from "../../ui/Text";
import { useAuth } from "../../auth/AuthContext";

type Nav = NativeStackNavigationProp<MainStackParamList>;

export function useMatchesQuery() {
  return useInfiniteQuery({
    queryKey: keys.matches,
    queryFn: ({ pageParam }) => matchesApi.list(pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => last.nextCursor,
  });
}

export function MessagesScreen() {
  const { colors } = useTheme();
  const { user } = useAuth();
  const navigation = useNavigation<Nav>();
  const query = useMatchesQuery();

  const all = query.data?.pages.flatMap((p) => p.matches) ?? [];
  const fresh = all.filter((m) => !m.lastMessage);
  const conversations = all.filter((m) => m.lastMessage);

  const open = (m: MatchItem) =>
    navigation.navigate("Chat", { matchId: m.id, name: m.user.firstName, userId: m.user.id });

  if (query.isLoading) return <Loading />;
  if (query.isError) return <ErrorState error={query.error} onRetry={() => void query.refetch()} />;

  return (
    <Screen edges={["top"]} padded={false}>
      <View style={styles.title}>
        <Text variant="title">Messages</Text>
      </View>
      {all.length === 0 ? (
        <EmptyState
          icon="chatbubbles-outline"
          title="Pas encore de match"
          message="Quand vous et une autre personne vous plairez mutuellement, la conversation apparaîtra ici."
        />
      ) : (
        <FlatList
          data={conversations}
          keyExtractor={(m) => m.id}
          refreshControl={
            <RefreshControl
              refreshing={query.isRefetching && !query.isFetchingNextPage}
              onRefresh={() => void query.refetch()}
              tintColor={colors.primary}
            />
          }
          onEndReached={() => query.hasNextPage && void query.fetchNextPage()}
          onEndReachedThreshold={0.4}
          ListHeaderComponent={
            fresh.length > 0 ? (
              <View style={styles.fresh}>
                <Text variant="label" tone="muted" style={styles.pad}>
                  Nouveaux matchs
                </Text>
                <ScrollView
                  horizontal
                  showsHorizontalScrollIndicator={false}
                  contentContainerStyle={styles.freshRow}
                >
                  {fresh.map((m) => (
                    <Pressable
                      key={m.id}
                      onPress={() => open(m)}
                      accessibilityRole="button"
                      accessibilityLabel={`Nouveau match avec ${m.user.firstName}`}
                      style={styles.freshItem}
                    >
                      <PhotoImage
                        photo={m.user.photos[0]}
                        name={m.user.firstName}
                        round
                        size={68}
                        style={{ borderWidth: 2, borderColor: colors.primary }}
                      />
                      <Text variant="caption" numberOfLines={1}>
                        {m.user.firstName}
                      </Text>
                    </Pressable>
                  ))}
                </ScrollView>
              </View>
            ) : null
          }
          ListEmptyComponent={
            <Text tone="muted" center style={styles.pad}>
              Dites bonjour à l&apos;un de vos nouveaux matchs 👋
            </Text>
          }
          renderItem={({ item }) => {
            const last = item.lastMessage;
            const mine = last?.senderId === user?.id;
            return (
              <Pressable
                testID={`conversation-${item.user.firstName}`}
                onPress={() => open(item)}
                accessibilityRole="button"
                accessibilityLabel={`Conversation avec ${item.user.firstName}${item.unreadCount ? `, ${item.unreadCount} non lus` : ""}`}
                style={({ pressed }) => [
                  styles.row,
                  pressed && { backgroundColor: colors.surfaceAlt },
                ]}
              >
                <PhotoImage
                  photo={item.user.photos[0]}
                  name={item.user.firstName}
                  round
                  size={56}
                />
                <View style={styles.rowText}>
                  <View style={styles.rowTop}>
                    <Text variant="heading" numberOfLines={1} style={styles.flex}>
                      {item.user.firstName}
                    </Text>
                    {last ? (
                      <Text variant="caption" tone="muted">
                        {formatShortTime(last.createdAt)}
                      </Text>
                    ) : null}
                  </View>
                  <View style={styles.rowTop}>
                    <Text
                      tone={item.unreadCount ? "default" : "muted"}
                      numberOfLines={1}
                      style={[styles.flex, item.unreadCount ? { fontWeight: "700" } : null]}
                    >
                      {mine ? "Vous : " : ""}
                      {last?.body}
                    </Text>
                    {item.unreadCount > 0 ? (
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
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  title: { paddingHorizontal: spacing.lg, paddingVertical: spacing.sm },
  pad: { paddingHorizontal: spacing.lg, paddingVertical: spacing.md },
  fresh: { paddingBottom: spacing.sm },
  freshRow: { paddingHorizontal: spacing.lg, gap: spacing.lg },
  freshItem: { alignItems: "center", gap: spacing.xs, width: 72 },
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md,
  },
  rowText: { flex: 1, gap: 2 },
  rowTop: { flexDirection: "row", alignItems: "center", gap: spacing.sm },
  flex: { flex: 1 },
  badge: {
    minWidth: 22,
    height: 22,
    borderRadius: radii.pill,
    alignItems: "center",
    justifyContent: "center",
    paddingHorizontal: 6,
  },
});
