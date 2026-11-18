import React, { useMemo } from "react";
import { FlatList, Pressable, RefreshControl, ScrollView, StyleSheet, View } from "react-native";
import type { ChatSummary } from "../../api/types";
import { useChats, useMatches } from "../../hooks/queries";
import { useAuth } from "../../hooks/useAuth";
import { useRealtime } from "../../hooks/useRealtime";
import type { TabScreenProps } from "../../navigation/types";
import { layout, spacing, useTheme } from "../../theme";
import { Avatar, EmptyState, ErrorState, Header, LoadingState, Screen, Text } from "../../ui";
import { formatShortDate } from "../../utils/dates";

export default function MatchesScreen({ navigation }: TabScreenProps<"Matches">) {
  const { colors } = useTheme();
  const { user } = useAuth();
  const { connected } = useRealtime();
  // Poll only while the realtime socket is down.
  const polling = connected ? false : 15_000;
  const matches = useMatches(polling);
  const chats = useChats(polling);

  const newMatches = useMemo(() => (matches.data ?? []).filter((m) => !m.hasMessages), [matches.data]);
  const openChat = (u: { userId: string; firstName: string; photoUrl: string | null }) =>
    navigation.navigate("Chat", { userId: u.userId, firstName: u.firstName, photoUrl: u.photoUrl });

  if (matches.isLoading || chats.isLoading) {
    return (
      <Screen>
        <Header title="Matchs" />
        <LoadingState />
      </Screen>
    );
  }
  if (matches.isError || chats.isError) {
    return (
      <Screen>
        <Header title="Matchs" />
        <ErrorState
          message="Impossible de charger vos conversations."
          onRetry={() => void Promise.all([matches.refetch(), chats.refetch()])}
        />
      </Screen>
    );
  }

  const conversations = chats.data ?? [];
  const nothing = newMatches.length === 0 && conversations.length === 0;

  const renderChat = ({ item }: { item: ChatSummary }) => {
    const mine = item.lastMessage.senderUserId === user?.id;
    const unread = item.unreadCount > 0;
    return (
      <Pressable
        accessibilityLabel={`Conversation avec ${item.user.firstName}${unread ? `, ${item.unreadCount} non lus` : ""}`}
        accessibilityRole="button"
        onPress={() => openChat(item.user)}
        style={({ pressed }) => [styles.row, pressed ? { backgroundColor: colors.surfaceAlt } : null]}
        testID={`chat-row-${item.user.userId}`}
      >
        <Avatar name={item.user.firstName} size={56} uri={item.user.photoUrl} />
        <View style={styles.rowText}>
          <View style={styles.rowTop}>
            <Text variant="bodyStrong">{item.user.firstName}</Text>
            <Text color={unread ? "accent" : "textSubtle"} variant="caption">
              {formatShortDate(item.lastMessage.createdAt)}
            </Text>
          </View>
          <View style={styles.rowTop}>
            <Text color={unread ? "text" : "textMuted"} numberOfLines={1} style={[styles.preview, unread ? styles.bold : null]}>
              {mine ? "Vous : " : ""}
              {item.lastMessage.content}
            </Text>
            {unread ? (
              <View style={[styles.badge, { backgroundColor: colors.accent }]}>
                <Text color="textOnAccent" variant="caption">
                  {item.unreadCount > 9 ? "9+" : item.unreadCount}
                </Text>
              </View>
            ) : null}
          </View>
        </View>
      </Pressable>
    );
  };

  return (
    <Screen testID="matches-screen">
      <Header title="Matchs" subtitle={connected ? undefined : "Reconnexion au temps réel…"} />
      {nothing ? (
        <EmptyState
          actionLabel="Découvrir des profils"
          icon="heart-outline"
          message="Quand un like est réciproque, la personne apparaît ici et vous pouvez discuter."
          onAction={() => navigation.navigate("Discover")}
          title="Pas encore de match"
        />
      ) : (
        <FlatList
          ListHeaderComponent={
            newMatches.length > 0 ? (
              <View style={styles.section}>
                <Text color="textSubtle" variant="overline">
                  Nouveaux matchs
                </Text>
                <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerStyle={styles.carousel}>
                  {newMatches.map((m) => (
                    <Pressable
                      key={m.id}
                      accessibilityLabel={`Écrire à ${m.user.firstName}`}
                      accessibilityRole="button"
                      onPress={() => openChat(m.user)}
                      style={styles.matchItem}
                      testID={`new-match-${m.user.userId}`}
                    >
                      <Avatar name={m.user.firstName} ring size={72} uri={m.user.photoUrl} />
                      <Text numberOfLines={1} variant="label">
                        {m.user.firstName}
                      </Text>
                    </Pressable>
                  ))}
                </ScrollView>
                {conversations.length > 0 ? (
                  <Text color="textSubtle" variant="overline">
                    Messages
                  </Text>
                ) : null}
              </View>
            ) : (
              <Text color="textSubtle" style={styles.sectionLabel} variant="overline">
                Messages
              </Text>
            )
          }
          contentContainerStyle={styles.list}
          data={conversations}
          keyExtractor={(c) => c.user.userId}
          refreshControl={
            <RefreshControl
              onRefresh={() => void Promise.all([matches.refetch(), chats.refetch()])}
              refreshing={matches.isRefetching || chats.isRefetching}
              tintColor={colors.accent}
            />
          }
          renderItem={renderChat}
        />
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  list: { paddingBottom: 110, width: "100%", maxWidth: layout.maxContentWidth, alignSelf: "center" },
  section: { gap: spacing.sm, marginBottom: spacing.xs },
  sectionLabel: { marginBottom: spacing.xs },
  carousel: { gap: spacing.md, paddingVertical: spacing.xs },
  matchItem: { alignItems: "center", gap: 6, width: 80 },
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    paddingVertical: spacing.sm,
    paddingHorizontal: spacing.xs,
    borderRadius: 16
  },
  rowText: { flex: 1, gap: 2 },
  rowTop: { flexDirection: "row", alignItems: "center", justifyContent: "space-between", gap: spacing.sm },
  preview: { flex: 1 },
  bold: { fontWeight: "600" },
  badge: { minWidth: 22, height: 22, borderRadius: 11, alignItems: "center", justifyContent: "center", paddingHorizontal: 6 }
});
