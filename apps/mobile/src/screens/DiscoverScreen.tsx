import React, { useCallback, useRef, useState } from "react";
import { StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { getFeed, swipe } from "../api/dating";
import type { Card, MatchItem } from "../api/types";
import { MatchModal } from "../components/MatchModal";
import { SwipeDeck } from "../components/SwipeDeck";
import { queryKeys } from "../lib/queryKeys";
import type { MainStackParamList } from "../navigation/types";
import { Screen } from "../shared/layout";
import { EmptyView, ErrorView, LoadingView } from "../shared/feedback";
import { Text, spacing } from "../shared/ui";

const REFILL_AT = 3;

export default function DiscoverScreen() {
  const navigation = useNavigation<NativeStackNavigationProp<MainStackParamList>>();
  const queryClient = useQueryClient();
  const [seen, setSeen] = useState<Set<string>>(() => new Set());
  const [match, setMatch] = useState<MatchItem | null>(null);
  const [swipeError, setSwipeError] = useState<string | null>(null);

  const feed = useQuery({ queryKey: queryKeys.feed, queryFn: () => getFeed(10), staleTime: 60_000 });

  // Remaining cards are read through a ref so onSettled never sees a stale closure.
  const remainingRef = useRef(0);
  const refillFeed = feed.refetch;

  const mutation = useMutation({
    mutationFn: ({ card, kind }: { card: Card; kind: "like" | "pass" }) => swipe(card.id, kind),
    // Refill only once the server has recorded the swipe, so swiped profiles cannot come back.
    onSettled: () => {
      if (remainingRef.current <= REFILL_AT) {
        void refillFeed().then(() => setSeen(new Set()));
      }
    },
    onSuccess: (result) => {
      if (result.matched && result.match) {
        setMatch(result.match);
        void queryClient.invalidateQueries({ queryKey: queryKeys.matches });
        void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
      }
    },
    onError: () => setSwipeError("Action non enregistrée. Vérifiez votre connexion.")
  });

  // Cards already swiped locally are hidden immediately; the server also excludes them from future feeds.
  const cards = (feed.data ?? []).filter((c) => !seen.has(c.id));
  remainingRef.current = cards.length;

  const handle = useCallback(
    (card: Card, kind: "like" | "pass") => {
      setSwipeError(null);
      setSeen((prev) => new Set(prev).add(card.id));
      remainingRef.current = Math.max(0, remainingRef.current - 1);
      mutation.mutate({ card, kind });
    },
    [mutation]
  );

  let body: React.ReactNode;
  if (feed.isLoading) {
    body = <LoadingView label="Recherche de profils…" />;
  } else if (feed.isError) {
    body = <ErrorView actionLabel="Réessayer" message="Impossible de charger les profils." onAction={() => void feed.refetch()} />;
  } else if (cards.length === 0) {
    body = feed.isFetching ? (
      <LoadingView label="Recherche de profils…" />
    ) : (
      <EmptyView
        actionLabel="Actualiser"
        message="Vous avez tout vu pour l'instant. Élargissez vos préférences (âge, distance) ou revenez plus tard."
        onAction={() => {
          setSeen(new Set());
          void feed.refetch();
        }}
        title="Plus personne autour de vous"
      />
    );
  } else {
    body = (
      <SwipeDeck
        cards={cards}
        onLike={(c) => handle(c, "like")}
        onOpenProfile={(c) => navigation.navigate("ProfileDetail", { id: c.id, swipeable: true })}
        onPass={(c) => handle(c, "pass")}
      />
    );
  }

  return (
    <Screen>
      <View style={styles.header}>
        <Text variant="heading" weight="bold">
          Découvrir
        </Text>
      </View>
      {swipeError ? (
        <Text tone="danger" variant="caption">
          {swipeError}
        </Text>
      ) : null}
      <View style={styles.body}>{body}</View>
      <MatchModal
        match={match}
        onChat={(m) => {
          setMatch(null);
          navigation.navigate("Chat", { conversationId: m.conversationId });
        }}
        onClose={() => setMatch(null)}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  header: { paddingBottom: spacing.md },
  body: { flex: 1, justifyContent: "center" }
});
