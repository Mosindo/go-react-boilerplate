import React, { useCallback, useEffect, useRef, useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import type { CompositeScreenProps } from "@react-navigation/native";
import type { BottomTabScreenProps } from "@react-navigation/bottom-tabs";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { ApiError } from "../../api/client";
import { discover, swipe } from "../../api/social";
import type { Match, PublicProfile, SwipeAction } from "../../api/types";
import { MatchModal } from "../../components/MatchModal";
import { SwipeCard } from "../../components/SwipeCard";
import { onDiscoverReset } from "../../lib/discoverEvents";
import { errorMessage } from "../../lib/errors";
import type { MainStackParams, TabParams } from "../../navigation/types";
import { EmptyState, ErrorState, Loading, PhotoImage, Screen, Text, useToast } from "../../ui";
import { MIN_TOUCH, spacing } from "../../theme/tokens";
import { useTheme } from "../../theme/ThemeProvider";

type Props = CompositeScreenProps<
  BottomTabScreenProps<TabParams, "Discover">,
  NativeStackScreenProps<MainStackParams>
>;

const REFILL_AT = 3;

export default function DiscoverScreen({ navigation }: Props) {
  const { colors } = useTheme();
  const toast = useToast();
  const [queue, setQueue] = useState<PublicProfile[]>([]);
  const [status, setStatus] = useState<"loading" | "ready" | "error" | "incomplete">("loading");
  const [error, setError] = useState<unknown>(null);
  const [exhausted, setExhausted] = useState(false);
  const [match, setMatch] = useState<Match | null>(null);
  const loadingRef = useRef(false);
  const seen = useRef(new Set<string>());
  const cardRef = useRef<{ fling: (d: "like" | "pass") => void }>(null);

  const load = useCallback(async () => {
    if (loadingRef.current) return;
    loadingRef.current = true;
    try {
      const batch = (await discover(10)).filter((p) => !seen.current.has(p.userId));
      batch.forEach((p) => seen.current.add(p.userId));
      setQueue((current) => [...current, ...batch]);
      setExhausted(batch.length === 0);
      setStatus("ready");
    } catch (e) {
      if (e instanceof ApiError && e.code === "profile_incomplete") setStatus("incomplete");
      else {
        setError(e);
        setStatus((s) => (s === "loading" ? "error" : s));
        toast.show(errorMessage(e), "error");
      }
    } finally {
      loadingRef.current = false;
    }
  }, [toast]);

  useEffect(() => {
    void load();
  }, [load]);

  // Preferences changed: drop the buffer (already-swiped ids stay excluded) and refill.
  useEffect(
    () =>
      onDiscoverReset(() => {
        setQueue([]);
        setExhausted(false);
        setStatus("loading");
        void load();
      }),
    [load]
  );

  // Keep the buffer topped up and warm the next card's photo.
  useEffect(() => {
    if (status === "ready" && !exhausted && queue.length <= REFILL_AT) void load();
  }, [queue, status, exhausted, load]);

  const top = queue[0];

  const handleSwipe = async (profile: PublicProfile, action: SwipeAction) => {
    setQueue((current) => current.filter((p) => p.userId !== profile.userId));
    try {
      const result = await swipe(profile.userId, action);
      if (result.matched && result.match && !result.duplicate) setMatch(result.match);
    } catch (e) {
      if (!(e instanceof ApiError && e.code === "profile_unavailable")) toast.show(errorMessage(e), "error");
    }
  };

  const refresh = () => {
    setExhausted(false);
    setStatus("loading");
    void load();
  };

  let body: React.ReactNode;
  if (status === "loading") body = <Loading label="Recherche de profils…" />;
  else if (status === "error") body = <ErrorState error={error} onRetry={refresh} />;
  else if (status === "incomplete")
    body = (
      <EmptyState
        icon="📷"
        title="Ajoutez une photo"
        message="Pour découvrir des profils, votre profil doit contenir au moins une photo."
        actionLabel="Gérer mes photos"
        onAction={() => navigation.navigate("Photos")}
      />
    );
  else if (!top)
    body = (
      <EmptyState
        icon="🌙"
        title="Vous avez tout vu pour le moment"
        message="Revenez plus tard ou élargissez vos préférences (âge, distance) pour voir plus de monde."
        actionLabel="Actualiser"
        onAction={refresh}
      />
    );
  else
    body = (
      <>
        <View style={styles.cardWrap}>
          <SwipeCard
            key={top.userId}
            ref={cardRef}
            profile={top}
            onSwipe={(direction) => void handleSwipe(top, direction)}
            onOpenProfile={() =>
              navigation.navigate("ProfileDetail", { userId: top.userId, profile: top, canSwipe: true })
            }
          />
        </View>
        {/* Renders the next photo off-screen so the following card appears instantly. */}
        {queue[1]?.photos[0] ? <PhotoImage photoId={queue[1].photos[0].id} style={styles.preload} /> : null}
        <View style={styles.actions}>
          <RoundButton
            label="Passer"
            symbol="✕"
            color={colors.pass}
            onPress={() => cardRef.current?.fling("pass")}
            testID="pass-button"
          />
          <RoundButton
            label="J'aime"
            symbol="♥"
            color={colors.like}
            big
            onPress={() => cardRef.current?.fling("like")}
            testID="like-button"
          />
        </View>
      </>
    );

  return (
    <Screen edges={["top"]} padded={false} testID="discover-screen">
      <View style={styles.header}>
        <Text variant="title" tone="accent">
          Ember
        </Text>
      </View>
      {body}
      <MatchModal
        match={match}
        onClose={() => setMatch(null)}
        onMessage={(m) => {
          setMatch(null);
          navigation.navigate("Chat", { conversationId: m.conversationId, name: m.profile.firstName });
        }}
      />
    </Screen>
  );
}

function RoundButton({
  label,
  symbol,
  color,
  big,
  onPress,
  testID
}: {
  label: string;
  symbol: string;
  color: string;
  big?: boolean;
  onPress: () => void;
  testID: string;
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
          borderColor: color,
          backgroundColor: colors.surface,
          opacity: pressed ? 0.8 : 1
        }
      ]}
    >
      <Text style={{ color, fontSize: big ? 32 : 26, lineHeight: big ? 38 : 32 }} accessibilityElementsHidden>
        {symbol}
      </Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  header: { paddingHorizontal: spacing.lg, paddingVertical: spacing.sm, minHeight: MIN_TOUCH },
  cardWrap: { flex: 1, paddingHorizontal: spacing.md },
  actions: {
    flexDirection: "row",
    justifyContent: "center",
    alignItems: "center",
    gap: spacing.xxl,
    paddingVertical: spacing.lg
  },
  preload: { position: "absolute", width: 1, height: 1, opacity: 0 },
  round: { borderWidth: 2, alignItems: "center", justifyContent: "center" }
});
