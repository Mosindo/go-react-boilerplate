import React from "react";
import {
  Alert,
  FlatList,
  Pressable,
  RefreshControl,
  ScrollView,
  StyleSheet,
  View
} from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { chatApi } from "../api/endpoints";
import type { Conversation } from "../api/types";
import { formatTime } from "../lib/format";
import type { MainStackParamList } from "../navigation/types";
import { keys } from "../state/keys";
import { useConversations, useMatches } from "../state/queries";
import { usePalette } from "../theme";
import { Avatar } from "../ui/AuthImage";
import { Screen } from "../ui/Screen";
import { EmptyState, ErrorState, Loading } from "../ui/states";
import { Text } from "../ui/Text";
import { useToast } from "../ui/Toast";

type Nav = NativeStackNavigationProp<MainStackParamList>;

export function MessagesScreen() {
  const navigation = useNavigation<Nav>();
  const p = usePalette();
  const toast = useToast();
  const queryClient = useQueryClient();
  const matches = useMatches();
  const conversations = useConversations();

  const clear = useMutation({
    mutationFn: chatApi.clear,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: keys.conversations }),
    onError: (e: Error) => toast.show(e.message, "error")
  });

  const items = conversations.data?.pages.flatMap((page) => page.conversations) ?? [];
  const matchList = matches.data?.pages.flatMap((page) => page.matches) ?? [];

  const confirmClear = (conversation: Conversation) =>
    Alert.alert(
      `Delete chat with ${conversation.peer.firstName}?`,
      "This removes the messages from your device list only. They keep their copy and you stay matched.",
      [
        { text: "Cancel", style: "cancel" },
        { text: "Delete", style: "destructive", onPress: () => clear.mutate(conversation.id) }
      ]
    );

  const header = (
    <View style={styles.rail}>
      <Text variant="heading" style={styles.pad}>
        New matches
      </Text>
      {matchList.length === 0 ? (
        <Text tone="muted" style={styles.pad}>
          Matches appear here when you like each other.
        </Text>
      ) : (
        <ScrollView
          horizontal
          showsHorizontalScrollIndicator={false}
          contentContainerStyle={styles.railContent}
        >
          {matchList.map((m) => (
            <Pressable
              key={m.id}
              testID={`match-${m.user.userId}`}
              accessibilityRole="button"
              accessibilityLabel={`Open chat with ${m.user.firstName}`}
              style={styles.matchItem}
              onPress={() =>
                navigation.navigate("Chat", {
                  conversationId: m.conversationId,
                  matchId: m.id,
                  peer: {
                    userId: m.user.userId,
                    firstName: m.user.firstName,
                    photoId: m.user.photos[0]?.id
                  }
                })
              }
            >
              <Avatar size={64} photoId={m.user.photos[0]?.id} label={m.user.firstName} />
              <Text variant="caption" numberOfLines={1}>
                {m.user.firstName}
              </Text>
            </Pressable>
          ))}
        </ScrollView>
      )}
      <Text variant="heading" style={styles.pad}>
        Messages
      </Text>
    </View>
  );

  if (conversations.isLoading || matches.isLoading) {
    return (
      <Screen>
        <Loading />
      </Screen>
    );
  }
  if (conversations.isError) {
    return (
      <Screen>
        <ErrorState
          message={conversations.error.message}
          onRetry={() => void conversations.refetch()}
        />
      </Screen>
    );
  }

  return (
    <Screen>
      <FlatList
        data={items}
        keyExtractor={(c) => c.id}
        ListHeaderComponent={header}
        ListEmptyComponent={
          <EmptyState
            title="No conversations yet"
            message="Say hello to one of your matches to start chatting."
          />
        }
        refreshControl={
          <RefreshControl
            refreshing={conversations.isRefetching}
            onRefresh={() => {
              void conversations.refetch();
              void matches.refetch();
            }}
            tintColor={p.primary}
          />
        }
        onEndReached={() => conversations.hasNextPage && void conversations.fetchNextPage()}
        renderItem={({ item }) => (
          <Pressable
            testID={`conversation-${item.peer.userId}`}
            accessibilityRole="button"
            accessibilityLabel={`Chat with ${item.peer.firstName}${item.unreadCount ? `, ${item.unreadCount} unread` : ""}`}
            onPress={() =>
              navigation.navigate("Chat", {
                conversationId: item.id,
                matchId: item.matchId,
                peer: item.peer
              })
            }
            onLongPress={() => confirmClear(item)}
            style={({ pressed }) => [styles.row, { opacity: pressed ? 0.7 : 1 }]}
          >
            <Avatar size={56} photoId={item.peer.photoId} label={item.peer.firstName} />
            <View style={styles.rowBody}>
              <View style={styles.rowTop}>
                <Text variant="label" style={styles.flex} numberOfLines={1}>
                  {item.peer.firstName}
                </Text>
                <Text variant="caption" tone="muted">
                  {formatTime(item.updatedAt)}
                </Text>
              </View>
              <Text
                numberOfLines={1}
                tone={item.unreadCount > 0 ? "default" : "muted"}
                style={item.unreadCount > 0 ? styles.bold : undefined}
              >
                {item.lastMessage?.body ?? ""}
              </Text>
            </View>
            {item.unreadCount > 0 ? (
              <View style={[styles.badge, { backgroundColor: p.primary }]}>
                <Text variant="caption" style={{ color: p.primaryText, fontWeight: "700" }}>
                  {item.unreadCount}
                </Text>
              </View>
            ) : null}
          </Pressable>
        )}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  rail: { gap: 12, paddingTop: 12 },
  pad: { paddingHorizontal: 20 },
  railContent: { paddingHorizontal: 20, gap: 14 },
  matchItem: { alignItems: "center", gap: 4, width: 68 },
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: 14,
    paddingHorizontal: 20,
    paddingVertical: 10
  },
  rowBody: { flex: 1, gap: 2 },
  rowTop: { flexDirection: "row", alignItems: "center" },
  flex: { flex: 1 },
  bold: { fontWeight: "700" },
  badge: {
    minWidth: 22,
    height: 22,
    borderRadius: 11,
    alignItems: "center",
    justifyContent: "center",
    paddingHorizontal: 6
  }
});
