import React, { useCallback, useEffect, useRef, useState } from "react";
import { StyleSheet, View } from "react-native";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { discoveryApi } from "../../api/endpoints";
import { errorMessage, isApiError } from "../../api/client";
import { queryKeys } from "../../api/queryKeys";
import type { DiscoverCard, Match, SwipeAction } from "../../api/types";
import { useMyProfile } from "../../hooks";
import type { TabScreenProps } from "../../navigation/types";
import { colors, spacing } from "../../theme";
import { EmptyState, ErrorState, IconButton, LoadingState, Screen, Text, showToast } from "../../ui";
import { MatchCelebration } from "./MatchCelebration";
import { SwipeDeck, type SwipeDeckHandle } from "./SwipeDeck";

const REFILL_THRESHOLD = 3;

export function DiscoverScreen({ navigation }: TabScreenProps<"Discover">) {
  const queryClient = useQueryClient();
  const me = useMyProfile();
  const deck = useRef<SwipeDeckHandle>(null);
  const [queue, setQueue] = useState<DiscoverCard[]>([]);
  const [lastPass, setLastPass] = useState<DiscoverCard | null>(null);
  const [match, setMatch] = useState<Match | null>(null);
  const handled = useRef(new Set<string>());

  const feed = useQuery({
    queryKey: queryKeys.discover,
    queryFn: async () => (await discoveryApi.feed(10)).profiles,
    staleTime: 0
  });

  // Merge fresh results into the local queue, skipping anything already handled.
  useEffect(() => {
    if (!feed.data) return;
    setQueue((current) => {
      const known = new Set(current.map((c) => c.userId));
      const fresh = feed.data.filter((c) => !known.has(c.userId) && !handled.current.has(c.userId));
      return fresh.length ? [...current, ...fresh] : current;
    });
  }, [feed.data]);

  const { refetch, isFetching } = feed;
  useEffect(() => {
    if (queue.length < REFILL_THRESHOLD && !isFetching && feed.data && feed.data.length > 0) {
      void refetch();
    }
  }, [queue.length, isFetching, refetch, feed.data]);

  const swipe = useMutation({
    mutationFn: ({ card, action }: { card: DiscoverCard; action: SwipeAction }) => discoveryApi.swipe(card.userId, action),
    onSuccess: (res) => {
      if (res.matched && res.match) {
        setMatch(res.match);
        void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
      }
    },
    onError: (error, { card }) => {
      if (isApiError(error, "already_swiped") || isApiError(error, "profile_not_found")) return;
      handled.current.delete(card.userId);
      setQueue((q) => [card, ...q]);
      showToast(errorMessage(error), "error");
    }
  });

  const onSwipe = useCallback(
    (card: DiscoverCard, action: SwipeAction) => {
      handled.current.add(card.userId);
      setQueue((q) => q.filter((c) => c.userId !== card.userId));
      setLastPass(action === "pass" ? card : null);
      swipe.mutate({ card, action });
    },
    [swipe]
  );

  const undo = useMutation({
    mutationFn: discoveryApi.undo,
    onSuccess: ({ profile }) => {
      handled.current.delete(profile.userId);
      const restored: DiscoverCard = { ...profile, sharedInterests: lastPass?.userId === profile.userId ? lastPass.sharedInterests : 0 };
      setQueue((q) => [restored, ...q.filter((c) => c.userId !== profile.userId)]);
      setLastPass(null);
    },
    onError: (error) => {
      setLastPass(null);
      showToast(errorMessage(error), "info");
    }
  });

  const top = queue[0];
  const openDetails = (card: DiscoverCard) => navigation.navigate("ProfileDetail", { userId: card.userId });

  let content: React.ReactNode;
  if (feed.isLoading && queue.length === 0) {
    content = <LoadingState label="On cherche des personnes près de toi…" />;
  } else if (feed.isError && queue.length === 0) {
    content = <ErrorState message={errorMessage(feed.error)} onRetry={() => void feed.refetch()} />;
  } else if (!top) {
    content = (
      <EmptyState
        testID="discover-empty"
        icon="moon-outline"
        title="Tu as fait le tour pour l'instant"
        message="Reviens plus tard, ou élargis ta distance et ta tranche d'âge (icône en haut à droite) pour découvrir plus de monde."
        actionLabel={feed.isFetching ? "Recherche…" : "Actualiser"}
        onAction={() => void feed.refetch()}
      />
    );
  } else {
    content = (
      <>
        <View style={styles.deck}>
          <SwipeDeck ref={deck} cards={queue} onSwipe={onSwipe} onOpenDetails={openDetails} />
        </View>
        <View style={styles.actions}>
          <IconButton
            icon="arrow-undo"
            label="Revoir le dernier profil passé"
            testID="discover-undo"
            size={48}
            color={colors.gold}
            disabled={!lastPass || undo.isPending}
            onPress={() => undo.mutate()}
          />
          <IconButton
            icon="close"
            label={`Passer ${top.firstName}`}
            testID="discover-pass"
            size={66}
            color={colors.pass}
            onPress={() => deck.current?.swipe("pass")}
          />
          <IconButton
            icon="heart"
            label={`Aimer ${top.firstName}`}
            testID="discover-like"
            size={66}
            color={colors.accentText}
            background={colors.accent}
            borderColor={colors.accent}
            onPress={() => deck.current?.swipe("like")}
          />
          <IconButton
            icon="person-outline"
            label={`Voir le profil complet de ${top.firstName}`}
            testID="discover-details"
            size={48}
            onPress={() => openDetails(top)}
          />
        </View>
      </>
    );
  }

  return (
    <Screen testID="discover-screen">
      <View style={styles.header}>
        <Text variant="title">Découvrir</Text>
        <IconButton icon="options-outline" label="Préférences" onPress={() => navigation.navigate("Preferences")} />
      </View>
      {content}
      <MatchCelebration
        match={match}
        myName={me.data?.firstName ?? ""}
        myPhotoUrl={me.data?.photos[0]?.url}
        onClose={() => setMatch(null)}
        onMessage={(m) => {
          setMatch(null);
          navigation.navigate("Conversation", { conversationId: m.conversationId });
        }}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  header: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    marginBottom: spacing.md
  },
  deck: {
    flex: 1
  },
  actions: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "center",
    gap: spacing.lg,
    paddingVertical: spacing.lg
  }
});
