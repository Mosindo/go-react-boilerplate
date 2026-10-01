import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Animated, PanResponder, Platform, StyleSheet, View, useWindowDimensions } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ApiError } from "../api/client";
import { discoverApi } from "../api/endpoints";
import { keys } from "../api/keys";
import type { Card as ProfileCard, SwipeAction } from "../api/types";
import { useFeedback } from "../components/Feedback";
import { errorMessage } from "../components/forms";
import { CardOverlay, MatchModal, PhotoCarousel } from "../components/profile";
import { EmptyState, ErrorState, IconButton, Loading, Screen, Text } from "../components/ui";
import type { RootStackParamList } from "../navigation/types";
import { radii, spacing, useTheme } from "../theme/theme";

type Nav = NativeStackNavigationProp<RootStackParamList>;

const BATCH = 10;
const REFILL_AT = 3;

type CardHandle = { fling: (action: SwipeAction) => void };

function SwipeCard({
  card,
  active,
  onSwipe,
  onInfo,
  handleRef
}: {
  card: ProfileCard;
  active: boolean;
  onSwipe: (action: SwipeAction) => void;
  onInfo: () => void;
  handleRef?: React.MutableRefObject<CardHandle | null>;
}) {
  const t = useTheme();
  const { width } = useWindowDimensions();
  const position = useRef(new Animated.ValueXY()).current;
  const threshold = Math.min(width * 0.3, 140);
  const onSwipeRef = useRef(onSwipe);
  onSwipeRef.current = onSwipe;

  const fling = useCallback(
    (action: SwipeAction) => {
      Animated.timing(position, {
        toValue: { x: action === "like" ? width * 1.4 : -width * 1.4, y: 0 },
        duration: 220,
        useNativeDriver: false
      }).start(() => onSwipeRef.current(action));
    },
    [position, width]
  );

  useEffect(() => {
    if (active && handleRef) handleRef.current = { fling };
  }, [active, fling, handleRef]);

  const pan = useMemo(
    () =>
      PanResponder.create({
        onMoveShouldSetPanResponder: (_, g) => Math.abs(g.dx) > 8 && Math.abs(g.dx) > Math.abs(g.dy),
        onPanResponderMove: Animated.event([null, { dx: position.x, dy: position.y }], { useNativeDriver: false }),
        onPanResponderRelease: (_, g) => {
          if (g.dx > threshold || g.vx > 0.8) fling("like");
          else if (g.dx < -threshold || g.vx < -0.8) fling("pass");
          else Animated.spring(position, { toValue: { x: 0, y: 0 }, useNativeDriver: false, friction: 6 }).start();
        }
      }),
    [fling, position, threshold]
  );

  const rotate = position.x.interpolate({ inputRange: [-width, 0, width], outputRange: ["-12deg", "0deg", "12deg"] });
  const likeOpacity = position.x.interpolate({ inputRange: [20, threshold], outputRange: [0, 1], extrapolate: "clamp" });
  const passOpacity = position.x.interpolate({ inputRange: [-threshold, -20], outputRange: [1, 0], extrapolate: "clamp" });

  return (
    <Animated.View
      {...(active ? pan.panHandlers : {})}
      pointerEvents={active ? "auto" : "none"}
      style={[
        styles.card,
        { backgroundColor: t.surface, borderColor: t.border },
        active ? { transform: [{ translateX: position.x }, { translateY: position.y }, { rotate }] } : { transform: [{ scale: 0.96 }] }
      ]}
      testID={active ? "discover-card" : undefined}
    >
      <PhotoCarousel card={card} fill onTap={onInfo} />
      <CardOverlay card={card} />
      <Animated.View style={[styles.stamp, styles.likeStamp, { opacity: likeOpacity, borderColor: t.like }]} pointerEvents="none">
        <Text variant="heading" color={t.like}>J&apos;AIME</Text>
      </Animated.View>
      <Animated.View style={[styles.stamp, styles.passStamp, { opacity: passOpacity, borderColor: t.pass }]} pointerEvents="none">
        <Text variant="heading" color={t.pass}>PASSE</Text>
      </Animated.View>
      <IconButton icon="information-circle" label={`Voir le profil de ${card.firstName}`} color="#FFFFFF" onPress={onInfo} testID="discover-info" style={styles.info} />
    </Animated.View>
  );
}

export function DiscoverScreen() {
  const t = useTheme();
  const nav = useNavigation<Nav>();
  const { toast } = useFeedback();
  const qc = useQueryClient();
  const [deck, setDeck] = useState<ProfileCard[]>([]);
  const [loading, setLoading] = useState(false);
  const [exhausted, setExhausted] = useState(false);
  const [loadError, setLoadError] = useState<ApiError | Error | null>(null);
  const [match, setMatch] = useState<{ card: ProfileCard; conversationId: string } | null>(null);
  const seen = useRef(new Set<string>());
  const topRef = useRef<CardHandle | null>(null);
  const busy = useRef(false);

  // Preferences/location screens invalidate this key; a new value resets the deck.
  const epoch = useQuery({ queryKey: ["discover", "epoch"], queryFn: async () => Date.now(), staleTime: Infinity });

  const loadMore = useCallback(async () => {
    if (busy.current) return;
    busy.current = true;
    setLoading(true);
    try {
      const batch = await discoverApi.feed(BATCH);
      const fresh = batch.filter((c) => !seen.current.has(c.userId));
      fresh.forEach((c) => seen.current.add(c.userId));
      setLoadError(null);
      if (fresh.length === 0) setExhausted(true);
      setDeck((d) => [...d, ...fresh]);
    } catch (e) {
      setLoadError(e as Error);
    } finally {
      busy.current = false;
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    seen.current = new Set();
    setDeck([]);
    setExhausted(false);
    setLoadError(null);
    void loadMore();
  }, [epoch.data, loadMore]);

  useEffect(() => {
    if (deck.length < REFILL_AT && !exhausted && !loadError && !busy.current && epoch.isSuccess) void loadMore();
  }, [deck.length, exhausted, loadError, loadMore, epoch.isSuccess]);

  const swipe = useMutation({
    mutationFn: ({ card, action }: { card: ProfileCard; action: SwipeAction }) => discoverApi.swipe(card.userId, action),
    onSuccess: (res, { card }) => {
      if (res.matched && res.conversationId) {
        setMatch({ card: res.match ?? card, conversationId: res.conversationId });
        void qc.invalidateQueries({ queryKey: keys.matches });
        void qc.invalidateQueries({ queryKey: keys.conversations });
        void qc.invalidateQueries({ queryKey: keys.notifications });
        void qc.invalidateQueries({ queryKey: keys.unread });
      }
    },
    onError: (e, { card }) => {
      if (e instanceof ApiError && (e.status === 409 || e.status === 404)) return; // already handled / no longer eligible
      setDeck((d) => [card, ...d]); // network or server failure: give the card back
      toast(errorMessage(e), "error");
    }
  });

  const onSwiped = (action: SwipeAction) => {
    const top = deck[0];
    if (!top) return;
    setDeck((d) => d.slice(1));
    swipe.mutate({ card: top, action });
  };

  const top = deck[0];
  const next = deck[1];
  const openProfile = (c: ProfileCard) => nav.navigate("ProfileDetail", { userId: c.userId });

  if (loadError instanceof ApiError && loadError.code === "profile_incomplete") {
    return (
      <Screen>
        <EmptyState icon="images-outline" title="Complétez votre profil" message="Ajoutez au moins une photo pour découvrir des profils." actionLabel="Gérer mes photos" onAction={() => nav.navigate("Photos")} />
      </Screen>
    );
  }

  return (
    <Screen>
      <View style={styles.header}>
        <Text variant="title">Découvrir</Text>
      </View>
      <View style={styles.stack}>
        {top ? (
          <>
            {next ? <SwipeCard key={next.userId} card={next} active={false} onSwipe={() => undefined} onInfo={() => undefined} /> : null}
            <SwipeCard key={top.userId} card={top} active onSwipe={onSwiped} onInfo={() => openProfile(top)} handleRef={topRef} />
          </>
        ) : loadError ? (
          <ErrorState message={loadError.message} onRetry={() => void loadMore()} />
        ) : loading || !epoch.isSuccess ? (
          <Loading label="Recherche de profils…" />
        ) : (
          <EmptyState
            icon="compass-outline"
            title="Vous avez tout vu pour le moment"
            message="Revenez plus tard ou élargissez vos préférences (âge, distance) pour voir plus de monde."
            actionLabel="Actualiser"
            onAction={() => {
              setExhausted(false);
              void loadMore();
            }}
          />
        )}
      </View>
      {top ? (
        <View style={styles.actions}>
          <IconButton
            icon="close"
            label={`Passer ${top.firstName}`}
            color={t.pass}
            size={32}
            testID="discover-pass"
            onPress={() => topRef.current?.fling("pass")}
            style={[styles.actionBtn, { backgroundColor: t.surface, borderColor: t.border }]}
          />
          <IconButton
            icon="heart"
            label={`Aimer ${top.firstName}`}
            color={t.onPrimary}
            size={32}
            testID="discover-like"
            onPress={() => topRef.current?.fling("like")}
            style={[styles.actionBtn, styles.likeBtn, { backgroundColor: t.primary, borderColor: t.primary }]}
          />
        </View>
      ) : null}
      <MatchModal
        match={match?.card ?? null}
        onClose={() => setMatch(null)}
        onMessage={() => {
          const m = match;
          setMatch(null);
          if (m) nav.navigate("Chat", { conversationId: m.conversationId, userId: m.card.userId, name: m.card.firstName });
        }}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  header: { paddingTop: spacing.sm, paddingBottom: spacing.md },
  stack: { flex: 1, maxWidth: 520, width: "100%", alignSelf: "center" },
  card: {
    ...StyleSheet.absoluteFillObject,
    borderRadius: radii.xl,
    borderWidth: 1,
    overflow: "hidden",
    ...Platform.select({ web: { boxShadow: "0 8px 24px rgba(40,20,40,0.15)" } as object, default: { elevation: 4 } })
  },
  stamp: { position: "absolute", top: 40, borderWidth: 3, borderRadius: radii.md, paddingHorizontal: spacing.md, paddingVertical: 4 },
  likeStamp: { left: 24, transform: [{ rotate: "-14deg" }] },
  passStamp: { right: 24, transform: [{ rotate: "14deg" }] },
  info: { position: "absolute", right: spacing.sm, bottom: spacing.md },
  actions: { flexDirection: "row", justifyContent: "center", gap: spacing.xxl, paddingVertical: spacing.lg },
  actionBtn: { width: 68, height: 68, borderRadius: 34, borderWidth: 1 },
  likeBtn: { width: 76, height: 76, borderRadius: 38 }
});
