import React, { useCallback, useEffect, useMemo, useState } from "react";
import { View } from "react-native";
import { useFocusEffect, useNavigation } from "@react-navigation/native";
import { useQuery } from "@tanstack/react-query";
import { ApiError } from "../../api/client";
import { matchingApi } from "../../api/endpoints";
import type { ProfileCard, SwipeResponse } from "../../api/types";
import { errorMessage } from "../../lib/format";
import { keys } from "../../providers/queryClient";
import { useTheme } from "../../theme";
import { EmptyState, ErrorState, Header, IconButton, LoadingState, Screen, Text } from "../../ui";
import { NotificationsBell } from "../notifications/NotificationsBell";
import { useMyProfile } from "../profile/queries";
import { MatchModal } from "./MatchModal";
import { SwipeDeck } from "./SwipeDeck";
import { useDecidedProfiles } from "./swipeStore";
import { useSwipe } from "./useSwipe";

const BATCH = 10;

export function DiscoverScreen() {
  const navigation = useNavigation();
  const { spacing, colors } = useTheme();
  const profile = useMyProfile();
  const decided = useDecidedProfiles();
  const [queue, setQueue] = useState<ProfileCard[]>([]);
  const [match, setMatch] = useState<{ user: NonNullable<SwipeResponse["matchedUser"]>; conversationId: string } | null>(null);

  const discovery = useQuery({ queryKey: keys.discovery, queryFn: () => matchingApi.discover(BATCH), staleTime: 0 });

  // Merge each fetched batch into a local queue, skipping profiles already shown.
  useEffect(() => {
    if (!discovery.data) {
      return;
    }
    setQueue((current) => {
      const known = new Set(current.map((c) => c.userId));
      return [...current, ...discovery.data.filter((c) => !known.has(c.userId))];
    });
  }, [discovery.data]);

  const visible = useMemo(() => queue.filter((c) => !decided.has(c.userId)), [queue, decided]);

  // Prefetch the next batch before the deck runs out. A partial batch means
  // nobody else matches right now, so we stop instead of polling.
  const lastBatchFull = (discovery.data?.length ?? 0) >= BATCH;
  const { isFetching, refetch } = discovery;
  useEffect(() => {
    if (visible.length < 3 && lastBatchFull && !isFetching) {
      void refetch();
    }
  }, [visible.length, lastBatchFull, isFetching, refetch]);

  // Coming back to an empty deck: look again (new members, changed preferences).
  useFocusEffect(
    useCallback(() => {
      if (visible.length === 0) {
        void refetch();
      }
    }, [visible.length, refetch])
  );

  const swipe = useSwipe((result) => {
    if (result.matchedUser && result.conversationId) {
      setMatch({ user: result.matchedUser, conversationId: result.conversationId });
    }
  });

  const header = (
    <Header
      large
      right={
        <>
          <NotificationsBell />
          <IconButton icon="options-outline" label="Préférences de rencontre" onPress={() => navigation.navigate("Preferences")} />
        </>
      }
      title="Découvrir"
    />
  );

  let body: React.ReactNode;
  if (discovery.isLoading) {
    body = <LoadingState label="On cherche des profils pour toi…" />;
  } else if (discovery.isError && visible.length === 0) {
    const incomplete = discovery.error instanceof ApiError && discovery.error.code === "profile_incomplete";
    body = incomplete ? (
      <EmptyState
        actionLabel="Compléter mon profil"
        icon="person-circle-outline"
        message={errorMessage(discovery.error)}
        onAction={() => navigation.navigate("EditPhotos")}
        title="Ton profil n’est pas complet"
      />
    ) : (
      <ErrorState message={errorMessage(discovery.error)} onRetry={() => void discovery.refetch()} />
    );
  } else if (visible.length === 0) {
    body = discovery.isFetching ? (
      <LoadingState />
    ) : (
      <EmptyState
        actionLabel="Élargir mes préférences"
        icon="sparkles-outline"
        message="Tu as vu tous les profils qui correspondent à tes critères. Reviens plus tard ou élargis ta recherche."
        onAction={() => navigation.navigate("Preferences")}
        title="Plus personne pour l’instant"
      />
    );
  } else {
    body = (
      <SwipeDeck
        cards={visible}
        onDecision={(card, action) => swipe.mutate({ userId: card.userId, action })}
        onOpen={(card) => navigation.navigate("ProfileDetail", { userId: card.userId, fromDiscovery: true })}
      />
    );
  }

  return (
    <Screen>
      {header}
      {profile.data && !profile.data.discoverable ? (
        <View style={{ marginHorizontal: spacing.lg, marginBottom: spacing.sm, padding: spacing.sm, borderRadius: 12, backgroundColor: colors.accentSoft }}>
          <Text tone="accent" variant="caption">
            Ton profil est en pause : tu peux découvrir, mais personne ne te voit. Réactive-le dans les réglages.
          </Text>
        </View>
      ) : null}
      <View style={{ flex: 1, paddingBottom: spacing.xs }}>{body}</View>
      <MatchModal
        match={match}
        myName={profile.data?.firstName ?? ""}
        myPhotoUrl={profile.data?.photos[0]?.url}
        onClose={() => setMatch(null)}
        onMessage={(conversationId) => {
          setMatch(null);
          navigation.navigate("Chat", { conversationId });
        }}
      />
    </Screen>
  );
}
