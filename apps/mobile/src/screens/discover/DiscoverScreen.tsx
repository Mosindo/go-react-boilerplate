import React, { useEffect, useRef } from "react";
import { StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import { useQuery } from "@tanstack/react-query";
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
  const swipe = useSwipe();
  const deck = useQuery({
    queryKey: queryKeys.discovery,
    queryFn: async () => (await matchingApi.discover(BATCH)).profiles,
    staleTime: 5 * 60_000
  });

  const cards: ProfileCard[] = deck.data ?? [];

  // Prefetch the next batch while a couple of cards remain. The server never
  // returns profiles that were already swiped.
  useEffect(() => {
    if (deck.isSuccess && !deck.isFetching && cards.length > 0 && cards.length <= 2 && !swipe.isPending) {
      void deck.refetch();
    }
  }, [cards.length, deck, swipe.isPending]);

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
        message="Tu as vu tous les profils correspondant à tes critères. Élargis ta distance ou ta tranche d'âge, ou reviens plus tard."
        actionLabel="Ajuster mes préférences"
        onAction={() => navigation.navigate("Preferences")}
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
