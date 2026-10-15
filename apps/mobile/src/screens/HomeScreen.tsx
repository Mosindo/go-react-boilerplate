import React, { useCallback, useEffect, useRef, useState } from "react";
import { Pressable, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useQueryClient } from "@tanstack/react-query";
import { mediaUrl } from "../api/client";
import { discover, swipe } from "../api/platform";
import type { Card, MatchInfo, SwipeAction } from "../api/types";
import { MatchOverlay } from "../components/MatchOverlay";
import { SwipeDeck, type DeckHandle } from "../components/SwipeDeck";
import { errorCodeOf, errorMessage } from "../i18n/errors";
import { APP_NAME } from "../i18n/fr";
import { queryKeys, useMyProfile } from "../hooks/queries";
import { useRealtime } from "../hooks/useRealtime";
import { DECK_BATCH_SIZE, excludeIds, mergeCards, shouldRefill } from "../lib/deck";
import { onProfileRemoved } from "../lib/removedProfiles";
import type { RootStackParamList } from "../navigation/types";
import { EmptyView, ErrorView, LoadingView, showError } from "../shared/feedback";
import { ScreenContainer } from "../shared/layout";
import { Text, controls, spacing } from "../shared/ui";
import { useTheme, elevation } from "../theme";

type MatchState = { match: MatchInfo; card: Card } | null;

export default function HomeScreen() {
  const palette = useTheme();
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>();
  const queryClient = useQueryClient();
  const profile = useMyProfile();
  const { markMatchCelebrated } = useRealtime();

  const [queue, setQueue] = useState<Card[]>([]);
  const [loading, setLoading] = useState(false);
  const [exhausted, setExhausted] = useState(false);
  const [loadError, setLoadError] = useState<unknown>(null);
  const [initialised, setInitialised] = useState(false);
  const [matchState, setMatchState] = useState<MatchState>(null);

  const deckRef = useRef<DeckHandle | null>(null);
  const decided = useRef(new Set<string>());
  const queueRef = useRef<Card[]>([]);
  queueRef.current = queue;
  const loadingRef = useRef(false);

  const load = useCallback(async () => {
    if (loadingRef.current) {
      return;
    }
    loadingRef.current = true;
    setLoading(true);
    setLoadError(null);
    try {
      const batch = await discover(excludeIds(queueRef.current), DECK_BATCH_SIZE);
      setQueue(mergeCards(queueRef.current, batch, decided.current));
      // A short batch means the server has nothing more to offer right now.
      if (batch.length < DECK_BATCH_SIZE) {
        setExhausted(true);
      }
    } catch (error) {
      setLoadError(error);
    } finally {
      loadingRef.current = false;
      setLoading(false);
      setInitialised(true);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  // A person blocked or reported from their full profile must leave the deck at once.
  useEffect(
    () =>
      onProfileRemoved((userId) => {
        decided.current.add(userId);
        setQueue((current) => current.filter((c) => c.userId !== userId));
      }),
    []
  );

  useEffect(() => {
    if (initialised && !loadError && shouldRefill(queue.length, exhausted, loading)) {
      void load();
    }
  }, [exhausted, initialised, load, loadError, loading, queue.length]);

  const onDecide = useCallback(
    async (card: Card, action: SwipeAction) => {
      decided.current.add(card.userId);
      setQueue((current) => current.filter((c) => c.userId !== card.userId));
      try {
        const result = await swipe(card.userId, action);
        if (result.matched && result.match) {
          markMatchCelebrated(result.match.id);
          setMatchState({ match: result.match, card });
          void queryClient.invalidateQueries({ queryKey: queryKeys.chats });
        }
      } catch (error) {
        if (errorCodeOf(error) === "already_swiped" || errorCodeOf(error) === "profile_not_found") {
          return; // Already handled elsewhere, or no longer available: the card is rightly gone.
        }
        // The decision did not reach the server: bring the card back so nothing is silently lost.
        decided.current.delete(card.userId);
        setQueue((current) => [card, ...current.filter((c) => c.userId !== card.userId)]);
        showError(errorMessage(error, "Votre choix n'a pas pu être enregistré. Réessayez."));
      }
    },
    [markMatchCelebrated, queryClient]
  );

  const openProfile = useCallback(
    (card: Card) => {
      queryClient.setQueryData(queryKeys.profile(card.userId), card);
      navigation.navigate("ProfileDetail", { userId: card.userId });
    },
    [navigation, queryClient]
  );

  const refresh = () => {
    setExhausted(false);
    void load();
  };

  const myPhoto = mediaUrl(profile.data?.profile?.photos[0]?.thumbUrl);
  const top = queue[0];

  const actionButton = (action: SwipeAction | "info", icon: React.ComponentProps<typeof Ionicons>["name"], label: string, size: number) => {
    const color = action === "like" ? palette.onPrimary : action === "pass" ? palette.pass : palette.brand;
    const background = action === "like" ? palette.primary : palette.surface;
    return (
      <Pressable
        accessibilityLabel={label}
        accessibilityRole="button"
        disabled={!top}
        onPress={() => {
          if (action === "info") {
            if (top) openProfile(top);
          } else {
            deckRef.current?.swipe(action);
          }
        }}
        style={({ pressed }) => [
          {
            width: size,
            height: size,
            borderRadius: size / 2,
            backgroundColor: background,
            borderWidth: action === "like" ? 0 : 1.5,
            borderColor: palette.border,
            alignItems: "center",
            justifyContent: "center",
            opacity: !top ? 0.4 : pressed ? 0.8 : 1
          },
          elevation(palette)
        ]}
        testID={`discover-${action}`}
      >
        <Ionicons color={color} name={icon} size={size * 0.46} />
      </Pressable>
    );
  };

  let content: React.ReactNode;
  if (profile.data && !profile.data.onboarding.complete) {
    content = (
      <EmptyView
        message="Ajoutez vos informations, vos préférences et une photo pour découvrir des profils."
        title="Terminez votre profil"
      />
    );
  } else if (loadError && queue.length === 0) {
    content = <ErrorView error={loadError} onRetry={refresh} />;
  } else if (!initialised) {
    content = <LoadingView fullScreen label="Recherche de profils…" />;
  } else if (!top) {
    content = (
      <EmptyView
        actionLabel="Actualiser"
        icon={<Ionicons color={palette.textSubtle} name="compass-outline" size={56} />}
        message="Vous avez vu tout le monde pour l'instant. Revenez plus tard ou élargissez vos critères de recherche."
        onAction={refresh}
        style={{ flex: 1 }}
        testID="discover-empty"
        title="C'est tout pour le moment"
      />
    );
  } else {
    content = (
      <>
        <View style={{ flex: 1, marginBottom: spacing.lg }}>
          <SwipeDeck cards={queue} deckRef={deckRef} onDecide={(card, action) => void onDecide(card, action)} onOpenProfile={openProfile} />
        </View>
        <View style={{ flexDirection: "row", justifyContent: "center", alignItems: "center", gap: spacing.xl, paddingBottom: spacing.lg }}>
          {actionButton("pass", "close", "Passer", controls.avatar.lg)}
          {actionButton("info", "person-outline", "Voir le profil complet", controls.avatar.md)}
          {actionButton("like", "heart", "J'aime", controls.avatar.lg)}
        </View>
      </>
    );
  }

  return (
    <ScreenContainer testID="home-screen">
      <View style={{ flexDirection: "row", alignItems: "center", justifyContent: "space-between", minHeight: 56 }}>
        <Text accessibilityRole="header" tone="brand" variant="heading" weight="bold">
          {APP_NAME}
        </Text>
        <Pressable
          accessibilityLabel="Régler mes préférences de rencontre"
          accessibilityRole="button"
          hitSlop={8}
          onPress={() => navigation.navigate("Preferences")}
          style={{ width: 44, height: 44, alignItems: "center", justifyContent: "center" }}
        >
          <Ionicons color={palette.text} name="options-outline" size={26} />
        </Pressable>
      </View>
      {content}

      <MatchOverlay
        firstName={matchState?.match.firstName ?? ""}
        mePhotoUri={myPhoto}
        myName={profile.data?.profile?.firstName}
        onContinue={() => setMatchState(null)}
        onMessage={() => {
          const current = matchState;
          setMatchState(null);
          if (current) {
            navigation.navigate("Thread", {
              conversationId: current.match.conversationId,
              peerId: current.match.userId,
              peerName: current.match.firstName
            });
          }
        }}
        theirPhotoUri={mediaUrl(matchState?.card.photos[0]?.thumbUrl)}
        visible={matchState !== null}
      />
    </ScreenContainer>
  );
}
