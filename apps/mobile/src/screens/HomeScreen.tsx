import React, { useCallback, useEffect, useRef, useState } from "react";
import { StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { ApiError } from "../api/client";
import { discoverApi, profileApi } from "../api/platform";
import { queryKeys } from "../api/queryClient";
import type { MatchSummary, PublicProfile, SwipeAction } from "../api/types";
import { MatchModal } from "../components/MatchModal";
import { SwipeDeck } from "../components/SwipeDeck";
import { useDeviceLocation } from "../hooks/useDeviceLocation";
import type { RootStackParamList } from "../navigation/types";
import { EmptyView, ErrorView, LoadingView, showToast } from "../shared/feedback";
import { SafeAreaLayout } from "../shared/layout";
import { Button, Notice, Text, spacing } from "../shared/ui";
import { applyBatch, needsRefill } from "../utils/deck";
import { subscribeSwiped } from "../utils/deckEvents";

type Nav = NativeStackNavigationProp<RootStackParamList>;

/** Discovery: a queue of candidates fetched in small batches and refilled when it runs low. */
export default function HomeScreen() {
  const navigation = useNavigation<Nav>();
  const queryClient = useQueryClient();
  const [queue, setQueue] = useState<PublicProfile[]>([]);
  const [fetching, setFetching] = useState(false);
  const [exhausted, setExhausted] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [match, setMatch] = useState<MatchSummary | null>(null);
  const swiped = useRef(new Set<string>());
  const queueRef = useRef<PublicProfile[]>([]);
  queueRef.current = queue;
  const location = useDeviceLocation();
  const profile = useQuery({ queryKey: queryKeys.profile, queryFn: profileApi.getOwn });

  const fetchBatch = useCallback(async () => {
    setFetching(true);
    try {
      const incoming = await discoverApi.next(10);
      const result = applyBatch(queueRef.current, incoming, swiped.current);
      setLoadError(null);
      setExhausted(result.exhausted);
      setQueue(result.queue);
    } catch (error) {
      setLoadError(error instanceof ApiError ? error.message : "Impossible de charger les profils.");
      setExhausted(true);
    } finally {
      setFetching(false);
    }
  }, []);

  useEffect(() => {
    if (needsRefill(queue.length, fetching, exhausted)) {
      void fetchBatch();
    }
  }, [queue.length, fetching, exhausted, fetchBatch]);

  useEffect(
    () =>
      subscribeSwiped((userId) => {
        swiped.current.add(userId);
        setQueue((current) => current.filter((p) => p.id !== userId));
      }),
    []
  );

  const refresh = useCallback(() => {
    setExhausted(false);
    setLoadError(null);
    setQueue([]);
    void fetchBatch();
  }, [fetchBatch]);

  const handleSwipe = useCallback(
    async (target: PublicProfile, action: SwipeAction) => {
      swiped.current.add(target.id);
      setQueue((current) => current.filter((p) => p.id !== target.id));
      try {
        const result = await discoverApi.swipe(target.id, action);
        if (result.matched && result.match && !result.alreadySwiped) {
          setMatch(result.match);
          void queryClient.invalidateQueries({ queryKey: queryKeys.matches });
          void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
        }
      } catch (error) {
        if (error instanceof ApiError && error.status === 404) {
          return; // profile became unavailable (blocked, hidden…): the card is simply gone
        }
        // the swipe did not reach the server: put the card back so nothing is lost silently
        swiped.current.delete(target.id);
        setQueue((current) => [target, ...current.filter((p) => p.id !== target.id)]);
        showToast("Action non enregistrée. Réessayez.");
      }
    },
    [queryClient]
  );

  const openMatch = (m: MatchSummary) => {
    setMatch(null);
    navigation.navigate("Conversation", { conversationId: m.conversationId, user: m.user });
  };

  const noLocation = profile.data && !profile.data.hasLocation && location.state !== "done";

  return (
    <SafeAreaLayout edges={["top"]}>
      <View style={styles.root} testID="discover-screen">
        <View style={styles.header}>
          <Text tone="primary" variant="eyebrow" weight="bold">
            Lumen
          </Text>
          <Text accessibilityRole="header" variant="heading">
            Découvrir
          </Text>
        </View>

        {noLocation ? (
          <View style={styles.banner}>
            <Notice message="Activez votre position pour voir des profils près de vous." tone="info" />
            <Button
              label="Activer la position"
              loading={location.state === "working"}
              onPress={() => void location.share().then(refresh)}
              size="sm"
              variant="secondary"
            />
          </View>
        ) : null}

        <View style={styles.deck}>
          {queue.length > 0 ? (
            <SwipeDeck
              onOpenProfile={(p) => navigation.navigate("ProfileDetail", { userId: p.id, profile: p, canSwipe: true })}
              onSwipe={(p, action) => void handleSwipe(p, action)}
              profiles={queue}
            />
          ) : fetching ? (
            <LoadingView label="Nous cherchons des profils…" />
          ) : loadError ? (
            <ErrorView message={loadError} onAction={refresh} />
          ) : (
            <EmptyView
              actionLabel="Actualiser"
              icon="sparkles-outline"
              message="Vous avez vu tout le monde pour l'instant. Élargissez vos préférences ou revenez un peu plus tard."
              onAction={refresh}
              title="C'est tout pour le moment"
            />
          )}
        </View>
        {queue.length === 0 && !fetching && !loadError ? (
          <Button label="Modifier mes préférences" onPress={() => navigation.navigate("Settings")} variant="ghost" />
        ) : null}
      </View>
      <MatchModal match={match} onClose={() => setMatch(null)} onMessage={openMatch} />
    </SafeAreaLayout>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, paddingHorizontal: spacing.lg, paddingBottom: spacing.sm, gap: spacing.sm },
  header: { paddingTop: spacing.sm },
  banner: { gap: spacing.sm },
  deck: { flex: 1 }
});
