import React, { useEffect, useRef } from "react";
import { StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { matchingApi } from "../../api/endpoints";
import type { ProfileCard, SwipeDecision } from "../../api/types";
import { SwipeCard, type SwipeCardHandle } from "../../components/discover/SwipeCard";
import { Header } from "../../components/Header";
import { IconButton } from "../../components/IconButton";
import { Screen } from "../../components/Screen";
import { EmptyState, ErrorState, LoadingState } from "../../components/States";
import { useSwipe } from "../../hooks/useSwipe";
import { queryKeys } from "../../queryClient";
import { spacing } from "../../theme/tokens";
import { errorCode, errorMessage } from "../../utils/errors";

const BATCH = 10;

export default function DiscoverScreen() {
  const navigation = useNavigation();
  const topCard = useRef<SwipeCardHandle>(null);
  const client = useQueryClient();
  const swipe = useSwipe();
  // Size of the last batch returned by the server: a short batch means the
  // pool is exhausted, so we stop prefetching until the member refreshes.
  const lastBatch = useRef(BATCH);
  const deck = useQuery({
    queryKey: queryKeys.discovery,
    queryFn: async () => {
      const fresh = (await matchingApi.discover(BATCH)).profiles;
      lastBatch.current = fresh.length;
      // Keep the cards already on screen in place and append the new ones.
      const current = client.getQueryData<ProfileCard[]>(queryKeys.discovery) ?? [];
      const known = new Set(current.map((c) => c.userId));
      return [...current, ...fresh.filter((c) => !known.has(c.userId))];
    },
    staleTime: 5 * 60_000
  });

  const cards: ProfileCard[] = deck.data ?? [];
  const { isSuccess, isFetching, refetch } = deck;

  // Prefetch the next batch while a couple of cards remain. The server never
  // returns profiles that were already swiped.
  useEffect(() => {
    if (isSuccess && !isFetching && !swipe.isPending && cards.length <= 2 && lastBatch.current === BATCH) {
      void refetch();
    }
  }, [cards.length, isSuccess, isFetching, refetch, swipe.isPending]);

  const refresh = () => {
    lastBatch.current = BATCH;
    void client.resetQueries({ queryKey: queryKeys.discovery });
  };

  const onSwiped = (card: ProfileCard, decision: SwipeDecision) => swipe.mutate({ card, decision });

  const header = (
    <Header
      title="Découvrir"
      right={<IconButton icon="options-outline" label="Préférences de rencontre" onPress={() => navigation.navigate("Preferences")} testID="discover-preferences" />}
    />
  );

  let body: React.ReactNode;
  if (deck.isPending) {
    body = <LoadingState label="On cherche des personnes près de toi…" />;
  } else if (deck.isError) {
    body =
      errorCode(deck.error) === "profile_incomplete" ? (
        <EmptyState icon="person-circle-outline" title="Profil incomplet" message={errorMessage(deck.error)} actionLabel="Compléter mon profil" onAction={() => navigation.navigate("Main", { screen: "Me" })} />
      ) : (
        <ErrorState message={errorMessage(deck.error)} onRetry={() => void deck.refetch()} />
      );
  } else if (cards.length === 0) {
    body = deck.isFetching ? (
      <LoadingState />
    ) : (
      <EmptyState
        icon="compass-outline"
        title="Plus personne pour le moment"
        message="Tu as vu tous les profils correspondant à tes critères. Élargis ta distance ou ta tranche d'âge (icône en haut à droite), ou reviens plus tard."
        actionLabel="Actualiser"
        onAction={refresh}
        testID="discover-empty"
      />
    );
  } else {
    const visible = cards.slice(0, 2);
    body = (
      <>
        <View style={styles.deck}>
          {visible
            .map((card, index) => (
              <SwipeCard
                key={card.userId}
                ref={index === 0 ? topCard : undefined}
                card={card}
                active={index === 0}
                onSwiped={(decision) => onSwiped(card, decision)}
                onOpen={() => navigation.navigate("ProfileDetail", { userId: card.userId, from: "discover" })}
              />
            ))
            .reverse()}
        </View>
        <View style={styles.actions}>
          <IconButton icon="close" label="Passer" size={64} onPress={() => topCard.current?.swipe("pass")} testID="discover-pass" />
          <IconButton icon="heart" label="J'aime" size={76} tone="primary" onPress={() => topCard.current?.swipe("like")} testID="discover-like" />
        </View>
      </>
    );
  }

  return (
    <Screen testID="discover-screen">
      {header}
      {body}
    </Screen>
  );
}

const styles = StyleSheet.create({
  deck: { flex: 1, marginHorizontal: spacing.lg, marginTop: spacing.xs },
  actions: { flexDirection: "row", justifyContent: "center", alignItems: "center", gap: spacing.xl, paddingVertical: spacing.lg, paddingBottom: spacing.md }
});
