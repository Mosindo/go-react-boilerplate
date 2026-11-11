import React, { useMemo } from "react";
import { FlatList, Pressable, RefreshControl, ScrollView, StyleSheet, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { errorMessage } from "../../api/client";
import type { Conversation } from "../../api/types";
import { useAuth } from "../../hooks/useAuth";
import { useConversations } from "../../hooks/queries";
import { useRealtime } from "../../hooks/useRealtime";
import type { AppStackParamList } from "../../navigation/types";
import { useTheme } from "../../theme/ThemeProvider";
import { Avatar, EmptyState, ErrorState, LoadingState, Text } from "../../ui";
import { formatRelative } from "../../utils/format";

export function ConversationsScreen() {
  const { colors, spacing } = useTheme();
  const { user } = useAuth();
  const navigation = useNavigation<NativeStackNavigationProp<AppStackParamList>>();
  const { connected } = useRealtime();
  const query = useConversations(!connected);

  const all = useMemo(() => query.data?.pages.flatMap((p) => p.conversations) ?? [], [query.data]);
  const fresh = all.filter((c) => !c.lastMessage);
  const threads = all.filter((c) => c.lastMessage);
  const open = (c: Conversation) => navigation.navigate("Chat", { matchId: c.matchId, user: c.user });

  if (query.isLoading) {
    return <LoadingState />;
  }
  if (query.isError) {
    return <ErrorState message={errorMessage(query.error)} onRetry={() => void query.refetch()} />;
  }

  return (
    <SafeAreaView edges={["top"]} style={[styles.flex, { backgroundColor: colors.background }]} testID="matches-screen">
      <FlatList
        data={threads}
        keyExtractor={(c) => c.matchId}
        refreshControl={
          <RefreshControl
            refreshing={query.isRefetching}
            onRefresh={() => void query.refetch()}
            tintColor={colors.primary}
          />
        }
        onEndReached={() => query.hasNextPage && !query.isFetchingNextPage && void query.fetchNextPage()}
        onEndReachedThreshold={0.4}
        contentContainerStyle={[styles.list, { padding: spacing.lg }]}
        ListHeaderComponent={
          <View style={{ gap: spacing.md, marginBottom: spacing.md }}>
            <Text variant="title">Matchs</Text>
            {fresh.length > 0 ? (
              <View style={{ gap: 10 }}>
                <Text variant="overline" tone="subtle">
                  Nouveaux matchs
                </Text>
                <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerStyle={{ gap: 14 }}>
                  {fresh.map((c) => (
                    <Pressable
                      key={c.matchId}
                      testID={`new-match-${c.user.userId}`}
                      accessibilityRole="button"
                      accessibilityLabel={`Écrire à ${c.user.firstName}`}
                      onPress={() => open(c)}
                      style={styles.newMatch}
                    >
                      <View style={[styles.ring, { borderColor: colors.primary }]}>
                        <Avatar url={c.user.photos[0]?.url} size={68} />
                      </View>
                      <Text variant="caption" numberOfLines={1}>
                        {c.user.firstName}
                      </Text>
                    </Pressable>
                  ))}
                </ScrollView>
              </View>
            ) : null}
            {threads.length > 0 ? (
              <Text variant="overline" tone="subtle">
                Messages
              </Text>
            ) : null}
          </View>
        }
        ListEmptyComponent={
          fresh.length === 0 ? (
            <EmptyState
              icon="heart-outline"
              title="Pas encore de match"
              message="Quand l'intérêt est réciproque, la conversation apparaîtra ici."
            />
          ) : (
            <Text tone="muted" align="center" style={{ marginTop: 24 }}>
              Brise la glace avec un de tes nouveaux matchs !
            </Text>
          )
        }
        renderItem={({ item }) => {
          const last = item.lastMessage;
          const mine = last?.senderId === user?.id;
          const unread = item.unreadCount > 0;
          return (
            <Pressable
              testID={`conversation-${item.user.userId}`}
              accessibilityRole="button"
              accessibilityLabel={`Conversation avec ${item.user.firstName}${unread ? `, ${item.unreadCount} non lus` : ""}`}
              onPress={() => open(item)}
              style={({ pressed }) => [
                styles.row,
                { borderBottomColor: colors.border },
                pressed ? { opacity: 0.7 } : null
              ]}
            >
              <Avatar url={item.user.photos[0]?.url} size={56} />
              <View style={styles.flex}>
                <View style={styles.rowTop}>
                  <Text variant="label" style={styles.flex} numberOfLines={1}>
                    {item.user.firstName}
                  </Text>
                  {last ? (
                    <Text variant="caption" tone={unread ? "primary" : "subtle"}>
                      {formatRelative(last.createdAt)}
                    </Text>
                  ) : null}
                </View>
                <View style={styles.rowTop}>
                  <Text
                    tone={unread ? "default" : "muted"}
                    weight={unread ? "600" : undefined}
                    numberOfLines={1}
                    style={styles.flex}
                  >
                    {mine ? "Toi : " : ""}
                    {last?.body}
                  </Text>
                  {unread ? (
                    <View style={[styles.badge, { backgroundColor: colors.primary }]}>
                      <Text variant="caption" weight="700" style={{ color: colors.onPrimary }}>
                        {item.unreadCount > 9 ? "9+" : item.unreadCount}
                      </Text>
                    </View>
                  ) : null}
                </View>
              </View>
            </Pressable>
          );
        }}
      />
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  list: { flexGrow: 1, width: "100%", maxWidth: 640, alignSelf: "center", paddingBottom: 40 },
  newMatch: { alignItems: "center", gap: 6, width: 76 },
  ring: { borderWidth: 2, borderRadius: 40, padding: 2 },
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: 14,
    paddingVertical: 12,
    borderBottomWidth: StyleSheet.hairlineWidth
  },
  rowTop: { flexDirection: "row", alignItems: "center", gap: 8 },
  badge: {
    minWidth: 22,
    height: 22,
    borderRadius: 11,
    alignItems: "center",
    justifyContent: "center",
    paddingHorizontal: 6
  }
});
