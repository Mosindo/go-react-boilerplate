import React, { useMemo } from "react";
import { FlatList, Pressable, RefreshControl, ScrollView, StyleSheet, View } from "react-native";
import { errorMessage } from "../../api/client";
import type { MatchSummary } from "../../api/types";
import { useMatches } from "../../hooks/queries";
import { useMyId } from "../../hooks/useAuth";
import { useRealtimeConnected } from "../../hooks/useRealtime";
import { useTheme } from "../../theme/ThemeProvider";
import { spacing } from "../../theme/palette";
import { Avatar, Badge, EmptyState, ErrorState, Header, Loading, Screen, Text } from "../../ui";
import { formatRelativeShort } from "../../utils/format";
import type { TabScreenProps } from "../../navigation/types";

export function MatchesScreen({ navigation }: TabScreenProps<"Matches">) {
  const { colors } = useTheme();
  const myId = useMyId();
  const connected = useRealtimeConnected();
  const query = useMatches(!connected);
  const all = useMemo(() => query.data?.pages.flatMap((p) => p.matches) ?? [], [query.data]);
  const fresh = all.filter((m) => !m.lastMessage);
  const conversations = all.filter((m) => m.lastMessage);
  const open = (m: MatchSummary) => navigation.navigate("Chat", { matchId: m.id });

  if (query.isLoading) {
    return (
      <Screen>
        <Header title="Matchs" large />
        <Loading />
      </Screen>
    );
  }
  if (query.error && all.length === 0) {
    return (
      <Screen>
        <Header title="Matchs" large />
        <ErrorState message={errorMessage(query.error)} onRetry={() => void query.refetch()} />
      </Screen>
    );
  }

  const header = (
    <View style={styles.headerBlock}>
      {fresh.length ? (
        <View style={styles.section}>
          <Text variant="overline" tone="subtle">
            NOUVEAUX MATCHS
          </Text>
          <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerStyle={styles.freshRow}>
            {fresh.map((m) => (
              <Pressable key={m.id} onPress={() => open(m)} style={styles.fresh} accessibilityRole="button" accessibilityLabel={`Écrire à ${m.user.firstName}`} testID={`fresh-${m.user.firstName}`}>
                <Avatar url={m.user.photos[0]?.url} name={m.user.firstName} size={72} ring />
                <Text variant="label" numberOfLines={1}>
                  {m.user.firstName}
                </Text>
              </Pressable>
            ))}
          </ScrollView>
        </View>
      ) : null}
      {conversations.length ? (
        <Text variant="overline" tone="subtle" style={styles.convTitle}>
          MESSAGES
        </Text>
      ) : null}
    </View>
  );

  return (
    <Screen padded={false} testID="matches-screen">
      <View style={styles.pad}>
        <Header title="Matchs" large />
      </View>
      {all.length === 0 ? (
        <EmptyState
          icon="heart-half-outline"
          title="Pas encore de match"
          message="Quand l'intérêt est réciproque, la personne apparaît ici et vous pouvez discuter."
          actionLabel="Découvrir des profils"
          onAction={() => navigation.navigate("Discover")}
        />
      ) : (
        <FlatList
          data={conversations}
          keyExtractor={(m) => m.id}
          ListHeaderComponent={header}
          contentContainerStyle={styles.list}
          refreshControl={<RefreshControl refreshing={query.isRefetching} onRefresh={() => void query.refetch()} tintColor={colors.primary} />}
          onEndReachedThreshold={0.4}
          onEndReached={() => {
            if (query.hasNextPage && !query.isFetchingNextPage) void query.fetchNextPage();
          }}
          ListEmptyComponent={
            <Text tone="muted" align="center" style={styles.hint}>
              Fais le premier pas : un message sincère vaut mieux qu’un simple « salut ».
            </Text>
          }
          renderItem={({ item }) => {
            const last = item.lastMessage!;
            const unread = item.unreadCount > 0;
            return (
              <Pressable
                onPress={() => open(item)}
                accessibilityRole="button"
                accessibilityLabel={`Conversation avec ${item.user.firstName}${unread ? `, ${item.unreadCount} non lus` : ""}`}
                style={({ pressed }) => [styles.row, { backgroundColor: pressed ? colors.surfaceMuted : "transparent" }]}
                testID={`conversation-${item.user.firstName}`}
              >
                <Avatar url={item.user.photos[0]?.url} name={item.user.firstName} size={56} />
                <View style={styles.flex}>
                  <View style={styles.rowTop}>
                    <Text variant="bodyStrong" numberOfLines={1} style={styles.flex}>
                      {item.user.firstName}
                    </Text>
                    <Text variant="caption" tone={unread ? "primary" : "subtle"}>
                      {formatRelativeShort(last.createdAt)}
                    </Text>
                  </View>
                  <View style={styles.rowTop}>
                    <Text variant="caption" tone={unread ? "default" : "muted"} numberOfLines={1} style={[styles.flex, unread ? styles.bold : null]}>
                      {last.senderUserId === myId ? "Toi : " : ""}
                      {last.content}
                    </Text>
                    <Badge count={item.unreadCount} />
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
  pad: { paddingHorizontal: spacing.lg },
  flex: { flex: 1 },
  list: { paddingBottom: spacing.xxl, maxWidth: 720, width: "100%", alignSelf: "center" },
  headerBlock: { gap: spacing.md, paddingTop: spacing.xs },
  section: { gap: spacing.sm, paddingHorizontal: spacing.lg },
  freshRow: { gap: spacing.md, paddingRight: spacing.lg },
  fresh: { alignItems: "center", gap: 6, width: 76 },
  convTitle: { paddingHorizontal: spacing.lg },
  row: { flexDirection: "row", alignItems: "center", gap: spacing.sm, paddingHorizontal: spacing.lg, paddingVertical: spacing.sm },
  rowTop: { flexDirection: "row", alignItems: "center", gap: spacing.xs },
  bold: { fontWeight: "600" },
  hint: { padding: spacing.xl }
});
