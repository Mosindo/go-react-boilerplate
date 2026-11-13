import React, { useMemo } from "react";
import { FlatList, Pressable, RefreshControl, ScrollView, StyleSheet, View } from "react-native";
import type { Conversation } from "../../api/types";
import type { TabScreenProps } from "../../navigation/types";
import { useConversations } from "../../state/queries";
import { useSession } from "../../state/session";
import { useTheme } from "../../theme/ThemeProvider";
import { Avatar, Badge, EmptyState, ErrorState, LoadingState, Screen, Text } from "../../ui";
import { describeError } from "../../utils/errors";
import { formatShortTime } from "../../utils/format";

export default function MatchesScreen({ navigation }: TabScreenProps<"Matches">) {
  const { colors, spacing } = useTheme();
  const { userId } = useSession();
  const conversations = useConversations();

  const [fresh, active] = useMemo(() => {
    const list = conversations.data ?? [];
    return [list.filter((c) => !c.lastMessage), list.filter((c) => c.lastMessage)];
  }, [conversations.data]);

  const open = (c: Conversation) =>
    navigation.navigate("Chat", { userId: c.user.userId, firstName: c.user.firstName, photoUrl: c.user.photoUrl });

  if (conversations.isLoading) {
    return (
      <Screen edges={["top"]}>
        <LoadingState />
      </Screen>
    );
  }
  if (conversations.isError) {
    return (
      <Screen edges={["top"]}>
        <ErrorState message={describeError(conversations.error)} onAction={() => void conversations.refetch()} />
      </Screen>
    );
  }

  const header = (
    <View style={{ gap: spacing.md, paddingTop: spacing.sm }}>
      <Text variant="title" accessibilityRole="header">
        Messages
      </Text>
      {fresh.length > 0 ? (
        <View style={{ gap: spacing.sm }}>
          <Text variant="label" weight="semibold" tone="muted">
            Nouveaux matchs
          </Text>
          <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerStyle={{ gap: spacing.md }}>
            {fresh.map((c) => (
              <Pressable
                key={c.matchId}
                onPress={() => open(c)}
                accessibilityRole="button"
                accessibilityLabel={`Nouveau match : ${c.user.firstName}`}
                style={styles.newMatch}
                testID={`new-match-${c.user.userId}`}
              >
                <View style={[styles.ring, { borderColor: colors.primary }]}>
                  <Avatar url={c.user.photoUrl} name={c.user.firstName} size={64} />
                </View>
                <Text variant="label" weight="medium" numberOfLines={1}>
                  {c.user.firstName}
                </Text>
              </Pressable>
            ))}
          </ScrollView>
        </View>
      ) : null}
      {active.length > 0 ? (
        <Text variant="label" weight="semibold" tone="muted">
          Conversations
        </Text>
      ) : null}
    </View>
  );

  return (
    <Screen edges={["top"]} testID="matches-screen">
      <FlatList
        data={active}
        keyExtractor={(c) => c.matchId}
        ListHeaderComponent={header}
        refreshControl={
          <RefreshControl
            refreshing={conversations.isRefetching}
            onRefresh={() => void conversations.refetch()}
            tintColor={colors.primary}
          />
        }
        ListEmptyComponent={
          fresh.length === 0 ? (
            <EmptyState
              icon="chatbubbles-outline"
              title="Pas encore de match"
              message="Quand l'intérêt est réciproque, la conversation apparaît ici."
              actionLabel="Découvrir des profils"
              onAction={() => navigation.navigate("Discover")}
            />
          ) : (
            <Text tone="muted" style={{ marginTop: spacing.lg }}>
              Brisez la glace avec vos nouveaux matchs !
            </Text>
          )
        }
        renderItem={({ item }) => (
          <Pressable
            onPress={() => open(item)}
            accessibilityRole="button"
            accessibilityLabel={`Conversation avec ${item.user.firstName}${item.unreadCount > 0 ? `, ${item.unreadCount} non lus` : ""}`}
            style={({ pressed }) => [styles.row, { paddingVertical: spacing.sm, opacity: pressed ? 0.7 : 1 }]}
            testID={`conversation-${item.user.userId}`}
          >
            <Avatar url={item.user.photoUrl} name={item.user.firstName} size={56} />
            <View style={styles.flex}>
              <View style={styles.titleRow}>
                <Text weight="semibold" style={styles.flex} numberOfLines={1}>
                  {item.user.firstName}
                </Text>
                {item.lastMessage ? (
                  <Text variant="caption" tone="subtle">
                    {formatShortTime(item.lastMessage.createdAt)}
                  </Text>
                ) : null}
              </View>
              <View style={styles.titleRow}>
                <Text
                  tone={item.unreadCount > 0 ? "default" : "muted"}
                  weight={item.unreadCount > 0 ? "semibold" : "regular"}
                  numberOfLines={1}
                  style={styles.flex}
                >
                  {item.lastMessage?.senderUserId === userId ? "Vous : " : ""}
                  {item.lastMessage?.content}
                </Text>
                <Badge count={item.unreadCount} />
              </View>
            </View>
          </Pressable>
        )}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  newMatch: { alignItems: "center", gap: 6, width: 76 },
  ring: { borderWidth: 2.5, borderRadius: 40, padding: 3 },
  row: { flexDirection: "row", alignItems: "center", gap: 14 },
  titleRow: { flexDirection: "row", alignItems: "center", gap: 8 }
});
