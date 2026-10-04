import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import React, { useMemo, useState } from "react";
import { FlatList, RefreshControl, StyleSheet, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { getErrorMessage } from "../../api/client";
import type { MatchSummary } from "../../api/types";
import { useCurrentUserId } from "../../hooks/useAuth";
import { useConversations, useHideConversation, useMatches, useUnmatch } from "../../hooks/useChat";
import type { MainStackParamList } from "../../navigation/types";
import { spacing, useColors } from "../../theme";
import { ActionSheet, ConfirmDialog, EmptyState, ErrorState, ListSkeleton, Text, toast } from "../../ui";
import { ConversationRow } from "./ConversationRow";
import { NewMatchesRow } from "./NewMatchesRow";

type Nav = NativeStackNavigationProp<MainStackParamList>;
type Pending = { kind: "hide" | "unmatch"; item: MatchSummary };

export function MatchesScreen() {
  const navigation = useNavigation<Nav>();
  const colors = useColors();
  const myUserId = useCurrentUserId();
  const conversations = useConversations();
  const matches = useMatches();
  const hide = useHideConversation();
  const unmatch = useUnmatch();
  const [menuFor, setMenuFor] = useState<MatchSummary | null>(null);
  const [pending, setPending] = useState<Pending | null>(null);

  const newMatches = useMemo(() => matches.items.filter((m) => !m.lastMessage), [matches.items]);
  const refreshing = (conversations.isRefetching || matches.isRefetching) && !conversations.isFetchingNextPage;

  const openChat = (match: MatchSummary) => navigation.navigate("Chat", { match });

  async function confirm() {
    if (!pending) return;
    const { kind, item } = pending;
    try {
      if (kind === "hide") await hide.mutateAsync(item.conversationId);
      else await unmatch.mutateAsync(item);
      toast.success(kind === "hide" ? "Conversation hidden" : `You unmatched ${item.user.displayName}`);
    } catch (e) {
      toast.error(getErrorMessage(e));
    } finally {
      setPending(null);
    }
  }

  let body: React.ReactNode;
  if (conversations.isPending && matches.isPending) {
    body = <ListSkeleton />;
  } else if (conversations.isError && conversations.items.length === 0) {
    body = <ErrorState error={conversations.error} onRetry={() => void conversations.refetch()} />;
  } else if (conversations.items.length === 0 && newMatches.length === 0) {
    body = (
      <EmptyState
        actionLabel="Go to Discover"
        icon="chatbubbles-outline"
        message="When you and someone else like each other, you'll find them here."
        onAction={() => navigation.navigate("Tabs", { screen: "Discover" })}
        title="No matches yet"
      />
    );
  } else {
    body = (
      <FlatList
        ListEmptyComponent={
          <Text center style={styles.noChats} tone="muted">
            No conversations yet. Say hello to one of your new matches!
          </Text>
        }
        ListFooterComponent={
          conversations.isFetchingNextPage ? <ListSkeleton rows={2} /> : <View style={{ height: spacing.lg }} />
        }
        ListHeaderComponent={<NewMatchesRow matches={newMatches} onPress={openChat} />}
        data={conversations.items}
        keyExtractor={(item) => item.id}
        onEndReached={() => {
          if (conversations.hasNextPage && !conversations.isFetchingNextPage) void conversations.fetchNextPage();
        }}
        onEndReachedThreshold={0.4}
        refreshControl={
          <RefreshControl
            onRefresh={() => {
              void conversations.refetch();
              void matches.refetch();
            }}
            refreshing={refreshing}
            tintColor={colors.primary}
          />
        }
        renderItem={({ item }) => (
          <ConversationRow
            item={item}
            myUserId={myUserId}
            onMore={() => setMenuFor(item)}
            onPress={() => openChat(item)}
          />
        )}
      />
    );
  }

  return (
    <SafeAreaView edges={["top", "left", "right"]} style={[styles.screen, { backgroundColor: colors.background }]}>
      <View style={styles.header}>
        <Text variant="heading">Matches</Text>
      </View>
      <View style={styles.screen}>{body}</View>

      <ActionSheet
        actions={
          menuFor
            ? [
                { label: "Open chat", icon: "chatbubble-outline", onPress: () => openChat(menuFor) },
                {
                  label: "View profile",
                  icon: "person-outline",
                  onPress: () => navigation.navigate("ProfileDetail", { userId: menuFor.user.userId, source: "match" })
                },
                {
                  label: "Hide conversation",
                  icon: "eye-off-outline",
                  onPress: () => setPending({ kind: "hide", item: menuFor })
                },
                {
                  label: "Unmatch",
                  icon: "heart-dislike-outline",
                  destructive: true,
                  onPress: () => setPending({ kind: "unmatch", item: menuFor })
                }
              ]
            : []
        }
        onClose={() => setMenuFor(null)}
        title={menuFor?.user.displayName}
        visible={menuFor !== null}
      />
      <ConfirmDialog
        confirmLabel={pending?.kind === "hide" ? "Hide" : "Unmatch"}
        destructive={pending?.kind === "unmatch"}
        loading={hide.isPending || unmatch.isPending}
        message={
          pending?.kind === "hide"
            ? "This clears the chat history for you only. They keep their copy and you stay matched."
            : "You won't be able to message each other anymore. This can't be undone."
        }
        onCancel={() => setPending(null)}
        onConfirm={confirm}
        title={
          pending?.kind === "hide" ? "Hide this conversation?" : `Unmatch ${pending?.item.user.displayName ?? ""}?`
        }
        visible={pending !== null}
      />
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  screen: { flex: 1 },
  header: { paddingHorizontal: spacing.md, paddingVertical: spacing.xs },
  noChats: { padding: spacing.lg }
});
