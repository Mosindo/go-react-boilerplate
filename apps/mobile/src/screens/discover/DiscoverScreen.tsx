import React, { useEffect, useRef, useState } from "react";
import { Platform, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import * as Haptics from "expo-haptics";
import { ApiError } from "../../api/client";
import { discoveryApi } from "../../api/social";
import type { Match, ProfileCard, SwipeAction } from "../../api/types";
import { MatchCelebration } from "../../components/MatchCelebration";
import { SwipeDeck, type SwipeDeckHandle } from "../../components/SwipeDeck";
import { onDeckDecision } from "../../hooks/deckEvents";
import { keys, useMyProfile } from "../../hooks/queries";
import type { AppStackParamList } from "../../navigation/types";
import { colors, spacing } from "../../theme/tokens";
import { EmptyState, ErrorState, IconButton, LoadingState, Screen, Text, showToast } from "../../ui";
import { errorMessage } from "../../utils/errors";

type Nav = NativeStackNavigationProp<AppStackParamList>;

const REFILL_AT = 3;

export default function DiscoverScreen() {
  const navigation = useNavigation<Nav>();
  const qc = useQueryClient();
  const me = useMyProfile();
  const deckRef = useRef<SwipeDeckHandle>(null);
  const [queue, setQueue] = useState<ProfileCard[]>([]);
  const decided = useRef(new Set<string>());
  const [celebrate, setCelebrate] = useState<Match | null>(null);

  const deck = useQuery({ queryKey: keys.deck, queryFn: () => discoveryApi.deck(10), staleTime: 60_000 });

  useEffect(() => {
    if (!deck.data) {
      // The deck query was reset (e.g. new preferences): start over.
      setQueue([]);
      return;
    }
    setQueue((current) => {
      const present = new Set(current.map((c) => c.userId));
      const fresh = deck.data.profiles.filter((p) => !present.has(p.userId) && !decided.current.has(p.userId));
      return fresh.length ? [...current, ...fresh] : current;
    });
  }, [deck.data]);

  // A profile answered from the detail screen leaves the deck too.
  useEffect(
    () =>
      onDeckDecision((userId) => {
        decided.current.add(userId);
        setQueue((q) => q.filter((c) => c.userId !== userId));
      }),
    []
  );

  const swipe = useMutation({
    mutationFn: (v: { card: ProfileCard; action: SwipeAction }) => discoveryApi.swipe(v.card.userId, v.action),
    onSuccess: (res) => {
      if (res.matched && res.match) {
        if (Platform.OS !== "web") {
          void Haptics.notificationAsync(Haptics.NotificationFeedbackType.Success);
        }
        setCelebrate(res.match);
        void qc.invalidateQueries({ queryKey: keys.matches });
      }
    },
    onError: (err) => {
      if (!(err instanceof ApiError && (err.code === "already_swiped" || err.code === "not_found"))) {
        showToast(errorMessage(err), "error");
      }
    }
  });

  const onSwipe = (card: ProfileCard, action: SwipeAction) => {
    decided.current.add(card.userId);
    setQueue((q) => q.filter((c) => c.userId !== card.userId));
    swipe.mutate({ card, action });
    if (queue.length - 1 <= REFILL_AT && !deck.isFetching) {
      void deck.refetch();
    }
  };

  const decideFromButton = (action: SwipeAction) => {
    if (action === "like" && Platform.OS !== "web") {
      void Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light);
    }
    deckRef.current?.swipe(action);
  };

  const myPhoto = me.data?.profile?.photos[0]?.url;
  const incomplete = deck.error instanceof ApiError && deck.error.code === "profile_incomplete";

  let body: React.ReactNode;
  if (queue.length > 0) {
    body = (
      <>
        <View style={styles.deck}>
          <SwipeDeck
            ref={deckRef}
            cards={queue}
            onSwipe={onSwipe}
            onOpen={(card) => navigation.navigate("ProfileDetail", { userId: card.userId, card, fromDeck: true })}
          />
        </View>
        <View style={styles.actions}>
          <IconButton
            icon="close"
            label={`Passer ${queue[0].firstName}`}
            size={64}
            color={colors.textMuted}
            onPress={() => decideFromButton("pass")}
            testID="deck-pass"
          />
          <IconButton
            icon="information-circle"
            label={`Voir le profil de ${queue[0].firstName}`}
            size={48}
            color={colors.plum}
            onPress={() => navigation.navigate("ProfileDetail", { userId: queue[0].userId, card: queue[0], fromDeck: true })}
          />
          <IconButton
            icon="heart"
            label={`Aimer ${queue[0].firstName}`}
            size={64}
            color={colors.textOnAccent}
            background={colors.accent}
            onPress={() => decideFromButton("like")}
            testID="deck-like"
          />
        </View>
      </>
    );
  } else if (deck.isLoading || (deck.isFetching && !deck.data)) {
    body = <LoadingState label="On cherche des profils pour toi…" />;
  } else if (incomplete) {
    body = (
      <EmptyState
        icon="images-outline"
        title="Ton profil est presque prêt"
        message="Ajoute au moins une photo pour découvrir d'autres personnes et être visible."
        actionLabel="Ajouter une photo"
        onAction={() => navigation.navigate("EditPhotos")}
      />
    );
  } else if (deck.error) {
    body = <ErrorState message={errorMessage(deck.error)} onRetry={() => deck.refetch()} />;
  } else {
    body = (
      <EmptyState
        icon="sparkles-outline"
        title="Tu as fait le tour pour l'instant"
        message={
          deck.data?.reason === "no_more_profiles_paused"
            ? "Ton profil est en pause : les autres ne te voient pas. Réactive-le dans les réglages."
            : "Reviens un peu plus tard, ou élargis tes critères de distance ou d'âge."
        }
        actionLabel="Modifier mes critères"
        onAction={() => navigation.navigate("Preferences")}
        testID="deck-empty"
      />
    );
  }

  return (
    <Screen testID="discover-screen">
      <View style={styles.header}>
        <Text variant="title" style={styles.brand}>
          Lueur
        </Text>
        <IconButton icon="options-outline" label="Critères de recherche" onPress={() => navigation.navigate("Preferences")} />
      </View>
      {body}
      <MatchCelebration
        match={celebrate}
        myPhotoUrl={myPhoto}
        myName={me.data?.profile?.firstName ?? "Moi"}
        onClose={() => setCelebrate(null)}
        onMessage={(m) => {
          // Let the modal finish closing before pushing the conversation, so
          // its fade-out overlay never lingers above the next screen.
          setCelebrate(null);
          setTimeout(() => navigation.navigate("Conversation", { matchId: m.id }), 280);
        }}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  header: { flexDirection: "row", alignItems: "center", justifyContent: "space-between", marginBottom: spacing.md },
  brand: { color: colors.accent },
  deck: { flex: 1 },
  actions: { flexDirection: "row", alignItems: "center", justifyContent: "center", gap: spacing.xxl, paddingVertical: spacing.lg }
});
