import React, { useMemo } from "react";
import { FlatList, Pressable, RefreshControl, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import type { Conversation, Match } from "../../api/types";
import { useAuth } from "../../auth/AuthContext";
import { useConversations, useMatches } from "../../hooks/queries";
import { formatListTime } from "../../lib/format";
import type { RootParamList } from "../../navigation/types";
import { radii, spacing, useTheme } from "../../theme/theme";
import { EmptyState, ErrorState, LoadingView } from "../../ui/Feedback";
import { Avatar } from "../../ui/Photo";
import { Screen } from "../../ui/Screen";
import { Text } from "../../ui/Text";

type Nav = NativeStackNavigationProp<RootParamList>;

function NewMatches({ matches }: { matches: Match[] }) {
  const nav = useNavigation<Nav>();
  const { colors } = useTheme();
  if (matches.length === 0) return null;
  return (
    <View style={styles.rail}>
      <Text variant="label" tone="textSubtle" style={styles.railTitle}>
        NOUVEAUX MATCHS
      </Text>
      <FlatList
        horizontal
        data={matches}
        keyExtractor={(m) => m.id}
        showsHorizontalScrollIndicator={false}
        contentContainerStyle={styles.railContent}
        renderItem={({ item }) => (
          <Pressable
            accessibilityRole="button"
            accessibilityLabel={`Nouveau match : ${item.user.firstName}`}
            onPress={() => nav.navigate("Chat", { conversationId: item.conversationId, peer: item.user })}
            style={styles.railItem}
            testID={`new-match-${item.user.firstName}`}
          >
            <View style={[styles.ring, { borderColor: colors.primary }]}>
              <Avatar path={item.user.photos[0]?.url} name={item.user.firstName} size={64} />
            </View>
            <Text variant="caption" numberOfLines={1} style={styles.railName}>
              {item.user.firstName}
            </Text>
          </Pressable>
        )}
      />
    </View>
  );
}

function ConversationRow({ item, meId }: { item: Conversation; meId: string }) {
  const nav = useNavigation<Nav>();
  const { colors } = useTheme();
  const unread = item.unreadCount > 0;
  const last = item.lastMessage;
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={`Conversation avec ${item.peer.firstName}${unread ? `, ${item.unreadCount} non lu` : ""}`}
      onPress={() => nav.navigate("Chat", { conversationId: item.id, peer: item.peer })}
      style={({ pressed }) => [
        styles.row,
        { backgroundColor: pressed ? colors.surfaceMuted : "transparent" }
      ]}
      testID={`conversation-${item.peer.firstName}`}
    >
      <Avatar path={item.peer.photos[0]?.url} name={item.peer.firstName} size={56} />
      <View style={styles.rowBody}>
        <View style={styles.rowTop}>
          <Text variant="heading" style={styles.flex} numberOfLines={1}>
            {item.peer.firstName}
          </Text>
          {last ? (
            <Text variant="caption" tone={unread ? "primary" : "textSubtle"}>
              {formatListTime(last.createdAt)}
            </Text>
          ) : null}
        </View>
        <View style={styles.rowTop}>
          <Text
            tone={unread ? "text" : "textMuted"}
            numberOfLines={1}
            style={[styles.flex, unread && styles.bold]}
          >
            {last ? `${last.senderId === meId ? "Toi : " : ""}${last.body}` : ""}
          </Text>
          {unread ? (
            <View style={[styles.unread, { backgroundColor: colors.primary }]}>
              <Text variant="caption" tone="onPrimary" style={styles.unreadText}>
                {item.unreadCount > 9 ? "9+" : item.unreadCount}
              </Text>
            </View>
          ) : null}
        </View>
      </View>
    </Pressable>
  );
}

export function MessagesScreen() {
  const { user } = useAuth();
  const conversations = useConversations();
  const matches = useMatches();

  const convs = useMemo(
    () => conversations.data?.pages.flatMap((p) => p.conversations) ?? [],
    [conversations.data]
  );
  const fresh = useMemo(
    () => (matches.data?.pages.flatMap((p) => p.matches) ?? []).filter((m) => !m.hasMessages),
    [matches.data]
  );

  if (conversations.isPending || matches.isPending) {
    return (
      <Screen edges={["top"]}>
        <LoadingView />
      </Screen>
    );
  }
  if (conversations.isError) {
    return (
      <Screen edges={["top"]}>
        <ErrorState message={conversations.error.message} onRetry={() => void conversations.refetch()} />
      </Screen>
    );
  }

  return (
    <Screen padded={false} edges={["top"]}>
      <Text variant="title" style={styles.title}>
        Messages
      </Text>
      <FlatList
        data={convs}
        keyExtractor={(c) => c.id}
        ListHeaderComponent={<NewMatches matches={fresh} />}
        renderItem={({ item }) => <ConversationRow item={item} meId={user?.id ?? ""} />}
        ListEmptyComponent={
          fresh.length === 0 ? (
            <EmptyState
              icon="chatbubbles-outline"
              title="Aucune conversation"
              message="Quand un like sera réciproque, vous pourrez commencer à discuter ici."
            />
          ) : (
            <View style={styles.hint}>
              <Text tone="textMuted" align="center">
                Choisis un nouveau match pour lancer la conversation.
              </Text>
            </View>
          )
        }
        onEndReached={() => {
          if (conversations.hasNextPage && !conversations.isFetchingNextPage)
            void conversations.fetchNextPage();
        }}
        onEndReachedThreshold={0.4}
        refreshControl={
          <RefreshControl
            refreshing={conversations.isRefetching}
            onRefresh={() => {
              void conversations.refetch();
              void matches.refetch();
            }}
          />
        }
        contentContainerStyle={convs.length === 0 && fresh.length === 0 ? styles.emptyGrow : undefined}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  bold: { fontWeight: "600" },
  title: { paddingHorizontal: spacing.md, paddingTop: spacing.xs, paddingBottom: spacing.sm },
  rail: { marginBottom: spacing.sm },
  railTitle: { paddingHorizontal: spacing.md, letterSpacing: 1, marginBottom: spacing.xs },
  railContent: { paddingHorizontal: spacing.md, gap: spacing.md },
  railItem: { alignItems: "center", width: 72 },
  ring: { borderWidth: 2, borderRadius: radii.pill, padding: 2 },
  railName: { marginTop: 4 },
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.sm,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm
  },
  rowBody: { flex: 1, gap: 2 },
  rowTop: { flexDirection: "row", alignItems: "center", gap: spacing.xs },
  unread: {
    minWidth: 20,
    height: 20,
    borderRadius: 10,
    alignItems: "center",
    justifyContent: "center",
    paddingHorizontal: 5
  },
  unreadText: { fontWeight: "700", fontSize: 11 },
  hint: { padding: spacing.lg },
  emptyGrow: { flexGrow: 1 }
});
