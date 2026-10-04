import { useFocusEffect, useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useQueryClient } from "@tanstack/react-query";
import React, { useCallback, useEffect, useRef, useState } from "react";
import { StyleSheet, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import * as discoverApi from "../../api/discover";
import { queryKeys } from "../../api/queryKeys";
import type { MatchSummary, PublicProfile, SwipeAction } from "../../api/types";
import { useOwnProfile } from "../../hooks/useProfile";
import type { MainStackParamList } from "../../navigation/types";
import { getDiscoverEpoch, takeDeckIntent, usePendingDeckIntent } from "../../store/deckIntent";
import { spacing, useColors } from "../../theme";
import { getErrorMessage, isApiError } from "../../api/client";
import { EmptyState, ErrorState, Skeleton, Text, toast } from "../../ui";
import { MatchModal } from "./MatchModal";
import { SwipeDeck, type SwipeDeckHandle } from "./SwipeDeck";
import { useDiscoverDeck } from "./useDiscoverDeck";

type Nav = NativeStackNavigationProp<MainStackParamList>;

export function DiscoverScreen() {
  const navigation = useNavigation<Nav>();
  const client = useQueryClient();
  const colors = useColors();
  const profileQuery = useOwnProfile();
  const { deck, dispatch, loading, error, refresh } = useDiscoverDeck(true);
  const [match, setMatch] = useState<MatchSummary | null>(null);
  const deckRef = useRef<SwipeDeckHandle>(null);
  const seenEpoch = useRef(getDiscoverEpoch());
  const intent = usePendingDeckIntent();
  const cardsRef = useRef(deck.cards);
  cardsRef.current = deck.cards;

  const commitSwipe = useCallback(
    async (profile: PublicProfile, action: SwipeAction) => {
      dispatch({ type: "swipe", userId: profile.userId }); // optimistic
      try {
        const result = await discoverApi.swipe(profile.userId, action);
        if (result.matched && result.match) {
          setMatch(result.match);
          void client.invalidateQueries({ queryKey: queryKeys.matchesAll });
          void client.invalidateQueries({ queryKey: queryKeys.notificationsAll });
        }
      } catch (e) {
        dispatch({ type: "rollback", profile }); // put the card back on top
        toast.error(getErrorMessage(e, "We couldn't save that swipe. Please try again."));
      }
    },
    [client, dispatch]
  );

  // Actions requested by other screens (like/pass from the profile detail, block/report removals).
  useFocusEffect(
    useCallback(() => {
      if (getDiscoverEpoch() !== seenEpoch.current) {
        seenEpoch.current = getDiscoverEpoch();
        refresh();
      }
      const pending = takeDeckIntent();
      if (!pending) return;
      const card = cardsRef.current.find((c) => c.userId === pending.userId);
      if (!card) return;
      if (pending.kind === "remove") dispatch({ type: "remove", userId: pending.userId });
      else void commitSwipe(card, pending.kind);
    }, [commitSwipe, dispatch, refresh])
  );
  // Same, for intents that arrive while this tab is already focused (e.g. block from a modal).
  useEffect(() => {
    if (!intent || intent.kind !== "remove") return;
    takeDeckIntent();
    dispatch({ type: "remove", userId: intent.userId });
  }, [intent, dispatch]);

  const incomplete = isApiError(error) && error.code === "profile_incomplete";
  const myPhoto = profileQuery.data?.photos[0] ?? null;

  function openProfile(profile: PublicProfile) {
    navigation.navigate("ProfileDetail", { userId: profile.userId, source: "discover" });
  }

  let content: React.ReactNode;
  if (incomplete) {
    content = (
      <EmptyState
        actionLabel="Finish my profile"
        icon="person-circle-outline"
        message="Add a photo and the basics to start meeting people."
        onAction={() => navigation.navigate("EditProfile")}
        title="Your profile isn't ready yet"
      />
    );
  } else if (deck.cards.length > 0) {
    content = <SwipeDeck cards={deck.cards} onOpenProfile={openProfile} onSwipe={commitSwipe} ref={deckRef} />;
  } else if (error) {
    content = <ErrorState error={error} onRetry={refresh} />;
  } else if (loading || !deck.exhausted) {
    content = (
      <View accessibilityLabel="Loading profiles" accessibilityRole="progressbar" style={styles.skeleton}>
        <Skeleton height={500} radius={28} style={{ flex: 1 }} />
      </View>
    );
  } else {
    content = (
      <EmptyState
        actionLabel="Refresh"
        icon="sparkles-outline"
        message="You've seen everyone for now. Widen your preferences or check back soon."
        onAction={refresh}
        onSecondary={() => navigation.navigate("Preferences")}
        secondaryLabel="Edit preferences"
        title="No more profiles nearby"
      />
    );
  }

  return (
    <SafeAreaView edges={["top", "left", "right"]} style={[styles.screen, { backgroundColor: colors.background }]}>
      <View style={styles.header}>
        <Text variant="heading">Discover</Text>
      </View>
      <View style={styles.body}>{content}</View>
      <MatchModal
        match={match}
        myName={profileQuery.data?.displayName ?? ""}
        myPhoto={myPhoto}
        onClose={() => setMatch(null)}
        onMessage={(m) => {
          setMatch(null);
          navigation.navigate("Chat", { match: m });
        }}
      />
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  screen: { flex: 1 },
  header: { paddingHorizontal: spacing.md, paddingVertical: spacing.xs },
  body: { flex: 1, paddingHorizontal: spacing.md, paddingBottom: spacing.xs },
  skeleton: { flex: 1 }
});
