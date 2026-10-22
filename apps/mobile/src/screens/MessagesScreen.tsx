import React from "react";
import { FlatList, Pressable, ScrollView, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useQuery } from "@tanstack/react-query";
import { errorMessage } from "../api/client";
import { fetchMatches } from "../api/discovery";
import type { Conversation } from "../api/types";
import { PersonAvatar } from "../components/PersonAvatar";
import { useAuth } from "../hooks/useAuth";
import { useConversationsQuery } from "../hooks/useFeeds";
import { formatShortTime } from "../lib/format";
import type { RootStackParamList } from "../navigation/types";
import { EmptyView, ErrorView, LoadingView } from "../shared/feedback";
import { ScreenContainer } from "../shared/layout";
import { Badge, ListItem, Text, spacing } from "../shared/ui";

export default function MessagesScreen() {
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>();
  const { user } = useAuth();
  const matches = useQuery({ queryKey: ["matches"], queryFn: () => fetchMatches(), staleTime: 30_000 });
  const conversations = useConversationsQuery();

  const open = (conversationId: string, peerId: string, name: string) => navigation.navigate("Chat", { conversationId, peerId, name });
  const fresh = (matches.data?.matches ?? []).filter((match) => !match.hasMessages);
  const rows = conversations.data?.pages.flatMap((page) => page.conversations) ?? [];

  if (conversations.isLoading && matches.isLoading) {
    return (
      <ScreenContainer edges={["top", "left", "right"]}>
        <LoadingView label="Chargement…" />
      </ScreenContainer>
    );
  }
  if (conversations.isError) {
    return (
      <ScreenContainer edges={["top", "left", "right"]}>
        <ErrorView actionLabel="Réessayer" message={errorMessage(conversations.error)} onAction={() => void conversations.refetch()} />
      </ScreenContainer>
    );
  }

  const renderConversation = ({ item }: { item: Conversation }) => {
    const last = item.lastMessage;
    const mine = last?.senderId === user?.id;
    return (
      <ListItem
        accessibilityLabel={`Conversation avec ${item.peer.firstName}${item.unreadCount ? `, ${item.unreadCount} non lus` : ""}`}
        leading={<PersonAvatar name={item.peer.firstName} photo={item.peer.photo} size={52} />}
        onPress={() => open(item.id, item.peer.userId, item.peer.firstName)}
        style={styles.row}
        subtitle={last ? `${mine ? "Vous : " : ""}${last.content}` : undefined}
        testID={`conversation-${item.peer.firstName}`}
        title={item.peer.firstName}
        trailing={
          <View style={styles.trailing}>
            {last ? (
              <Text tone="muted" variant="caption">
                {formatShortTime(last.createdAt)}
              </Text>
            ) : null}
            {item.unreadCount > 0 ? <Badge label={String(item.unreadCount)} variant="primary" /> : null}
          </View>
        }
      />
    );
  };

  return (
    <ScreenContainer contentMaxWidth={640} edges={["top", "left", "right"]} testID="messages-screen">
      <FlatList
        ItemSeparatorComponent={() => <View style={styles.separator} />}
        ListEmptyComponent={
          fresh.length === 0 ? (
            <EmptyView
              message="Dès qu'un like est réciproque, votre match apparaît ici et vous pouvez discuter librement."
              testID="messages-empty"
              title="Pas encore de conversation"
            />
          ) : (
            <Text tone="muted">Choisissez un nouveau match pour envoyer le premier message.</Text>
          )
        }
        ListHeaderComponent={
          <View style={styles.headerBlock}>
            <Text accessibilityRole="header" variant="heading" weight="bold">
              Messages
            </Text>
            {fresh.length > 0 ? (
              <View style={styles.fresh}>
                <Text tone="secondary" variant="label" weight="semibold">
                  Nouveaux matchs
                </Text>
                <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerStyle={styles.freshRow} testID="new-matches">
                  {fresh.map((match) => (
                    <Pressable
                      accessibilityLabel={`Nouveau match : ${match.user.firstName}`}
                      accessibilityRole="button"
                      key={match.id}
                      onPress={() => open(match.conversationId, match.user.userId, match.user.firstName)}
                      style={styles.freshItem}
                      testID={`match-${match.user.firstName}`}
                    >
                      <PersonAvatar name={match.user.firstName} photo={match.user.photo} size={64} />
                      <Text variant="caption" weight="semibold" numberOfLines={1}>
                        {match.user.firstName}
                      </Text>
                    </Pressable>
                  ))}
                </ScrollView>
              </View>
            ) : null}
          </View>
        }
        contentContainerStyle={styles.list}
        data={rows}
        keyExtractor={(item) => item.id}
        onEndReached={() => {
          if (conversations.hasNextPage && !conversations.isFetchingNextPage) {
            void conversations.fetchNextPage();
          }
        }}
        onRefresh={() => {
          void conversations.refetch();
          void matches.refetch();
        }}
        refreshing={conversations.isRefetching}
        renderItem={renderConversation}
      />
    </ScreenContainer>
  );
}

const styles = StyleSheet.create({
  list: { paddingBottom: spacing.xl, gap: 0 },
  headerBlock: { gap: spacing.lg, marginBottom: spacing.md },
  fresh: { gap: spacing.sm },
  freshRow: { gap: spacing.md, paddingVertical: spacing.xs },
  freshItem: { alignItems: "center", gap: spacing.xxs, width: 72 },
  row: { paddingVertical: spacing.sm },
  separator: { height: spacing.sm },
  trailing: { alignItems: "flex-end", gap: spacing.xs }
});
