import React, { useEffect, useReducer } from "react";
import { StyleSheet, View } from "react-native";
import type { CompositeScreenProps } from "@react-navigation/native";
import type { BottomTabScreenProps } from "@react-navigation/bottom-tabs";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { ApiError, discoveryApi, queryKeys, type PublicProfile, type SwipeAction } from "../../api";
import { showToast } from "../../shared/feedback";
import { BrandMark, Button, EmptyState, ErrorState, Screen, Skeleton } from "../../shared/ui";
import { radius, spacing } from "../../theme";
import type { MainStackParamList, MainTabParamList } from "../../navigation/types";
import { announceMatch } from "../matches/matchStore";
import { useDeckEpoch } from "./deckEpoch";
import { deckReducer, initialDeckState, needsRefill, PAGE_SIZE } from "./deck";
import { SwipeDeck } from "./SwipeDeck";

type Props = CompositeScreenProps<
  BottomTabScreenProps<MainTabParamList, "Discover">,
  NativeStackScreenProps<MainStackParamList>
>;

export function DiscoveryScreen(props: Props) {
  const epoch = useDeckEpoch();
  return <DiscoveryDeck key={epoch} {...props} />;
}

function DiscoveryDeck({ navigation }: Props) {
  const queryClient = useQueryClient();
  const [deck, dispatch] = useReducer(deckReducer, initialDeckState);

  const refill = useMutation({
    mutationFn: () => discoveryApi.list(PAGE_SIZE),
    meta: { silent: true },
    onSuccess: (profiles) => dispatch({ type: "received", profiles })
  });
  const { mutate: fetchMore, isPending: isFetching, isError: fetchFailed } = refill;

  useEffect(() => {
    if (needsRefill(deck) && !isFetching && !fetchFailed) {
      fetchMore();
    }
  }, [deck, isFetching, fetchFailed, fetchMore]);

  const swipe = useMutation({
    mutationFn: ({ profile, action }: { profile: PublicProfile; action: SwipeAction }) =>
      discoveryApi.swipe(profile.userId, action),
    meta: { silent: true },
    onSuccess: (response) => {
      if (response.result === "matched" && response.match) {
        announceMatch(response.match);
        void queryClient.invalidateQueries({ queryKey: queryKeys.matches });
        void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
      }
    },
    onError: (error, { profile }) => {
      const gone = error instanceof ApiError && (error.status === 404 || error.status === 403);
      if (!gone) {
        dispatch({ type: "restore", profile });
        showToast("Votre choix n'a pas pu être enregistré. Réessayez.", "error");
      }
    }
  });

  const handleSwipe = (profile: PublicProfile, action: SwipeAction) => {
    dispatch({ type: "swiped", userId: profile.userId });
    swipe.mutate({ profile, action });
  };

  const openProfile = (profile: PublicProfile) => {
    queryClient.setQueryData(queryKeys.publicProfile(profile.userId), profile);
    navigation.navigate("UserProfile", { userId: profile.userId });
  };

  const retry = () => {
    dispatch({ type: "resume" });
    fetchMore();
  };

  let body: React.ReactNode;
  if (deck.queue.length > 0) {
    body = <SwipeDeck onOpenProfile={openProfile} onSwipe={handleSwipe} queue={deck.queue} />;
  } else if (fetchFailed) {
    body = <ErrorState message="Impossible de charger de nouveaux profils." onRetry={retry} testID="deck-error" />;
  } else if (deck.exhausted) {
    body = (
      <EmptyState
        actionLabel="Actualiser"
        icon="leaf-outline"
        message="Vous avez tout vu pour le moment. Revenez plus tard ou élargissez vos critères."
        onAction={retry}
        testID="deck-empty"
        title="Plus personne autour de vous"
      />
    );
  } else {
    body = <DeckSkeleton />;
  }

  return (
    <Screen edges={["top"]} padded={false} testID="discover-screen">
      <View style={styles.header}>
        <BrandMark size={32} />
        <Button
          label="Critères"
          icon="options-outline"
          onPress={() => navigation.navigate("EditPreferences")}
          variant="ghost"
        />
      </View>
      {body}
    </Screen>
  );
}

function DeckSkeleton() {
  return (
    <View accessibilityLabel="Chargement des profils" style={styles.skeleton} testID="deck-loading">
      <Skeleton height="100%" style={styles.skeletonCard} />
    </View>
  );
}

const styles = StyleSheet.create({
  header: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    paddingHorizontal: spacing.lg,
    paddingBottom: spacing.sm
  },
  skeleton: { flex: 1, padding: spacing.lg },
  skeletonCard: { flex: 1, borderRadius: radius.lg }
});
