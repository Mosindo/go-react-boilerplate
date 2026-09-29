import React, { useCallback, useEffect, useReducer, useRef, useState } from "react";
import { StyleSheet, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { DISCOVER_BATCH_SIZE, fetchDiscover } from "../api/discovery";
import { swipe } from "../api/matching";
import type { ConversationSummary, PublicProfile } from "../api/models";
import { getMyProfile } from "../api/profile";
import { errorStatus } from "../components/errors";
import { Banner, IconButton, Skeleton, StateView, Txt } from "../components/kit";
import { MatchModal } from "../components/MatchModal";
import { ProfileModal } from "../components/ProfileModal";
import { SwipeDeck, type DeckHandle } from "../components/SwipeDeck";
import { putConversationFirst } from "../lib/dating/cache";
import { discoverQueueReducer, initialDiscoverQueue, shouldPrefetch } from "../lib/dating/queue";
import { classifySwipeFailure, type SwipeAction } from "../lib/dating/swipe";
import { CONVERSATIONS_KEY, type ConversationsData } from "../realtime/queries";
import { useTheme } from "../shared/ui/theme";

type Props = { onOpenConversation: (conversationId: string) => void };

type LoadState = {
  status: "idle" | "loading" | "error";
  error: string | null;
  exhausted: boolean;
  incomplete: boolean;
};

type Notice = { tone: "danger" | "info"; message: string };

const INITIAL_LOAD: LoadState = {
  status: "idle",
  error: null,
  exhausted: false,
  incomplete: false
};

export default function DiscoverScreen({ onOpenConversation }: Props) {
  const { colors, spacing, radii } = useTheme();
  const queryClient = useQueryClient();

  const [state, dispatch] = useReducer(discoverQueueReducer, initialDiscoverQueue);
  const [load, setLoad] = useState<LoadState>(INITIAL_LOAD);
  const [notice, setNotice] = useState<Notice | null>(null);
  const [match, setMatch] = useState<ConversationSummary | null>(null);
  const [info, setInfo] = useState<PublicProfile | null>(null);
  const [refreshing, setRefreshing] = useState(false);

  const stateRef = useRef(state);
  const inFlight = useRef(false);
  const batch = useRef(0);
  const mounted = useRef(true);
  const deck = useRef<DeckHandle | null>(null);

  useEffect(() => {
    stateRef.current = state;
  }, [state]);

  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);

  const myProfile = useQuery({
    queryKey: ["me", "profile"],
    queryFn: getMyProfile,
    staleTime: 5 * 60_000
  });
  const myPhoto = myProfile.data?.photos[0] ?? null;

  const loadMore = useCallback(async () => {
    if (inFlight.current) return;
    inFlight.current = true;
    setLoad((s) => ({ ...s, status: "loading", error: null }));
    try {
      batch.current += 1;
      const profiles = await queryClient.fetchQuery({
        queryKey: ["discover", batch.current],
        queryFn: ({ signal }) => fetchDiscover(DISCOVER_BATCH_SIZE, signal),
        staleTime: 0,
        gcTime: 0
      });
      if (!mounted.current) return;
      const known = new Set<string>(stateRef.current.seen);
      for (const p of stateRef.current.queue) known.add(p.userId);
      const fresh = profiles.filter((p) => !known.has(p.userId));
      dispatch({ type: "append", profiles });
      setLoad({
        status: "idle",
        error: null,
        exhausted: fresh.length === 0,
        incomplete: false
      });
    } catch (e) {
      if (!mounted.current) return;
      const status = errorStatus(e);
      if (status === 422) {
        setLoad({
          status: "idle",
          error: null,
          exhausted: false,
          incomplete: true
        });
      } else {
        setLoad((s) => ({
          ...s,
          status: "error",
          error: "We couldn't load new people. Check your connection and try again."
        }));
      }
    } finally {
      inFlight.current = false;
    }
  }, [queryClient]);

  // Keep the queue topped up: fetch when the deck is at or under the threshold.
  const remaining = state.queue.length;
  useEffect(() => {
    if (load.status !== "idle" || load.exhausted || load.incomplete) return;
    if (shouldPrefetch(remaining)) void loadMore();
  }, [remaining, load.status, load.exhausted, load.incomplete, loadMore]);

  const refresh = useCallback(async () => {
    setRefreshing(true);
    setNotice(null);
    await loadMore();
    if (mounted.current) setRefreshing(false);
  }, [loadMore]);

  const submitSwipe = useCallback(
    async (profile: PublicProfile, action: SwipeAction) => {
      try {
        const result = await swipe(profile.userId, action);
        if (result.matched) {
          const conversation = result.conversation;
          if (conversation) {
            queryClient.setQueryData<ConversationsData>(CONVERSATIONS_KEY, (d) =>
              d ? putConversationFirst(d, conversation) : d
            );
            if (mounted.current) setMatch(conversation);
          }
          void queryClient.invalidateQueries({ queryKey: CONVERSATIONS_KEY });
          void queryClient.invalidateQueries({ queryKey: ["notifications"] });
        }
      } catch (e) {
        if (!mounted.current) return;
        switch (classifySwipeFailure(errorStatus(e))) {
          case "already-swiped":
            break;
          case "unavailable":
            setNotice({
              tone: "info",
              message: "That profile is no longer available."
            });
            break;
          case "profile-incomplete":
            setLoad((s) => ({ ...s, incomplete: true }));
            break;
          default:
            dispatch({ type: "restore", profile });
            setNotice({
              tone: "danger",
              message: "We couldn't send your choice. The card is back, try again."
            });
        }
      }
    },
    [queryClient]
  );

  const handleSwipe = useCallback(
    (profile: PublicProfile, action: SwipeAction) => {
      setNotice(null);
      dispatch({ type: "act", userId: profile.userId });
      void submitSwipe(profile, action);
    },
    [submitSwipe]
  );

  const handleOpenInfo = useCallback((profile: PublicProfile) => setInfo(profile), []);
  const closeInfo = useCallback(() => setInfo(null), []);
  const handleBlocked = useCallback(
    (userId: string) => {
      dispatch({ type: "drop", userId });
      void queryClient.invalidateQueries({ queryKey: CONVERSATIONS_KEY });
    },
    [queryClient]
  );
  const pressPass = useCallback(() => deck.current?.swipe("pass"), []);
  const pressLike = useCallback(() => deck.current?.swipe("like"), []);
  const keepSwiping = useCallback(() => setMatch(null), []);
  const sayHello = useCallback(
    (conversationId: string) => {
      setMatch(null);
      onOpenConversation(conversationId);
    },
    [onOpenConversation]
  );

  const hasCards = state.queue.length > 0;
  let content: React.ReactNode;
  if (load.incomplete && !hasCards) {
    content = (
      <StateView
        testID="discover-incomplete"
        glyph="✨"
        title="Finish your profile to start discovering"
        message="Add a photo and your location so people nearby can see you too."
      />
    );
  } else if (hasCards) {
    content = (
      <>
        {notice ? (
          <Banner
            tone={notice.tone}
            message={notice.message}
            onDismiss={() => setNotice(null)}
            style={{ marginBottom: spacing.sm }}
          />
        ) : null}
        <View style={styles.stage}>
          <SwipeDeck
            profiles={state.queue.slice(0, 3)}
            onSwipe={handleSwipe}
            onOpenInfo={handleOpenInfo}
            handleRef={deck}
          />
        </View>
        <View style={[styles.actions, { gap: spacing.xxl, paddingTop: spacing.md }]}>
          <IconButton
            glyph="✕"
            label="Pass"
            size={68}
            glyphSize={28}
            color={colors.pass}
            backgroundColor={colors.surface}
            borderColor={colors.borderStrong}
            onPress={pressPass}
            testID="discover-pass-button"
          />
          <IconButton
            glyph="♥"
            label="Like"
            size={76}
            glyphSize={32}
            color={colors.primaryForeground}
            backgroundColor={colors.like}
            borderColor={colors.like}
            onPress={pressLike}
            testID="discover-like-button"
          />
        </View>
      </>
    );
  } else if (load.status === "error") {
    content = (
      <StateView
        testID="discover-error"
        glyph="📡"
        title="Can't reach Alba"
        message={load.error ?? undefined}
        actionLabel="Try again"
        onAction={() => void refresh()}
        refreshing={refreshing}
        onRefresh={() => void refresh()}
      />
    );
  } else if (load.exhausted && load.status === "idle") {
    content = (
      <StateView
        testID="discover-empty"
        glyph="🌙"
        title="You've seen everyone nearby"
        message="Widen your distance or check back soon."
        actionLabel="Refresh"
        onAction={() => void refresh()}
        refreshing={refreshing}
        onRefresh={() => void refresh()}
      />
    );
  } else {
    content = (
      <View style={styles.stage} testID="discover-loading" accessibilityLabel="Loading people">
        <Skeleton style={{ flex: 1, borderRadius: radii.xl }} />
      </View>
    );
  }

  return (
    <SafeAreaView edges={["top"]} style={[styles.flex, { backgroundColor: colors.background }]}>
      <View style={[styles.header, { paddingHorizontal: spacing.lg, paddingVertical: spacing.sm }]}>
        <Txt variant="title" accessibilityRole="header">
          Discover
        </Txt>
      </View>
      <View style={[styles.flex, { paddingHorizontal: spacing.lg, paddingBottom: spacing.md }]}>
        {content}
      </View>
      <MatchModal
        visible={match !== null}
        conversation={match}
        myPhoto={myPhoto}
        onSayHello={sayHello}
        onKeepSwiping={keepSwiping}
      />
      <ProfileModal
        visible={info !== null}
        userId={info?.userId ?? null}
        initialProfile={info}
        onClose={closeInfo}
        onBlocked={handleBlocked}
      />
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  header: { flexDirection: "row", alignItems: "center" },
  stage: { flex: 1 },
  actions: {
    flexDirection: "row",
    justifyContent: "center",
    alignItems: "center"
  }
});
