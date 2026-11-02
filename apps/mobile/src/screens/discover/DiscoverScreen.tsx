import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Animated, PanResponder, Pressable, StyleSheet, Text, View, useWindowDimensions } from "react-native";
import { useFocusEffect, useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useQueryClient } from "@tanstack/react-query";
import { apiErrorMessage } from "../../api/client";
import { discoverApi } from "../../api/endpoints";
import type { Card } from "../../api/types";
import { CardPhoto, SharedBadge } from "../../components/ProfileCardView";
import { useToast } from "../../components/Toast";
import { AppText, Button, EmptyState, ErrorState, Loading, PhotoImage, Screen } from "../../components/ui";
import type { MainStackParamList } from "../../navigation/types";
import { fontSizes, radii, spacing, useTheme } from "../../theme/theme";

type Nav = NativeStackNavigationProp<MainStackParamList>;
type Decision = "like" | "pass";

const PREFETCH_BELOW = 3;

export default function DiscoverScreen() {
  const { colors } = useTheme();
  const toast = useToast();
  const navigation = useNavigation<Nav>();
  const queryClient = useQueryClient();
  const { width } = useWindowDimensions();

  const [queue, setQueue] = useState<Card[]>([]);
  const [status, setStatus] = useState<"loading" | "ready" | "error">("loading");
  const [error, setError] = useState("");
  const [match, setMatch] = useState<{ card: Card; matchId: string } | null>(null);
  const seen = useRef(new Set<string>()); // ids already shown or decided in this session
  const fetching = useRef(false);
  const exhausted = useRef(false);

  const load = useCallback(
    async (initial: boolean) => {
      if (fetching.current) return;
      fetching.current = true;
      try {
        const { items } = await discoverApi.deck(10);
        const fresh = items.filter((c) => !seen.current.has(c.userId));
        fresh.forEach((c) => seen.current.add(c.userId));
        exhausted.current = fresh.length === 0;
        setQueue((q) => [...q, ...fresh]);
        setStatus("ready");
      } catch (e) {
        if (initial) {
          setError(apiErrorMessage(e));
          setStatus("error");
        } else {
          toast.show(apiErrorMessage(e), "error");
        }
      } finally {
        fetching.current = false;
      }
    },
    [toast]
  );

  useEffect(() => {
    void load(true);
  }, [load]);

  useEffect(() => {
    if (status === "ready" && queue.length <= PREFETCH_BELOW && !exhausted.current) void load(false);
  }, [queue.length, status, load]);

  const decide = useCallback(
    async (card: Card, decision: Decision) => {
      setQueue((q) => q.filter((c) => c.userId !== card.userId)); // optimistic
      try {
        const result = decision === "like" ? await discoverApi.like(card.userId) : await discoverApi.pass(card.userId);
        if (result.matched && result.matchId) {
          setMatch({ card, matchId: result.matchId });
          void queryClient.invalidateQueries({ queryKey: ["conversations"] });
        }
      } catch (e) {
        // Put the card back so the decision is not silently lost.
        setQueue((q) => [card, ...q]);
        toast.show(apiErrorMessage(e), "error");
      }
    },
    [queryClient, toast]
  );

  // `seen` is never cleared: the server already excludes decided profiles, and
  // clearing it while a like/pass is still in flight would let that profile reappear.
  const refresh = () => {
    exhausted.current = false;
    if (queue.length === 0) setStatus("loading");
    void load(true);
  };

  // New people may have joined while the user was elsewhere: retry when the tab regains
  // focus on an empty deck. The callback reads state through a ref and has no
  // dependencies, otherwise every status change would re-trigger it in a loop.
  const latest = useRef({ status, empty: queue.length === 0, refresh });
  latest.current = { status, empty: queue.length === 0, refresh };
  useFocusEffect(
    useCallback(() => {
      const { status: s, empty, refresh: reload } = latest.current;
      if (s === "ready" && empty) reload();
    }, [])
  );

  const top = queue[0];
  const next = queue[1];

  return (
    <Screen edges={["top"]} padded={false}>
      <View style={styles.header}>
        <AppText variant="heading">Découvrir</AppText>
      </View>
      <View style={styles.deck}>
        {status === "loading" ? <Loading label="Recherche de profils…" /> : null}
        {status === "error" ? <ErrorState message={error} onRetry={refresh} /> : null}
        {status === "ready" && !top ? (
          <EmptyState
            emoji="🌿"
            title="Vous avez tout vu pour l'instant"
            message="Revenez plus tard, ou élargissez vos critères (distance, âge) dans votre profil."
            action={<Button label="Actualiser" variant="secondary" onPress={refresh} />}
          />
        ) : null}
        {top ? (
          <>
            {next ? (
              <View style={[styles.cardSlot, { transform: [{ scale: 0.95 }] }]} pointerEvents="none">
                <PhotoImage url={next.photos[0]?.url} style={StyleSheet.absoluteFill} />
              </View>
            ) : null}
            <SwipeCard
              key={top.userId}
              card={top}
              width={width}
              onDecide={(d) => void decide(top, d)}
              onInfo={() => navigation.navigate("UserProfile", { userId: top.userId })}
            />
          </>
        ) : null}
      </View>
      {top ? (
        <View style={styles.actions}>
          <RoundButton label="Passer" glyph="✕" color={colors.textMuted} onPress={() => void decide(top, "pass")} testID="pass" />
          <RoundButton label="J'aime" glyph="♥" color={colors.primary} big onPress={() => void decide(top, "like")} testID="like" />
        </View>
      ) : null}
      {match ? (
        <View style={[StyleSheet.absoluteFill, styles.matchOverlay, { backgroundColor: colors.overlay }]} accessibilityViewIsModal>
          <View style={[styles.matchCard, { backgroundColor: colors.surface }]}>
            <Text style={{ fontSize: 48 }}>🎉</Text>
            <AppText variant="title" center>
              C'est un match !
            </AppText>
            <AppText muted center>
              Vous et {match.card.firstName} vous plaisez mutuellement.
            </AppText>
            <Button
              label={`Écrire à ${match.card.firstName}`}
              onPress={() => {
                const m = match;
                setMatch(null);
                navigation.navigate("Chat", { matchId: m.matchId, partnerId: m.card.userId, partnerName: m.card.firstName });
              }}
              testID="match-write"
            />
            <Button label="Continuer à découvrir" variant="ghost" onPress={() => setMatch(null)} testID="match-continue" />
          </View>
        </View>
      ) : null}
    </Screen>
  );
}

function RoundButton({
  label,
  glyph,
  color,
  onPress,
  big,
  testID
}: {
  label: string;
  glyph: string;
  color: string;
  onPress: () => void;
  big?: boolean;
  testID?: string;
}) {
  const { colors } = useTheme();
  const size = big ? 72 : 60;
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      onPress={onPress}
      testID={testID}
      style={({ pressed }) => [
        styles.round,
        {
          width: size,
          height: size,
          borderRadius: size / 2,
          backgroundColor: colors.surface,
          borderColor: color,
          opacity: pressed ? 0.7 : 1
        }
      ]}
    >
      <Text style={{ color, fontSize: big ? 32 : 26, fontWeight: "800" }}>{glyph}</Text>
    </Pressable>
  );
}

/** Draggable top card. Release past the threshold flings it off and reports the decision. */
function SwipeCard({ card, width, onDecide, onInfo }: { card: Card; width: number; onDecide: (d: Decision) => void; onInfo: () => void }) {
  const { colors } = useTheme();
  const position = useRef(new Animated.ValueXY()).current;
  const threshold = width * 0.28;
  const decided = useRef(false);

  const fling = useCallback(
    (d: Decision) => {
      if (decided.current) return;
      decided.current = true;
      Animated.timing(position, {
        toValue: { x: (d === "like" ? 1 : -1) * width * 1.3, y: 0 },
        duration: 180,
        useNativeDriver: false
      }).start(() => onDecide(d));
    },
    [onDecide, position, width]
  );

  const responder = useMemo(
    () =>
      PanResponder.create({
        onMoveShouldSetPanResponder: (_e, g) => Math.abs(g.dx) > 12 && Math.abs(g.dx) > Math.abs(g.dy),
        onPanResponderMove: Animated.event([null, { dx: position.x, dy: position.y }], { useNativeDriver: false }),
        onPanResponderRelease: (_e, g) => {
          if (g.dx > threshold) fling("like");
          else if (g.dx < -threshold) fling("pass");
          else Animated.spring(position, { toValue: { x: 0, y: 0 }, useNativeDriver: false, friction: 6 }).start();
        }
      }),
    [fling, position, threshold]
  );

  const rotate = position.x.interpolate({ inputRange: [-width, 0, width], outputRange: ["-12deg", "0deg", "12deg"] });
  const likeOpacity = position.x.interpolate({ inputRange: [0, threshold], outputRange: [0, 1], extrapolate: "clamp" });
  const passOpacity = position.x.interpolate({ inputRange: [-threshold, 0], outputRange: [1, 0], extrapolate: "clamp" });

  return (
    <Animated.View
      {...responder.panHandlers}
      style={[styles.cardSlot, { transform: [{ translateX: position.x }, { translateY: position.y }, { rotate }] }]}
      accessibilityLabel={`${card.firstName}, ${card.age} ans`}
      accessibilityActions={[
        { name: "like", label: "J'aime" },
        { name: "pass", label: "Passer" }
      ]}
      onAccessibilityAction={(e) => fling(e.nativeEvent.actionName === "like" ? "like" : "pass")}
    >
      <CardPhoto card={card} onInfo={onInfo}>
        <SharedBadge count={card.sharedInterests} />
        <Animated.View style={[styles.stamp, styles.stampLike, { opacity: likeOpacity, borderColor: colors.success }]} pointerEvents="none">
          <Text style={[styles.stampText, { color: colors.success }]}>J'AIME</Text>
        </Animated.View>
        <Animated.View style={[styles.stamp, styles.stampPass, { opacity: passOpacity, borderColor: colors.danger }]} pointerEvents="none">
          <Text style={[styles.stampText, { color: colors.danger }]}>PASSER</Text>
        </Animated.View>
      </CardPhoto>
    </Animated.View>
  );
}

const styles = StyleSheet.create({
  header: { paddingHorizontal: spacing.lg, paddingVertical: spacing.sm },
  deck: { flex: 1, marginHorizontal: spacing.lg, marginBottom: spacing.md },
  cardSlot: { ...StyleSheet.absoluteFillObject, borderRadius: radii.lg, overflow: "hidden", backgroundColor: "#222" },
  actions: { flexDirection: "row", justifyContent: "center", alignItems: "center", gap: spacing.xl, paddingBottom: spacing.lg },
  round: {
    alignItems: "center",
    justifyContent: "center",
    borderWidth: 2,
    shadowColor: "#000",
    shadowOpacity: 0.15,
    shadowRadius: 8,
    shadowOffset: { width: 0, height: 3 },
    elevation: 3
  },
  stamp: { position: "absolute", top: 60, borderWidth: 4, borderRadius: radii.md, paddingHorizontal: 12, paddingVertical: 4 },
  stampLike: { left: 24, transform: [{ rotate: "-14deg" }] },
  stampPass: { right: 24, transform: [{ rotate: "14deg" }] },
  stampText: { fontSize: fontSizes.xl, fontWeight: "900", letterSpacing: 2 },
  matchOverlay: { alignItems: "center", justifyContent: "center", padding: spacing.xl },
  matchCard: { width: "100%", maxWidth: 380, borderRadius: radii.lg, padding: spacing.xl, gap: spacing.md, alignItems: "stretch" }
});
