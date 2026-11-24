import React, { useMemo } from "react";
import { FlatList, Pressable, RefreshControl, ScrollView, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { Conversation } from "../../api/types";
import { useAuth } from "../../auth/AuthProvider";
import { AppText } from "../../components/AppText";
import { Avatar } from "../../components/Avatar";
import { Header } from "../../components/Header";
import { Screen } from "../../components/Screen";
import { EmptyState, ErrorState, LoadingState } from "../../components/States";
import { useConversations } from "../../hooks/queries";
import { colors, radii, spacing } from "../../theme/tokens";
import { errorMessage } from "../../utils/errors";
import { shortTimestamp } from "../../utils/format";

export default function MatchesScreen() {
  const navigation = useNavigation();
  const { account } = useAuth();
  const query = useConversations();
  const all = useMemo(() => query.data?.pages.flatMap((p) => p.conversations) ?? [], [query.data]);
  const fresh = all.filter((c) => !c.lastMessage);
  const threads = all.filter((c) => c.lastMessage);

  const open = (c: Conversation) =>
    navigation.navigate("Chat", {
      conversationId: c.id,
      matchId: c.matchId,
      userId: c.otherUser.userId,
      name: c.otherUser.firstName,
      photoUrl: c.otherUser.photo?.url ?? null
    });

  if (query.isPending) {
    return (
      <Screen>
        <Header title="Matchs" />
        <LoadingState />
      </Screen>
    );
  }
  if (query.isError) {
    return (
      <Screen>
        <Header title="Matchs" />
        <ErrorState message={errorMessage(query.error)} onRetry={() => void query.refetch()} />
      </Screen>
    );
  }

  const newMatches =
    fresh.length > 0 ? (
      <View style={styles.section}>
        <AppText variant="overline" color={colors.textSubtle} style={styles.sectionTitle}>
          Nouveaux matchs
        </AppText>
        <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerStyle={styles.carousel}>
          {fresh.map((c) => (
            <Pressable key={c.id} onPress={() => open(c)} style={styles.newMatch} accessibilityRole="button" accessibilityLabel={`Écrire à ${c.otherUser.firstName}`} testID={`new-match-${c.otherUser.userId}`}>
              <Avatar url={c.otherUser.photo?.url} name={c.otherUser.firstName} size={72} ring />
              <AppText variant="caption" numberOfLines={1}>
                {c.otherUser.firstName}
              </AppText>
            </Pressable>
          ))}
        </ScrollView>
      </View>
    ) : null;

  return (
    <Screen testID="matches-screen">
      <Header title="Matchs" />
      <FlatList
        data={threads}
        keyExtractor={(c) => c.id}
        ListHeaderComponent={
          <>
            {newMatches}
            {threads.length > 0 ? (
              <AppText variant="overline" color={colors.textSubtle} style={styles.sectionTitle}>
                Messages
              </AppText>
            ) : null}
          </>
        }
        ListEmptyComponent={
          fresh.length === 0 ? (
            <EmptyState icon="heart-half-outline" title="Pas encore de match" message="Quand un like est réciproque, la conversation apparaît ici." actionLabel="Découvrir des profils" onAction={() => navigation.navigate("Main", { screen: "Discover" })} />
          ) : (
            <AppText color={colors.textMuted} align="center" style={styles.hint}>
              Fais le premier pas : touche un match pour lui écrire.
            </AppText>
          )
        }
        renderItem={({ item }) => {
          const mine = item.lastMessage?.senderId === account?.id;
          const unread = item.unreadCount > 0;
          return (
            <Pressable onPress={() => open(item)} style={({ pressed }) => [styles.row, pressed && styles.pressed]} accessibilityRole="button" testID={`conversation-${item.otherUser.userId}`}>
              <Avatar url={item.otherUser.photo?.url} name={item.otherUser.firstName} size={56} />
              <View style={styles.rowText}>
                <View style={styles.rowTop}>
                  <AppText variant="bodyStrong" style={styles.flex} numberOfLines={1}>
                    {item.otherUser.firstName}
                  </AppText>
                  {item.lastMessage ? (
                    <AppText variant="caption" color={unread ? colors.primary : colors.textSubtle}>
                      {shortTimestamp(item.lastMessage.createdAt)}
                    </AppText>
                  ) : null}
                </View>
                <View style={styles.rowTop}>
                  <AppText numberOfLines={1} style={styles.flex} color={unread ? colors.text : colors.textMuted} variant={unread ? "bodyStrong" : "body"}>
                    {mine ? "Toi : " : ""}
                    {item.lastMessage?.body}
                  </AppText>
                  {unread ? (
                    <View style={styles.badge} accessibilityLabel={`${item.unreadCount} non lus`}>
                      <AppText variant="caption" color={colors.onPrimary}>
                        {item.unreadCount > 9 ? "9+" : item.unreadCount}
                      </AppText>
                    </View>
                  ) : null}
                </View>
              </View>
            </Pressable>
          );
        }}
        onEndReached={() => {
          if (query.hasNextPage && !query.isFetchingNextPage) void query.fetchNextPage();
        }}
        refreshControl={<RefreshControl refreshing={query.isRefetching} onRefresh={() => void query.refetch()} tintColor={colors.primary} />}
        contentContainerStyle={styles.list}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  list: { paddingBottom: spacing.xxl, flexGrow: 1 },
  section: { gap: spacing.sm, marginBottom: spacing.md },
  sectionTitle: { paddingHorizontal: spacing.lg, marginTop: spacing.sm, marginBottom: spacing.xs },
  carousel: { paddingHorizontal: spacing.lg, gap: spacing.md },
  newMatch: { alignItems: "center", gap: spacing.xs, width: 76 },
  row: { flexDirection: "row", alignItems: "center", gap: spacing.md, paddingHorizontal: spacing.lg, paddingVertical: spacing.sm },
  pressed: { backgroundColor: colors.surface },
  rowText: { flex: 1, gap: 2 },
  rowTop: { flexDirection: "row", alignItems: "center", gap: spacing.sm },
  flex: { flex: 1 },
  badge: { minWidth: 22, height: 22, borderRadius: radii.pill, backgroundColor: colors.primary, alignItems: "center", justifyContent: "center", paddingHorizontal: 6 },
  hint: { padding: spacing.lg }
});
