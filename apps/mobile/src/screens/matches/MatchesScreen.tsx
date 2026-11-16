import React, { useMemo } from "react";
import { FlatList, Pressable, RefreshControl, ScrollView, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import type { Match } from "../../api/types";
import { useMatches } from "../../hooks/queries";
import { useAuth } from "../../hooks/useAuth";
import type { AppStackParamList } from "../../navigation/types";
import { colors, spacing } from "../../theme/tokens";
import { Avatar, Badge, EmptyState, ErrorState, Header, LoadingState, Screen, Text } from "../../ui";
import { errorMessage } from "../../utils/errors";
import { relativeTime } from "../../utils/format";

type Nav = NativeStackNavigationProp<AppStackParamList>;

export default function MatchesScreen() {
  const navigation = useNavigation<Nav>();
  const { user } = useAuth();
  const query = useMatches();
  const all = useMemo(() => query.data?.pages.flatMap((p) => p.matches) ?? [], [query.data]);
  const fresh = all.filter((m) => !m.lastMessage);
  const conversations = all.filter((m) => m.lastMessage);
  const open = (m: Match) => navigation.navigate("Conversation", { matchId: m.id });

  if (query.isLoading) {
    return (
      <Screen>
        <Header title="Matchs" large />
        <LoadingState />
      </Screen>
    );
  }
  if (query.error && !all.length) {
    return (
      <Screen>
        <Header title="Matchs" large />
        <ErrorState message={errorMessage(query.error)} onRetry={() => query.refetch()} />
      </Screen>
    );
  }
  if (!all.length) {
    return (
      <Screen>
        <Header title="Matchs" large />
        <EmptyState
          icon="heart-outline"
          title="Pas encore de match"
          message="Quand l'intérêt est réciproque, la personne apparaît ici et vous pouvez discuter."
          actionLabel="Découvrir des profils"
          onAction={() => navigation.navigate("Tabs", { screen: "Discover" })}
          testID="matches-empty"
        />
      </Screen>
    );
  }

  return (
    <Screen padded={false} testID="matches-screen">
      <FlatList
        data={conversations}
        keyExtractor={(m) => m.id}
        contentContainerStyle={styles.list}
        refreshControl={<RefreshControl refreshing={query.isRefetching} onRefresh={() => query.refetch()} tintColor={colors.accent} />}
        onEndReached={() => query.hasNextPage && !query.isFetchingNextPage && query.fetchNextPage()}
        onEndReachedThreshold={0.4}
        ListHeaderComponent={
          <View style={styles.headerBlock}>
            <Header title="Matchs" large />
            {fresh.length ? (
              <View style={styles.freshBlock}>
                <Text variant="label" tone="muted">
                  NOUVEAUX MATCHS
                </Text>
                <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerStyle={styles.freshRow}>
                  {fresh.map((m) => (
                    <Pressable
                      key={m.id}
                      onPress={() => open(m)}
                      style={styles.fresh}
                      accessibilityRole="button"
                      accessibilityLabel={`Écrire à ${m.user.firstName}`}
                      testID={`new-match-${m.user.firstName}`}
                    >
                      <Avatar uri={m.user.photos[0]?.url} name={m.user.firstName} size={72} ring />
                      <Text variant="caption" numberOfLines={1}>
                        {m.user.firstName}
                      </Text>
                    </Pressable>
                  ))}
                </ScrollView>
              </View>
            ) : null}
            {conversations.length ? (
              <Text variant="label" tone="muted">
                MESSAGES
              </Text>
            ) : null}
          </View>
        }
        renderItem={({ item }) => {
          const mine = item.lastMessage?.senderId === user?.id;
          return (
            <Pressable
              onPress={() => open(item)}
              style={({ pressed }) => [styles.row, pressed && styles.pressed]}
              accessibilityRole="button"
              accessibilityLabel={`Conversation avec ${item.user.firstName}`}
              testID={`conversation-${item.user.firstName}`}
            >
              <Avatar uri={item.user.photos[0]?.url} name={item.user.firstName} size={56} />
              <View style={styles.rowText}>
                <View style={styles.rowTop}>
                  <Text variant="bodyStrong" numberOfLines={1} style={styles.flex}>
                    {item.user.firstName}
                  </Text>
                  {item.lastMessage ? (
                    <Text variant="caption" tone="subtle">
                      {relativeTime(item.lastMessage.createdAt)}
                    </Text>
                  ) : null}
                </View>
                <View style={styles.rowTop}>
                  <Text variant="caption" tone={item.unreadCount ? "default" : "muted"} numberOfLines={1} style={styles.flex}>
                    {mine ? "Toi : " : ""}
                    {item.lastMessage?.body}
                  </Text>
                  <Badge count={item.unreadCount} />
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
  list: { paddingHorizontal: spacing.xl, paddingTop: spacing.lg, paddingBottom: 120, maxWidth: 600, width: "100%", alignSelf: "center" },
  headerBlock: { gap: spacing.md, marginBottom: spacing.sm },
  freshBlock: { gap: spacing.sm, marginBottom: spacing.md },
  freshRow: { gap: spacing.lg, paddingVertical: spacing.xs },
  fresh: { alignItems: "center", gap: spacing.xs, width: 80 },
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    paddingVertical: spacing.md,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: colors.border
  },
  pressed: { opacity: 0.7 },
  rowText: { flex: 1, gap: 4 },
  rowTop: { flexDirection: "row", alignItems: "center", gap: spacing.sm },
  flex: { flex: 1 }
});
