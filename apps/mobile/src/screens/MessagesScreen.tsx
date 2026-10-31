import React from "react";
import { FlatList, Pressable, RefreshControl, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useQuery } from "@tanstack/react-query";
import { getConversations, getReceivedLikes } from "../api/dating";
import type { Card, Conversation } from "../api/types";
import { RemotePhoto } from "../components/RemotePhoto";
import { formatTime } from "../lib/format";
import { queryKeys } from "../lib/queryKeys";
import type { MainStackParamList } from "../navigation/types";
import { Screen } from "../shared/layout";
import { EmptyView, ErrorView, LoadingView } from "../shared/feedback";
import { Badge, Text, colors, radii, spacing } from "../shared/ui";

function LikesRow({ cards, onOpen }: { cards: Card[]; onOpen: (c: Card) => void }) {
  if (cards.length === 0) return null;
  return (
    <View style={styles.likes}>
      <Text tone="muted" variant="eyebrow" weight="bold">
        Ils vous ont liké ({cards.length})
      </Text>
      <FlatList
        data={cards}
        horizontal
        keyExtractor={(c) => c.id}
        renderItem={({ item }) => (
          <Pressable accessibilityLabel={`${item.firstName}, ${item.age} ans`} onPress={() => onOpen(item)} style={styles.likeItem}>
            <RemotePhoto style={styles.likePhoto} url={item.photos[0]?.url} />
            <Text variant="caption" weight="semibold">
              {item.firstName}
            </Text>
          </Pressable>
        )}
        showsHorizontalScrollIndicator={false}
      />
    </View>
  );
}

function ConversationRow({ item, onPress }: { item: Conversation; onPress: () => void }) {
  const unread = item.unreadCount > 0;
  return (
    <Pressable accessibilityRole="button" onPress={onPress} style={styles.row}>
      <RemotePhoto style={styles.avatar} url={item.user.photos[0]?.url} />
      <View style={styles.rowBody}>
        <View style={styles.rowTop}>
          <Text weight="bold">{item.user.firstName}</Text>
          <Text tone="muted" variant="caption">
            {formatTime(item.sortAt)}
          </Text>
        </View>
        <Text numberOfLines={1} tone={unread ? "default" : "muted"} weight={unread ? "semibold" : "regular"}>
          {item.lastMessage ? item.lastMessage.body : "Nouveau match — dites bonjour !"}
        </Text>
      </View>
      {unread ? <Badge label={String(item.unreadCount)} variant="primary" /> : null}
    </Pressable>
  );
}

export default function MessagesScreen() {
  const navigation = useNavigation<NativeStackNavigationProp<MainStackParamList>>();
  const conversations = useQuery({ queryKey: queryKeys.conversations, queryFn: getConversations });
  const likes = useQuery({ queryKey: queryKeys.received, queryFn: getReceivedLikes });

  const refreshing = conversations.isRefetching || likes.isRefetching;
  const refresh = () => {
    void conversations.refetch();
    void likes.refetch();
  };

  if (conversations.isLoading)
    return (
      <Screen>
        <LoadingView label="Chargement des conversations…" />
      </Screen>
    );
  if (conversations.isError) {
    return (
      <Screen>
        <ErrorView actionLabel="Réessayer" message="Impossible de charger vos conversations." onAction={refresh} />
      </Screen>
    );
  }

  return (
    <Screen padded={false}>
      <FlatList
        ListEmptyComponent={
          <EmptyView
            message="Likez des profils : dès qu'un like est réciproque, la conversation s'ouvre ici."
            title="Pas encore de match"
          />
        }
        ListHeaderComponent={
          <View style={styles.header}>
            <Text variant="heading" weight="bold">
              Messages
            </Text>
            <LikesRow cards={likes.data ?? []} onOpen={(c) => navigation.navigate("ProfileDetail", { id: c.id, swipeable: true })} />
          </View>
        }
        contentContainerStyle={styles.list}
        data={conversations.data ?? []}
        keyExtractor={(c) => c.id}
        refreshControl={<RefreshControl onRefresh={refresh} refreshing={refreshing} tintColor={colors.primary} />}
        renderItem={({ item }) => <ConversationRow item={item} onPress={() => navigation.navigate("Chat", { conversationId: item.id })} />}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  list: { paddingHorizontal: spacing.lg, paddingTop: spacing.md, paddingBottom: spacing.xxxl, gap: spacing.sm },
  header: { gap: spacing.md, paddingBottom: spacing.sm },
  likes: { gap: spacing.sm },
  likeItem: { alignItems: "center", marginRight: spacing.md, gap: spacing.xxs },
  likePhoto: { width: 64, height: 64, borderRadius: 32, overflow: "hidden", borderWidth: 2, borderColor: colors.gold },
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    padding: spacing.md,
    borderRadius: radii.lg,
    backgroundColor: colors.surface,
    borderWidth: 1,
    borderColor: colors.border
  },
  avatar: { width: 56, height: 56, borderRadius: 28, overflow: "hidden" },
  rowBody: { flex: 1, gap: 2 },
  rowTop: { flexDirection: "row", justifyContent: "space-between" }
});
