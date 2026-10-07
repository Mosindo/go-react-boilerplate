import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  Animated,
  Image,
  PanResponder,
  Pressable,
  StyleSheet,
  View,
  useWindowDimensions
} from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { useFocusEffect } from "@react-navigation/native";
import { useQueryClient } from "@tanstack/react-query";
import { ApiError } from "../api/client";
import { fetchDiscover, postSwipe } from "../api/discover";
import { EmptyView, ErrorView } from "../shared/feedback";
import { Notice, Text, radii, spacing } from "../shared/ui";
import { MatchOverlay } from "../features/discover/MatchOverlay";
import { ProfileCard } from "../features/discover/ProfileCard";
import { onSwipeOutcome } from "../features/discover/swipeBus";
import {
  appendBatch,
  decideSwipe,
  exitTarget,
  nextPhotoIndex,
  rotationFor,
  shouldPrefetch,
  stampOpacity,
  swipeThreshold
} from "../features/discover/swipe";
import { errorMessage, isApiErrorCode } from "../features/errors";
import { openChat, openUserProfile, useAppNavigation } from "../features/nav";
import { queryKeys } from "../features/queryKeys";
import type { Match, PublicProfile, SwipeAction } from "../features/types";
import { photoUri } from "../features/ui/PhotoImage";
import { ScreenHeader } from "../features/ui/ScreenHeader";
import { Skeleton } from "../features/ui/Skeleton";
import { usePalette } from "../features/ui/palette";

const BATCH_SIZE = 10;

export default function DiscoverScreen() {
  const p = usePalette();
  const navigation = useAppNavigation();
  const queryClient = useQueryClient();
  const { width } = useWindowDimensions();

  const [deck, setDeck] = useState<PublicProfile[]>([]);
  const [hasMore, setHasMore] = useState(true);
  const [fetching, setFetching] = useState(false);
  const [initialLoad, setInitialLoad] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [photoIndex, setPhotoIndex] = useState(0);
  const [match, setMatch] = useState<Match | null>(null);
  const [toast, setToast] = useState<string | null>(null);

  const deckRef = useRef<PublicProfile[]>([]);
  const seenRef = useRef<Set<string>>(new Set());
  const fetchingRef = useRef(false);
  const animatingRef = useRef(false);

  const pan = useRef(new Animated.ValueXY()).current;
  const rotation = useRef(new Animated.Value(0)).current;
  const likeOpacity = useRef(new Animated.Value(0)).current;
  const nopeOpacity = useRef(new Animated.Value(0)).current;

  const setDeckBoth = useCallback((next: PublicProfile[]) => {
    deckRef.current = next;
    setDeck(next);
  }, []);

  const loadBatch = useCallback(async () => {
    if (fetchingRef.current) {
      return;
    }
    fetchingRef.current = true;
    setFetching(true);
    setError(null);
    try {
      const items = await fetchDiscover(BATCH_SIZE);
      const before = deckRef.current;
      const next = appendBatch(before, items, seenRef.current);
      setDeckBoth(next);
      setHasMore(next.length > before.length);
    } catch (err) {
      setError(
        isApiErrorCode(err, "profile_incomplete")
          ? "Add at least one photo to your profile to start discovering people."
          : errorMessage(err, "Couldn't load people nearby.")
      );
    } finally {
      fetchingRef.current = false;
      setFetching(false);
      setInitialLoad(false);
    }
  }, [setDeckBoth]);

  useEffect(() => {
    void loadBatch();
  }, [loadBatch]);

  useFocusEffect(
    useCallback(() => {
      if (!initialLoad && deckRef.current.length === 0 && !fetchingRef.current) {
        setHasMore(true);
        void loadBatch();
      }
    }, [initialLoad, loadBatch])
  );

  useEffect(() => {
    if (!error && !initialLoad && shouldPrefetch(deck.length, hasMore, fetching)) {
      void loadBatch();
    }
  }, [deck.length, hasMore, fetching, error, initialLoad, loadBatch]);

  // Warm the image cache for the next card and the remaining photos of the current one.
  useEffect(() => {
    const urls: string[] = [];
    const next = deck[1]?.photos[0];
    const nextUri = photoUri(next);
    if (nextUri) {
      urls.push(nextUri);
    }
    for (const photo of deck[0]?.photos.slice(1) ?? []) {
      const uri = photoUri(photo);
      if (uri) {
        urls.push(uri);
      }
    }
    for (const uri of urls) {
      Image.prefetch(uri).catch(() => undefined);
    }
  }, [deck]);

  useEffect(() => {
    const id = pan.x.addListener(({ value }) => {
      rotation.setValue(rotationFor(value, width));
      const stamps = stampOpacity(value, width);
      likeOpacity.setValue(stamps.like);
      nopeOpacity.setValue(stamps.pass);
    });
    return () => pan.x.removeListener(id);
  }, [pan, rotation, likeOpacity, nopeOpacity, width]);

  useEffect(() => {
    if (!toast) {
      return undefined;
    }
    const id = setTimeout(() => setToast(null), 3500);
    return () => clearTimeout(id);
  }, [toast]);

  const submitSwipe = useCallback(
    (target: PublicProfile, action: SwipeAction) => {
      postSwipe(target.userId, action)
        .then((result) => {
          if (result.matched && result.match) {
            setMatch(result.match);
            void queryClient.invalidateQueries({ queryKey: queryKeys.matches });
            void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
          }
        })
        .catch((err: unknown) => {
          if (isApiErrorCode(err, "already_swiped")) {
            return;
          }
          const restorable = !(err instanceof ApiError) || err.status >= 500;
          if (restorable) {
            seenRef.current.delete(target.userId);
            setDeckBoth([target, ...deckRef.current]);
          }
          setToast(
            restorable
              ? "Couldn't save that choice. Please try again."
              : errorMessage(err, "Couldn't save that choice.")
          );
        });
    },
    [queryClient, setDeckBoth]
  );

  const commit = useCallback(
    (action: SwipeAction) => {
      const top = deckRef.current[0];
      if (!top || animatingRef.current) {
        return;
      }
      animatingRef.current = true;
      Animated.timing(pan, {
        toValue: { x: exitTarget(action, width), y: 0 },
        duration: 220,
        useNativeDriver: false
      }).start(() => {
        seenRef.current.add(top.userId);
        setDeckBoth(deckRef.current.filter((c) => c.userId !== top.userId));
        setPhotoIndex(0);
        pan.setValue({ x: 0, y: 0 });
        animatingRef.current = false;
        submitSwipe(top, action);
      });
    },
    [pan, width, setDeckBoth, submitSwipe]
  );

  const commitRef = useRef(commit);
  commitRef.current = commit;

  const springBack = useCallback(() => {
    Animated.spring(pan, { toValue: { x: 0, y: 0 }, friction: 6, useNativeDriver: false }).start();
  }, [pan]);

  const responder = useMemo(
    () =>
      PanResponder.create({
        onStartShouldSetPanResponder: () => false,
        onMoveShouldSetPanResponder: (_e, g) =>
          !animatingRef.current && Math.abs(g.dx) > 8 && Math.abs(g.dx) > Math.abs(g.dy),
        onPanResponderMove: Animated.event([null, { dx: pan.x, dy: pan.y }], { useNativeDriver: false }),
        onPanResponderRelease: (_e, g) => {
          const decision = decideSwipe({ dx: g.dx, dy: g.dy, vx: g.vx }, width);
          if (decision === "none") {
            springBack();
          } else {
            commitRef.current(decision);
          }
        },
        onPanResponderTerminate: springBack
      }),
    [pan, width, springBack]
  );

  // Outcomes from the full-profile screen (like/pass there).
  useEffect(
    () =>
      onSwipeOutcome((outcome) => {
        seenRef.current.add(outcome.userId);
        const wasTop = deckRef.current[0]?.userId === outcome.userId;
        setDeckBoth(deckRef.current.filter((c) => c.userId !== outcome.userId));
        if (wasTop) {
          setPhotoIndex(0);
        }
        if (outcome.matched && outcome.match) {
          setMatch(outcome.match);
        }
      }),
    [setDeckBoth]
  );

  const top = deck[0];
  const next = deck[1];

  const refresh = useCallback(() => {
    setHasMore(true);
    void loadBatch();
  }, [loadBatch]);

  const openInfo = useCallback(() => {
    if (top) {
      openUserProfile(navigation, { userId: top.userId });
    }
  }, [navigation, top]);

  const stepPhoto = useCallback(
    (step: -1 | 1) => {
      if (top) {
        setPhotoIndex((current) => nextPhotoIndex(current, top.photos.length, step));
      }
    },
    [top]
  );

  const threshold = swipeThreshold(width);
  const nextScale = pan.x.interpolate({
    inputRange: [-threshold, 0, threshold],
    outputRange: [1, 0.94, 1],
    extrapolate: "clamp"
  });
  const rotate = rotation.interpolate({ inputRange: [-30, 30], outputRange: ["-30deg", "30deg"] });

  let body: React.ReactNode;
  if (initialLoad) {
    body = (
      <View style={styles.deckArea} testID="discover-loading">
        <Skeleton style={{ flex: 1, borderRadius: radii.xl }} />
      </View>
    );
  } else if (error && !top) {
    body = (
      <View style={styles.centered}>
        <ErrorView
          actionLabel="Try again"
          message={error}
          onAction={refresh}
          testID="discover-error"
          title="Couldn't load people"
        />
      </View>
    );
  } else if (!top) {
    body = fetching ? (
      <View style={styles.deckArea} testID="discover-loading">
        <Skeleton style={{ flex: 1, borderRadius: radii.xl }} />
      </View>
    ) : (
      <View style={styles.centered}>
        <EmptyView
          actionLabel="Refresh"
          message="Check back in a little while, or widen your preferences (age range and distance) to see more people."
          onAction={refresh}
          testID="discover-empty"
          title="You're all caught up"
        />
      </View>
    );
  } else {
    body = (
      <>
        <View style={styles.deckArea}>
          {next ? (
            <Animated.View
              pointerEvents="none"
              style={[styles.cardLayer, { transform: [{ scale: nextScale }] }]}
            >
              <ProfileCard
                onInfo={() => undefined}
                onPhotoStep={() => undefined}
                photoIndex={0}
                profile={next}
              />
            </Animated.View>
          ) : null}
          <Animated.View
            {...responder.panHandlers}
            style={[
              styles.cardLayer,
              { transform: [{ translateX: pan.x }, { translateY: pan.y }, { rotate }] }
            ]}
            testID="discover-top-card"
          >
            <ProfileCard
              key={top.userId}
              onInfo={openInfo}
              onPhotoStep={stepPhoto}
              photoIndex={photoIndex}
              profile={top}
            >
              <Animated.View
                pointerEvents="none"
                style={[styles.stamp, styles.stampLike, { opacity: likeOpacity, borderColor: p.like }]}
              >
                <Text style={{ color: p.like }} variant="heading" weight="bold">
                  LIKE
                </Text>
              </Animated.View>
              <Animated.View
                pointerEvents="none"
                style={[styles.stamp, styles.stampNope, { opacity: nopeOpacity, borderColor: p.nope }]}
              >
                <Text style={{ color: p.nope }} variant="heading" weight="bold">
                  NOPE
                </Text>
              </Animated.View>
            </ProfileCard>
          </Animated.View>
        </View>
        <View style={styles.actions}>
          <Pressable
            accessibilityHint="Skips this person"
            accessibilityLabel={`Pass on ${top.firstName}`}
            accessibilityRole="button"
            onPress={() => commit("pass")}
            style={[styles.roundButton, { backgroundColor: p.surface, borderColor: p.nope }]}
            testID="discover-pass-button"
          >
            <Text style={{ color: p.nope, fontSize: 28, lineHeight: 32 }} weight="bold">
              {"✕"}
            </Text>
          </Pressable>
          <Pressable
            accessibilityHint="Lets them know you are interested"
            accessibilityLabel={`Like ${top.firstName}`}
            accessibilityRole="button"
            onPress={() => commit("like")}
            style={[styles.roundButton, styles.likeButton, { backgroundColor: p.like, borderColor: p.like }]}
            testID="discover-like-button"
          >
            <Text style={{ color: "#ffffff", fontSize: 30, lineHeight: 34 }} weight="bold">
              {"♥"}
            </Text>
          </Pressable>
        </View>
      </>
    );
  }

  return (
    <SafeAreaView
      edges={["top"]}
      style={[styles.root, { backgroundColor: p.background }]}
      testID="discover-screen"
    >
      <ScreenHeader title="Discover" />
      {toast ? <Notice style={styles.toast} title={toast} tone="warning" /> : null}
      {body}
      <MatchOverlay
        match={match}
        onDismiss={() => setMatch(null)}
        onMessage={(m) => {
          setMatch(null);
          openChat(navigation, m.conversationId, m.user);
        }}
      />
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1 },
  deckArea: { flex: 1, marginHorizontal: spacing.lg, marginTop: spacing.md },
  cardLayer: { ...StyleSheet.absoluteFillObject },
  centered: { flex: 1, justifyContent: "center", padding: spacing.lg },
  actions: {
    flexDirection: "row",
    justifyContent: "center",
    alignItems: "center",
    gap: spacing.xl,
    paddingVertical: spacing.lg
  },
  roundButton: {
    width: 64,
    height: 64,
    borderRadius: 32,
    borderWidth: 2,
    alignItems: "center",
    justifyContent: "center"
  },
  likeButton: { width: 72, height: 72, borderRadius: 36 },
  stamp: {
    position: "absolute",
    top: 48,
    borderWidth: 4,
    borderRadius: radii.sm,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.xxs
  },
  stampLike: { left: spacing.lg, transform: [{ rotate: "-14deg" }] },
  stampNope: { right: spacing.lg, transform: [{ rotate: "14deg" }] },
  toast: { marginHorizontal: spacing.lg, marginTop: spacing.sm }
});
