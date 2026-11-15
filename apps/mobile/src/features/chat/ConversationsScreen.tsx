import React from "react";
import { FlatList, Pressable, ScrollView, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useInfiniteQuery } from "@tanstack/react-query";
import { chatApi } from "../../api/chat";
import { queryKeys } from "../../api/queryKeys";
import type { Conversation } from "../../api/types";
import { errorMessage } from "../../lib/errors";
import { formatListDate } from "../../lib/format";
import type { AppStackParamList } from "../../navigation/types";
import { useRealtimeConnected } from "../../realtime/RealtimeProvider";
import { useSession } from "../../session/SessionProvider";
import { spacing, useTheme } from "../../theme";
import { Avatar, EmptyState, ErrorState, Header, Loading, Screen, Text } from "../../ui";

type Nav = NativeStackNavigationProp<AppStackParamList>;

export function ConversationsScreen() {
  const navigation = useNavigation<Nav>();
  const { colors } = useTheme();
  const { user } = useSession();
  const connected = useRealtimeConnected();
  const conversations = useInfiniteQuery({
    queryKey: queryKeys.conversations,
    queryFn: ({ pageParam }) => chatApi.conversations(pageParam),
    initialPageParam: null as string | null,
    getNextPageParam: (last) => last.nextCursor,
    refetchInterval: connected ? false : 15_000
  });

  const all = conversations.data?.pages.flatMap((p) => p.conversations) ?? [];
  const fresh = all.filter((c) => !c.lastMessage);
  const threads = all.filter((c) => c.lastMessage);
  const open = (c: Conversation) => navigation.navigate("Chat", { conversationId: c.id });

  const renderThread = ({ item }: { item: Conversation }) => {
    const unread = item.unreadCount > 0;
    const mine = item.lastMessage?.senderId === user?.id;
    return (
      <Pressable
        onPress={() => open(item)}
        style={({ pressed }) => [styles.thread, pressed && { opacity: 0.7 }]}
        accessibilityRole="button"
        accessibilityLabel={`Conversation avec ${item.user.firstName}${unread ? `, ${item.unreadCount} message non lu` : ""}`}
        testID="conversation-row"
      >
        <Avatar photo={item.user.photos[0]} name={item.user.firstName} size={56} />
        <View style={styles.threadText}>
          <View style={styles.threadTop}>
            <Text variant="heading" numberOfLines={1} style={styles.flex}>
              {item.user.firstName}
            </Text>
            {item.lastMessage ? (
              <Text variant="caption" color={unread ? "primary" : "textSubtle"}>
                {formatListDate(item.lastMessage.createdAt)}
              </Text>
            ) : null}
          </View>
          <View style={styles.threadTop}>
            <Text variant="caption" color={unread ? "text" : "textMuted"} numberOfLines={1} style={[styles.flex, unread && styles.bold]}>
              {mine ? "Vous : " : ""}
              {item.lastMessage?.body}
            </Text>
            {unread ? (
              <View style={[styles.unread, { backgroundColor: colors.primary }]}>
                <Text variant="micro" color="onPrimary">
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
    <Screen testID="conversations-screen">
      <Header title="Messages" large />
      {conversations.isPending ? (
        <Loading />
      ) : conversations.isError ? (
        <ErrorState message={errorMessage(conversations.error)} onRetry={() => void conversations.refetch()} />
      ) : all.length === 0 ? (
        <EmptyState
          icon="chatbubbles-outline"
          title="Aucun match pour l'instant"
          message="Quand l'intérêt est réciproque, la conversation apparaît ici."
        />
      ) : (
        <FlatList
          data={threads}
          keyExtractor={(c) => c.id}
          renderItem={renderThread}
          refreshing={conversations.isRefetching}
          onRefresh={() => void conversations.refetch()}
          onEndReached={() => conversations.hasNextPage && !conversations.isFetchingNextPage && void conversations.fetchNextPage()}
          onEndReachedThreshold={0.4}
          ItemSeparatorComponent={() => <View style={[styles.separator, { backgroundColor: colors.border }]} />}
          ListHeaderComponent={
            fresh.length ? (
              <View style={styles.freshWrap}>
                <Text variant="micro" color="textSubtle">
                  NOUVEAUX MATCHS
                </Text>
                <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerStyle={styles.fresh}>
                  {fresh.map((c) => (
                    <Pressable
                      key={c.id}
                      onPress={() => open(c)}
                      style={styles.freshItem}
                      accessibilityRole="button"
                      accessibilityLabel={`Nouveau match : ${c.user.firstName}`}
                      testID="new-match"
                    >
                      <Avatar photo={c.user.photos[0]} name={c.user.firstName} size={68} ring />
                      <Text variant="caption" numberOfLines={1}>
                        {c.user.firstName}
                      </Text>
                    </Pressable>
                  ))}
                </ScrollView>
                {threads.length ? (
                  <Text variant="micro" color="textSubtle">
                    CONVERSATIONS
                  </Text>
                ) : null}
              </View>
            ) : null
          }
          ListEmptyComponent={
            <Text color="textMuted" align="center" style={styles.hint}>
              Envoyez le premier message à vos nouveaux matchs !
            </Text>
          }
        />
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  thread: { flexDirection: "row", alignItems: "center", gap: spacing.sm, paddingVertical: spacing.sm },
  threadText: { flex: 1, gap: 3 },
  threadTop: { flexDirection: "row", alignItems: "center", gap: spacing.xs },
  bold: { fontWeight: "700" },
  unread: { minWidth: 20, height: 20, borderRadius: 10, paddingHorizontal: 6, alignItems: "center", justifyContent: "center" },
  separator: { height: StyleSheet.hairlineWidth, marginLeft: 68 },
  freshWrap: { gap: spacing.sm, marginBottom: spacing.sm },
  fresh: { gap: spacing.md, paddingVertical: spacing.xs },
  freshItem: { alignItems: "center", gap: 4, width: 72 },
  hint: { marginTop: spacing.xl }
});
