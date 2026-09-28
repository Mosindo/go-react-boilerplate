import React, { useCallback } from "react";
import { FlatList, RefreshControl, StyleSheet, View } from "react-native";
import type { BottomTabScreenProps } from "@react-navigation/bottom-tabs";
import type { CompositeScreenProps } from "@react-navigation/native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { errorMessage } from "../api/client";
import type { MatchSummary } from "../api/types";
import { formatListTime } from "../domain/chat";
import { flattenMatches, useMatches } from "../hooks/useMatches";
import { useMe } from "../hooks/useAuth";
import type { MainStackParamList, TabParamList } from "../navigation/types";
import { EmptyView, ErrorView, ListSkeleton } from "../shared/feedback";
import { Header, SafeAreaLayout } from "../shared/layout";
import { Avatar, Badge, ListItem, Loader, Text } from "../shared/ui";
import { spacing, useTheme } from "../theme";

type Props = CompositeScreenProps<
  BottomTabScreenProps<TabParamList, "Matches">,
  NativeStackScreenProps<MainStackParamList>
>;

export default function MatchesScreen({ navigation }: Props) {
  const theme = useTheme();
  const myId = useMe().data?.id;
  const query = useMatches();
  const matches = flattenMatches(query.data);

  const open = useCallback(
    (match: MatchSummary) =>
      navigation.navigate("Chat", {
        conversationId: match.conversationId,
        matchId: match.matchId,
        userId: match.user.userId,
        name: match.user.firstName
      }),
    [navigation]
  );

  const renderItem = ({ item }: { item: MatchSummary }) => {
    const unread = item.unreadCount > 0;
    const preview = item.lastMessage
      ? `${item.lastMessage.senderId === myId ? "You: " : ""}${item.lastMessage.body}`
      : "You matched. Say hello!";
    return (
      <ListItem
        accessibilityHint="Opens the conversation"
        bold={unread}
        left={<Avatar name={item.user.firstName} photo={item.user.photo} size={52} />}
        onPress={() => open(item)}
        right={
          <View style={styles.right}>
            <Text tone="muted" variant="caption">
              {formatListTime(item.lastMessage?.createdAt ?? item.createdAt)}
            </Text>
            <Badge count={item.unreadCount} label={`${item.unreadCount} unread messages`} />
          </View>
        }
        subtitle={preview}
        title={`${item.user.firstName}, ${item.user.age}`}
      />
    );
  };

  return (
    <SafeAreaLayout edges={["top"]}>
      <View style={styles.header}>
        <Header title="Matches" />
      </View>
      {query.isPending ? (
        <ListSkeleton />
      ) : query.isError && matches.length === 0 ? (
        <ErrorView message={errorMessage(query.error)} onRetry={() => void query.refetch()} />
      ) : (
        <FlatList
          contentContainerStyle={matches.length === 0 ? styles.emptyList : undefined}
          data={matches}
          keyExtractor={(item) => item.matchId}
          ListEmptyComponent={
            <EmptyView
              actionLabel="Go discover"
              message="When you and someone else like each other, your conversation appears here."
              onAction={() => navigation.navigate("Discover")}
              title="No matches yet"
            />
          }
          ListFooterComponent={query.isFetchingNextPage ? <Loader /> : null}
          onEndReached={() => {
            if (query.hasNextPage && !query.isFetchingNextPage) {
              void query.fetchNextPage();
            }
          }}
          refreshControl={
            <RefreshControl
              onRefresh={() => void query.refetch()}
              refreshing={query.isRefetching && !query.isFetchingNextPage}
              tintColor={theme.primary}
            />
          }
          renderItem={renderItem}
        />
      )}
    </SafeAreaLayout>
  );
}

const styles = StyleSheet.create({
  header: { paddingHorizontal: spacing.lg, paddingVertical: spacing.sm },
  emptyList: { flexGrow: 1 },
  right: { alignItems: "flex-end", gap: spacing.xs }
});
