import { Ionicons } from "@expo/vector-icons";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import React, { useCallback, useEffect, useRef, useState } from "react";
import { Modal, Pressable, StyleSheet, View } from "react-native";

import { ApiError } from "../../api/client";
import { discoverApi } from "../../api/endpoints";
import type { Card, SwipeResult } from "../../api/types";
import type { MainStackParamList } from "../../navigation/types";
import { keys } from "../../realtime/RealtimeProvider";
import { radii, spacing, useTheme } from "../../theme";
import { Button } from "../../ui/Button";
import { useFeedback } from "../../ui/Feedback";
import { PhotoImage } from "../../ui/PhotoImage";
import { Screen } from "../../ui/Screen";
import { EmptyState, ErrorState, Loading } from "../../ui/States";
import { SwipeCard, type SwipeCardHandle, type SwipeDirection } from "../../ui/SwipeCard";
import { Text } from "../../ui/Text";

type Nav = NativeStackNavigationProp<MainStackParamList>;

const BATCH = 10;
const REFILL_AT = 3;

export function DiscoverScreen() {
  const { colors } = useTheme();
  const navigation = useNavigation<Nav>();
  const qc = useQueryClient();
  const { toast } = useFeedback();

  const first = useQuery({
    queryKey: keys.discover,
    queryFn: () => discoverApi.list(BATCH),
    staleTime: Infinity,
    gcTime: Infinity,
    refetchOnWindowFocus: false,
  });

  const [queue, setQueue] = useState<Card[]>([]);
  const [exhausted, setExhausted] = useState(false);
  const [match, setMatch] = useState<SwipeResult | null>(null);
  const seen = useRef(new Set<string>());
  const appliedAt = useRef(0);
  const fetching = useRef(false);
  const topRef = useRef<SwipeCardHandle>(null);

  // A new first batch (initial load, or invalidated after profile/preferences changes) resets the deck.
  useEffect(() => {
    if (!first.data || first.dataUpdatedAt === appliedAt.current) return;
    appliedAt.current = first.dataUpdatedAt;
    seen.current = new Set(first.data.profiles.map((p) => p.id));
    setQueue(first.data.profiles);
    setExhausted(first.data.profiles.length === 0);
  }, [first.data, first.dataUpdatedAt]);

  const refill = useCallback(async () => {
    if (fetching.current) return;
    fetching.current = true;
    try {
      const { profiles } = await discoverApi.list(BATCH);
      const fresh = profiles.filter((p) => !seen.current.has(p.id));
      fresh.forEach((p) => seen.current.add(p.id));
      setQueue((q) => [...q, ...fresh]);
      if (fresh.length === 0) setExhausted(true);
    } catch {
      // keep what we have; the user can pull again with the refresh button
    } finally {
      fetching.current = false;
    }
  }, []);

  useEffect(() => {
    if (first.data && !exhausted && queue.length > 0 && queue.length <= REFILL_AT) void refill();
  }, [queue.length, exhausted, first.data, refill]);

  const swipe = useCallback(
    async (card: Card, action: SwipeDirection) => {
      setQueue((q) => q.filter((c) => c.id !== card.id));
      try {
        const result = await discoverApi.swipe(card.id, action);
        if (result.matched && action === "like") {
          setMatch(result);
          void qc.invalidateQueries({ queryKey: keys.matches });
          void qc.invalidateQueries({ queryKey: keys.summary });
        }
      } catch (err) {
        if (err instanceof ApiError && err.code === "not_available") return; // profile vanished: nothing to undo
        setQueue((q) => [card, ...q]);
        toast("Action impossible pour le moment. Réessayez.", "error");
      }
    },
    [qc, toast],
  );

  const reload = () => {
    seen.current = new Set();
    setExhausted(false);
    void qc.invalidateQueries({ queryKey: keys.discover });
  };

  if (first.isLoading) return <Loading label="Recherche de profils…" />;
  if (first.isError) {
    if (first.error instanceof ApiError && first.error.code === "profile_incomplete") {
      return (
        <EmptyState
          icon="person-circle-outline"
          title="Complétez votre profil"
          message={first.error.message}
        />
      );
    }
    return <ErrorState error={first.error} onRetry={() => void first.refetch()} />;
  }

  const top = queue[0];
  const next = queue[1];

  return (
    <Screen edges={["top"]} padded={false}>
      <View style={styles.header}>
        <Text variant="title" tone="primary">
          Alba
        </Text>
      </View>

      {top ? (
        <>
          <View style={styles.deck}>
            {next ? (
              <View style={[styles.behind]} pointerEvents="none">
                <SwipeCard
                  key={next.id}
                  card={next}
                  interactive={false}
                  onSwiped={() => undefined}
                  onOpen={() => undefined}
                />
              </View>
            ) : null}
            <SwipeCard
              key={top.id}
              ref={topRef}
              card={top}
              interactive
              onSwiped={(d) => void swipe(top, d)}
              onOpen={() => navigation.navigate("ProfileDetail", { userId: top.id })}
            />
          </View>
          <View style={styles.actions}>
            <RoundButton
              icon="close"
              label="Passer"
              color={colors.pass}
              onPress={() => topRef.current?.swipe("pass")}
              testID="swipe-pass"
            />
            <RoundButton
              icon="information"
              label="Voir le profil"
              color={colors.accent}
              small
              onPress={() => navigation.navigate("ProfileDetail", { userId: top.id })}
            />
            <RoundButton
              icon="heart"
              label="J'aime"
              color={colors.like}
              onPress={() => topRef.current?.swipe("like")}
              testID="swipe-like"
            />
          </View>
        </>
      ) : (
        <EmptyState
          icon="compass-outline"
          title="Vous avez fait le tour"
          message="Plus personne à découvrir pour l'instant. Élargissez vos préférences ou revenez un peu plus tard."
          actionLabel="Actualiser"
          onAction={reload}
        />
      )}

      <Modal
        visible={match !== null}
        transparent
        animationType="fade"
        onRequestClose={() => setMatch(null)}
      >
        <View style={[styles.matchBackdrop, { backgroundColor: colors.overlay }]}>
          <View style={[styles.matchCard, { backgroundColor: colors.surface }]}>
            <PhotoImage
              photo={match?.user?.photos[0]}
              name={match?.user?.firstName}
              round
              size={112}
            />
            <Text variant="display" tone="primary" center>
              C&apos;est un match !
            </Text>
            <Text tone="muted" center>
              Vous et {match?.user?.firstName} vous plaisez mutuellement.
            </Text>
            <Button
              label="Envoyer un message"
              testID="match-chat"
              onPress={() => {
                const m = match;
                setMatch(null);
                if (m?.matchId && m.user)
                  navigation.navigate("Chat", {
                    matchId: m.matchId,
                    name: m.user.firstName,
                    userId: m.user.id,
                  });
              }}
            />
            <Button label="Continuer" variant="ghost" onPress={() => setMatch(null)} />
          </View>
        </View>
      </Modal>
    </Screen>
  );
}

function RoundButton({
  icon,
  label,
  color,
  onPress,
  small,
  testID,
}: {
  icon: React.ComponentProps<typeof Ionicons>["name"];
  label: string;
  color: string;
  onPress: () => void;
  small?: boolean;
  testID?: string;
}) {
  const { colors } = useTheme();
  const size = small ? 52 : 68;
  return (
    <Pressable
      testID={testID}
      accessibilityRole="button"
      accessibilityLabel={label}
      onPress={onPress}
      style={({ pressed }) => [
        styles.round,
        {
          width: size,
          height: size,
          borderRadius: size / 2,
          backgroundColor: colors.surface,
          borderColor: color,
          opacity: pressed ? 0.7 : 1,
        },
      ]}
    >
      <Ionicons name={icon} size={small ? 24 : 32} color={color} />
    </Pressable>
  );
}

const styles = StyleSheet.create({
  header: { paddingHorizontal: spacing.lg, paddingVertical: spacing.sm },
  deck: { flex: 1, marginHorizontal: spacing.lg, marginBottom: spacing.md },
  behind: {
    ...StyleSheet.absoluteFillObject,
    transform: [{ scale: 0.95 }, { translateY: 10 }],
    opacity: 0.9,
  },
  actions: {
    flexDirection: "row",
    justifyContent: "center",
    alignItems: "center",
    gap: spacing.xl,
    paddingBottom: spacing.lg,
  },
  round: { alignItems: "center", justifyContent: "center", borderWidth: 2 },
  matchBackdrop: { flex: 1, alignItems: "center", justifyContent: "center", padding: spacing.xl },
  matchCard: {
    width: "100%",
    maxWidth: 400,
    borderRadius: radii.lg,
    padding: spacing.xl,
    alignItems: "center",
    gap: spacing.md,
  },
});
