import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useQueryClient } from "@tanstack/react-query";
import React, { useCallback, useEffect, useRef, useState } from "react";
import { StyleSheet, View } from "react-native";
import { datingApi } from "../api/dating";
import { errorMessage, isApiError } from "../api/client";
import type { Card, SwipeAction } from "../api/types";
import { useMatchCelebration } from "../components/MatchOverlay";
import { SwipeDeck } from "../components/SwipeDeck";
import { keys, useMyProfile } from "../hooks/queries";
import { useLocationUpdate } from "../hooks/useLocationUpdate";
import type { MainStackParamList } from "../navigation/types";
import { APP_NAME, radii, spacing, useTheme } from "../theme";
import { Button, EmptyState, ErrorState, LoadingState, Screen, Text, useToast } from "../ui";

const REFILL_AT = 3;

export default function DiscoverScreen() {
  const { colors } = useTheme();
  const navigation = useNavigation<NativeStackNavigationProp<MainStackParamList>>();
  const client = useQueryClient();
  const toast = useToast();
  const { celebrate } = useMatchCelebration();
  const profile = useMyProfile();
  const location = useLocationUpdate();

  const [deck, setDeck] = useState<Card[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [exhausted, setExhausted] = useState(false);
  const handled = useRef(new Set<string>());
  const fetching = useRef(false);
  const deckRef = useRef<Card[]>([]);
  deckRef.current = deck;

  const fill = useCallback(async () => {
    if (fetching.current) {
      return;
    }
    fetching.current = true;
    try {
      const batch = await datingApi.discover(10);
      const known = new Set(deckRef.current.map((c) => c.userId));
      const fresh = batch.filter((c) => !known.has(c.userId) && !handled.current.has(c.userId));
      if (fresh.length > 0) {
        setDeck((current) => {
          const ids = new Set(current.map((c) => c.userId));
          return [...current, ...fresh.filter((c) => !ids.has(c.userId))];
        });
      }
      setExhausted(fresh.length === 0);
      setError(null);
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      fetching.current = false;
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void fill();
  }, [fill]);

  useEffect(() => {
    if (!loading && !exhausted && deck.length <= REFILL_AT) {
      void fill();
    }
  }, [deck.length, loading, exhausted, fill]);

  const onSwipe = useCallback(
    async (card: Card, action: SwipeAction) => {
      handled.current.add(card.userId);
      setDeck((current) => current.filter((c) => c.userId !== card.userId));
      try {
        const result = await datingApi.swipe(card.userId, action);
        if (result.matched && result.matchId && !result.alreadyProcessed) {
          void client.invalidateQueries({ queryKey: keys.matches });
          celebrate(result.matchId);
        }
      } catch (e) {
        if (isApiError(e, 404)) {
          return; // the profile is no longer available: it stays removed from the deck
        }
        handled.current.delete(card.userId);
        setDeck((current) => [card, ...current]);
        toast.show(isApiError(e, 429) ? "Doucement, un instant avant le prochain." : errorMessage(e), "error");
      }
    },
    [celebrate, client, toast]
  );

  const refresh = async () => {
    setLoading(true);
    setExhausted(false);
    await fill();
  };

  const enableLocation = async () => {
    const result = await location.update();
    if (result.ok) {
      client.setQueryData(keys.me, result.profile);
      toast.show("Position enregistrée (approximative)", "success");
      await refresh();
    } else {
      toast.show("Impossible d'obtenir la position.", "error");
    }
  };

  const needsLocation = profile.data?.profile && !profile.data.profile.hasLocation;

  let body: React.ReactNode;
  if (loading && deck.length === 0) {
    body = <LoadingState label="Recherche de profils…" />;
  } else if (error && deck.length === 0) {
    body = <ErrorState message={error} onRetry={() => void refresh()} />;
  } else if (deck.length === 0) {
    body = (
      <EmptyState
        icon="compass-outline"
        title="Vous avez tout vu pour l'instant"
        message="Élargissez votre distance ou votre tranche d'âge, ou revenez un peu plus tard : de nouvelles personnes arrivent chaque jour."
        actionLabel="Actualiser"
        onAction={() => void refresh()}
      />
    );
  } else {
    body = (
      <SwipeDeck
        cards={deck}
        onSwipe={(c, a) => void onSwipe(c, a)}
        onOpenDetails={(c) => navigation.navigate("UserProfile", { userId: c.userId })}
      />
    );
  }

  return (
    <Screen>
      <Text variant="heading" tone="primary" style={styles.title}>
        {APP_NAME}
      </Text>
      {needsLocation ? (
        <View style={[styles.banner, { backgroundColor: colors.primarySoft }]}>
          <Text variant="caption" style={styles.flex}>
            Activez la localisation pour voir des personnes près de vous.
          </Text>
          <Button
            label="Activer"
            variant="secondary"
            onPress={() => void enableLocation()}
            loading={location.loading}
            style={styles.bannerButton}
          />
        </View>
      ) : null}
      <View style={styles.flex}>{body}</View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  title: { marginBottom: spacing.md },
  banner: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    padding: spacing.md,
    borderRadius: radii.md,
    marginBottom: spacing.md
  },
  bannerButton: { minHeight: 36, paddingHorizontal: spacing.lg }
});
