import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  ActivityIndicator,
  Alert,
  BackHandler,
  FlatList,
  Pressable,
  RefreshControl,
  StyleSheet,
  View
} from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { useQueryClient } from "@tanstack/react-query";
import { hideConversation, listConversations } from "../api/chat";
import type { ConversationSummary } from "../api/models";
import { errorMessage } from "../components/errors";
import { ConversationRow } from "../components/ConversationRow";
import { Banner, MIN_TARGET, Skeleton, StateView, Txt } from "../components/kit";
import { PersonAvatar } from "../components/PersonAvatar";
import {
  findConversation,
  flattenConversations,
  removeConversation,
  selectActiveConversations,
  selectNewMatches
} from "../lib/dating/cache";
import {
  CONVERSATIONS_KEY,
  useConversationsQuery,
  type ConversationsData
} from "../realtime/queries";
import { useAuth } from "../hooks/useAuth";
import { useTheme } from "../shared/ui/theme";
import ChatThreadScreen from "./ChatThreadScreen";

type Props = {
  openConversationId: string | null;
  onOpenedConversation: () => void;
};

const KEEP_SWIPING_HINT = "Keep swiping on Discover to meet someone new.";

function NewMatchesStrip({
  matches,
  onPress,
  onLongPress
}: {
  matches: ConversationSummary[];
  onPress: (c: ConversationSummary) => void;
  onLongPress: (c: ConversationSummary) => void;
}) {
  const { spacing } = useTheme();
  if (matches.length === 0) return null;
  return (
    <View style={{ paddingBottom: spacing.md }} testID="new-matches">
      <Txt
        variant="label"
        tone="muted"
        weight="bold"
        accessibilityRole="header"
        style={{ paddingHorizontal: spacing.lg, paddingBottom: spacing.sm }}
      >
        New matches
      </Txt>
      <FlatList
        horizontal
        data={matches}
        keyExtractor={(c) => c.id}
        showsHorizontalScrollIndicator={false}
        contentContainerStyle={{
          paddingHorizontal: spacing.lg,
          gap: spacing.md
        }}
        renderItem={({ item }) => (
          <Pressable
            testID={`new-match-${item.id}`}
            accessibilityRole="button"
            accessibilityLabel={`New match: ${item.user.firstName}. Say hello`}
            onPress={() => onPress(item)}
            onLongPress={() => onLongPress(item)}
            style={[styles.matchItem, { minHeight: MIN_TARGET }]}
          >
            <PersonAvatar name={item.user.firstName} photo={item.user.photo} size={68} />
            <Txt variant="label" weight="semibold" numberOfLines={1} style={styles.matchName}>
              {item.user.firstName}
            </Txt>
          </Pressable>
        )}
      />
    </View>
  );
}

export default function ConversationsScreen({ openConversationId, onOpenedConversation }: Props) {
  const { colors, spacing } = useTheme();
  const { user } = useAuth();
  const myUserId = user?.id ?? "";
  const queryClient = useQueryClient();
  const query = useConversationsQuery();

  const [openId, setOpenId] = useState<string | null>(null);
  const [snapshot, setSnapshot] = useState<ConversationSummary | null>(null);
  const [flash, setFlash] = useState<string | null>(null);
  const [refreshing, setRefreshing] = useState(false);

  const openedRef = useRef(onOpenedConversation);
  useEffect(() => {
    openedRef.current = onOpenedConversation;
  }, [onOpenedConversation]);

  const pages = query.data?.pages;
  const all = useMemo(() => (pages ? flattenConversations(pages) : []), [pages]);
  const newMatches = useMemo(() => selectNewMatches(all), [all]);
  const active = useMemo(() => selectActiveConversations(all), [all]);

  const openThread = useCallback((conversation: ConversationSummary) => {
    setFlash(null);
    setSnapshot(conversation);
    setOpenId(conversation.id);
  }, []);

  // External request to open a thread (match modal, notification tap).
  useEffect(() => {
    if (!openConversationId) return;
    let cancelled = false;
    void (async () => {
      const cachedPages = queryClient.getQueryData<ConversationsData>(CONVERSATIONS_KEY)?.pages;
      let found = cachedPages ? findConversation(cachedPages, openConversationId) : null;
      if (!found) {
        try {
          const res = await listConversations();
          found = res.conversations.find((c) => c.id === openConversationId) ?? null;
          void queryClient.invalidateQueries({ queryKey: CONVERSATIONS_KEY });
        } catch {
          found = null;
        }
      }
      if (cancelled) return;
      if (found) openThread(found);
      else setFlash("That conversation is no longer available.");
      openedRef.current();
    })();
    return () => {
      cancelled = true;
    };
  }, [openConversationId, openThread, queryClient]);

  const closeThread = useCallback(() => {
    setOpenId(null);
    setSnapshot(null);
  }, []);

  const onThreadGone = useCallback((message?: string) => {
    setOpenId(null);
    setSnapshot(null);
    if (message) setFlash(message);
  }, []);

  useEffect(() => {
    if (!openId) return undefined;
    const sub = BackHandler.addEventListener("hardwareBackPress", () => {
      closeThread();
      return true;
    });
    return () => sub.remove();
  }, [openId, closeThread]);

  const confirmHide = useCallback(
    (conversation: ConversationSummary) => {
      Alert.alert(
        `Delete chat with ${conversation.user.firstName}?`,
        "It disappears from your list only. You stay matched, and a new message brings it back.",
        [
          { text: "Cancel", style: "cancel" },
          {
            text: "Delete",
            style: "destructive",
            onPress: () => {
              void (async () => {
                try {
                  await hideConversation(conversation.id);
                  queryClient.setQueryData<ConversationsData>(CONVERSATIONS_KEY, (d) =>
                    d ? removeConversation(d, conversation.id) : d
                  );
                } catch (e) {
                  setFlash(errorMessage(e, "We couldn't delete this chat. Please try again."));
                }
              })();
            }
          }
        ]
      );
    },
    [queryClient]
  );

  const onRefresh = useCallback(async () => {
    setRefreshing(true);
    try {
      await query.refetch();
    } finally {
      setRefreshing(false);
    }
  }, [query]);

  const onEndReached = useCallback(() => {
    if (query.hasNextPage && !query.isFetchingNextPage) void query.fetchNextPage();
  }, [query]);

  if (openId) {
    const current = (pages ? findConversation(pages, openId) : null) ?? snapshot;
    if (current) {
      return (
        <ChatThreadScreen
          key={current.id}
          conversation={current}
          onBack={closeThread}
          onGone={onThreadGone}
        />
      );
    }
  }

  let body: React.ReactNode;
  if (query.isPending) {
    body = (
      <View style={{ padding: spacing.lg, gap: spacing.md }} testID="conversations-loading">
        {[0, 1, 2, 3, 4].map((i) => (
          <View key={i} style={[styles.skeletonRow, { gap: spacing.md }]}>
            <Skeleton style={{ width: 56, height: 56, borderRadius: 28 }} />
            <View style={styles.flex}>
              <Skeleton style={{ height: 16, width: "40%", marginBottom: 8 }} />
              <Skeleton style={{ height: 14, width: "75%" }} />
            </View>
          </View>
        ))}
      </View>
    );
  } else if (query.isError && all.length === 0) {
    body = (
      <StateView
        testID="conversations-error"
        title="Couldn't load your matches"
        message="Check your connection and try again."
        actionLabel="Try again"
        onAction={() => void query.refetch()}
        refreshing={refreshing}
        onRefresh={() => void onRefresh()}
      />
    );
  } else if (all.length === 0) {
    body = (
      <StateView
        testID="conversations-empty"
        glyph="💌"
        title="No matches yet"
        message={`When you and someone else like each other, they show up here. ${KEEP_SWIPING_HINT}`}
        refreshing={refreshing}
        onRefresh={() => void onRefresh()}
      />
    );
  } else {
    body = (
      <FlatList
        testID="conversations-list"
        data={active}
        keyExtractor={(c) => c.id}
        renderItem={({ item }) => (
          <ConversationRow
            conversation={item}
            myUserId={myUserId}
            onPress={openThread}
            onLongPress={confirmHide}
          />
        )}
        ListHeaderComponent={
          <NewMatchesStrip matches={newMatches} onPress={openThread} onLongPress={confirmHide} />
        }
        ListEmptyComponent={
          newMatches.length > 0 ? (
            <Txt tone="muted" style={[styles.hint, { padding: spacing.xl }]}>
              Tap a new match to say hello.
            </Txt>
          ) : null
        }
        ListFooterComponent={
          query.isFetchingNextPage ? (
            <ActivityIndicator color={colors.primary} style={{ padding: spacing.lg }} />
          ) : null
        }
        contentContainerStyle={{ paddingBottom: spacing.xl }}
        onEndReached={onEndReached}
        onEndReachedThreshold={0.5}
        refreshControl={
          <RefreshControl
            refreshing={refreshing}
            onRefresh={() => void onRefresh()}
            tintColor={colors.primary}
            colors={[colors.primary]}
          />
        }
      />
    );
  }

  return (
    <SafeAreaView edges={["top"]} style={[styles.flex, { backgroundColor: colors.background }]}>
      <View style={{ paddingHorizontal: spacing.lg, paddingVertical: spacing.sm }}>
        <Txt variant="title" accessibilityRole="header">
          Matches
        </Txt>
      </View>
      {flash ? (
        <Banner
          tone="info"
          message={flash}
          onDismiss={() => setFlash(null)}
          testID="conversations-flash"
          style={{ marginHorizontal: spacing.lg, marginBottom: spacing.sm }}
        />
      ) : null}
      <View style={styles.flex}>{body}</View>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  matchItem: { alignItems: "center", width: 76, gap: 6 },
  matchName: { textAlign: "center", maxWidth: 76 },
  skeletonRow: { flexDirection: "row", alignItems: "center" },
  hint: { textAlign: "center" }
});
