import React, { useEffect, useRef } from "react";
import { StyleSheet, View } from "react-native";
import { useQuery } from "@tanstack/react-query";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { discoverApi } from "../../api/endpoints";
import { keys, useNotifications } from "../../hooks/queries";
import type { RootParamList } from "../../navigation/types";
import { spacing } from "../../theme/theme";
import { IconButton } from "../../ui/Button";
import { EmptyState, ErrorState, LoadingView } from "../../ui/Feedback";
import { Screen } from "../../ui/Screen";
import { Text } from "../../ui/Text";
import { SwipeDeck } from "./SwipeDeck";
import { useSwipe } from "./useSwipe";
import { wasSwiped } from "./swipedMemory";

const BATCH_SIZE = 10;

export function DiscoverScreen() {
  const nav = useNavigation<NativeStackNavigationProp<RootParamList>>();
  const swipe = useSwipe();
  const notifications = useNotifications();
  const unread = notifications.data?.pages[0]?.unreadCount ?? 0;

  // Size of the last server batch: a short batch means everybody eligible has already been returned.
  const lastBatchSize = useRef(0);
  const deck = useQuery({
    queryKey: keys.discover,
    queryFn: async () => {
      const batch = await discoverApi.next(BATCH_SIZE);
      lastBatchSize.current = batch.length;
      return batch.filter((c) => !wasSwiped(c.userId));
    },
    staleTime: 5 * 60_000
  });
  const cards = deck.data ?? [];

  // Keep the deck topped up: fetch the next batch while a few cards remain.
  const { refetch, isFetching } = deck;
  useEffect(() => {
    if (
      deck.isSuccess &&
      cards.length > 0 &&
      cards.length <= 3 &&
      lastBatchSize.current >= BATCH_SIZE &&
      !isFetching
    )
      void refetch();
  }, [cards.length, deck.isSuccess, isFetching, refetch]);

  return (
    <Screen padded={false} edges={["top"]}>
      <View style={styles.header}>
        <Text variant="title">Ember</Text>
        <IconButton
          icon="notifications-outline"
          label="Notifications"
          badge={unread}
          onPress={() => nav.navigate("Notifications")}
          testID="open-notifications"
        />
      </View>

      {deck.isPending ? (
        <LoadingView label="Recherche de profils…" />
      ) : deck.isError ? (
        <ErrorState message={deck.error.message} onRetry={() => void refetch()} />
      ) : cards.length === 0 ? (
        <EmptyState
          icon="compass-outline"
          title="Tu as fait le tour pour l'instant"
          message="Plus personne ne correspond à tes critères autour de toi. Élargis ta distance ou ta tranche d'âge, ou reviens un peu plus tard."
          actionLabel="Modifier mes préférences"
          onAction={() => nav.navigate("Preferences")}
        />
      ) : (
        <SwipeDeck
          cards={cards}
          onSwipe={(card, action) => swipe.mutate({ card, action })}
          onOpenProfile={(card) => nav.navigate("UserProfile", { userId: card.userId, card, canSwipe: true })}
        />
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  header: {
    flexDirection: "row",
    justifyContent: "space-between",
    alignItems: "center",
    paddingHorizontal: spacing.md,
    paddingTop: spacing.xs
  }
});
