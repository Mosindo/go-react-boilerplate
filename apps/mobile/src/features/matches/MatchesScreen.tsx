import React, { useCallback, useMemo } from "react";
import { FlatList, RefreshControl, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import type { ConversationItem, MatchItem } from "../../api";
import { formatListTimestamp } from "../../shared/format";
import { flattenPages } from "../../shared/pagination";
import { Avatar, Badge, EmptyState, ErrorState, ListRow, Screen, Skeleton, Text } from "../../shared/ui";
import { spacing } from "../../theme";
import type { MainStackParamList } from "../../navigation/types";
import { useCurrentUser } from "../auth/AuthProvider";
import { useConversations, useMatches } from "./hooks";
import { selectNewMatches } from "./newMatches";
import { NewMatchesStrip } from "./NewMatchesStrip";

type Navigation = NativeStackNavigationProp<MainStackParamList>;

const ConversationRow = React.memo(function ConversationRow({
  item,
  currentUserId,
  onOpen
}: {
  item: ConversationItem;
  currentUserId: string;
  onOpen: (item: ConversationItem) => void;
}) {
  const last = item.lastMessage;
  const preview = `${last.senderId === currentUserId ? "Vous : " : ""}${last.body}`;
  const unread = item.unreadCount > 0;
  return (
    <ListRow
      accessibilityLabel={`${item.otherUser.firstName}. ${preview}${unread ? `. ${item.unreadCount} non lus` : ""}`}
      left={<Avatar name={item.otherUser.firstName} size={56} thumbUrl={item.otherUser.thumbUrl} />}
      onPress={() => onOpen(item)}
      right={
        <View style={styles.meta}>
          <Text color="textMuted" variant="caption">
            {formatListTimestamp(last.createdAt)}
          </Text>
          <Badge count={item.unreadCount} />
        </View>
      }
      subtitle={preview}
      testID={`conversation-${item.id}`}
      title={item.otherUser.firstName}
    />
  );
});

export function MatchesScreen() {
  const navigation = useNavigation<Navigation>();
  const { id: currentUserId } = useCurrentUser();
  const matchesQuery = useMatches();
  const conversationsQuery = useConversations();

  const conversations = useMemo(() => flattenPages(conversationsQuery.data), [conversationsQuery.data]);
  const matches = useMemo(() => flattenPages(matchesQuery.data), [matchesQuery.data]);
  const newMatches = useMemo(() => selectNewMatches(matches, conversations), [matches, conversations]);

  const openConversation = useCallback(
    (item: ConversationItem) =>
      navigation.navigate("Chat", {
        conversationId: item.id,
        userId: item.otherUser.userId,
        firstName: item.otherUser.firstName,
        thumbUrl: item.otherUser.thumbUrl,
        matchId: item.matchId
      }),
    [navigation]
  );

  const openMatch = useCallback(
    (match: MatchItem) =>
      navigation.navigate("Chat", {
        conversationId: match.conversationId,
        userId: match.profile.userId,
        firstName: match.profile.firstName,
        thumbUrl: match.profile.photos[0]?.thumbUrl ?? null,
        matchId: match.matchId
      }),
    [navigation]
  );

  const refresh = useCallback(() => {
    void matchesQuery.refetch();
    void conversationsQuery.refetch();
  }, [matchesQuery, conversationsQuery]);

  const renderItem = useCallback(
    ({ item }: { item: ConversationItem }) => (
      <ConversationRow currentUserId={currentUserId} item={item} onOpen={openConversation} />
    ),
    [currentUserId, openConversation]
  );

  if (conversationsQuery.isError && matchesQuery.isError) {
    return (
      <Screen edges={["top"]}>
        <ErrorState onRetry={refresh} />
      </Screen>
    );
  }

  const loading = conversationsQuery.isLoading || matchesQuery.isLoading;
  const empty = !loading && newMatches.length === 0 && conversations.length === 0;

  return (
    <Screen edges={["top"]} padded={false} testID="matches-screen">
      <Text style={styles.title} variant="title">
        Matchs
      </Text>
      {loading ? (
        <View style={styles.skeletons}>
          {[0, 1, 2, 3].map((key) => (
            <Skeleton height={56} key={key} />
          ))}
        </View>
      ) : empty ? (
        <EmptyState
          icon="heart-outline"
          message="Continuez à explorer : vos prochains matchs apparaîtront ici."
          title="Pas encore de match"
        />
      ) : (
        <FlatList
          ListHeaderComponent={
            <View style={styles.header}>
              {newMatches.length > 0 ? (
                <NewMatchesStrip
                  matches={newMatches}
                  onEndReached={() => {
                    if (matchesQuery.hasNextPage && !matchesQuery.isFetchingNextPage) void matchesQuery.fetchNextPage();
                  }}
                  onOpen={openMatch}
                />
              ) : null}
              {conversations.length > 0 ? <Text variant="heading">Messages</Text> : null}
            </View>
          }
          contentContainerStyle={styles.list}
          data={conversations}
          keyExtractor={(item) => item.id}
          onEndReached={() => {
            if (conversationsQuery.hasNextPage && !conversationsQuery.isFetchingNextPage)
              void conversationsQuery.fetchNextPage();
          }}
          onEndReachedThreshold={0.5}
          refreshControl={<RefreshControl onRefresh={refresh} refreshing={conversationsQuery.isRefetching} />}
          renderItem={renderItem}
        />
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  title: { paddingHorizontal: spacing.xl, paddingTop: spacing.sm, paddingBottom: spacing.md },
  header: { gap: spacing.lg, paddingBottom: spacing.sm },
  list: { paddingHorizontal: spacing.xl, paddingBottom: spacing.xxl },
  skeletons: { paddingHorizontal: spacing.xl, gap: spacing.lg },
  meta: { alignItems: "flex-end", gap: spacing.xs }
});
