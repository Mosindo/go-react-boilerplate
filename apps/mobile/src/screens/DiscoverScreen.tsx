import React, { useCallback, useRef } from "react";
import { StyleSheet, View } from "react-native";
import { useFocusEffect, useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import type { PublicProfile } from "../api";
import { useFeedback } from "../components/Feedback";
import { Screen } from "../components/Screen";
import { EmptyState, ErrorState, LoadingState } from "../components/States";
import { SwipeDeck } from "../components/SwipeDeck";
import { Text } from "../components/Text";
import { useDiscoverDeck } from "../hooks/useDiscoverDeck";
import type { RootStackParamList } from "../navigation/types";
import { useRealtime } from "../realtime/RealtimeProvider";
import { spacing } from "../theme";

export function DiscoverScreen() {
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>();
  const { toast } = useFeedback();
  const { showMatch } = useRealtime();
  const onNotice = useCallback((msg: string) => toast(msg, "error"), [toast]);
  const deck = useDiscoverDeck(true, showMatch, onNotice);

  // New people may have joined while the user was elsewhere: retry when coming back to an empty deck.
  const latest = useRef({ state: deck.state, reload: deck.reload });
  latest.current = { state: deck.state, reload: deck.reload };
  useFocusEffect(
    useCallback(() => {
      const { state, reload } = latest.current;
      if (state === "empty" || state === "incomplete") reload();
    }, [])
  );

  const openProfile = (profile: PublicProfile) =>
    navigation.navigate("ProfileDetail", { profile, canSwipe: true });

  let body: React.ReactNode;
  if (deck.state === "loading") body = <LoadingState label="Recherche de profils…" />;
  else if (deck.state === "error")
    body = <ErrorState message={deck.error ?? undefined} onRetry={deck.reload} />;
  else if (deck.state === "incomplete")
    body = (
      <EmptyState
        title="Complétez votre profil"
        message="Ajoutez vos préférences, une photo et votre position pour découvrir des profils."
        actionLabel="Compléter"
        onAction={() => navigation.navigate("EditProfile")}
      />
    );
  else if (deck.state === "empty" || deck.queue.length === 0)
    body = (
      <EmptyState
        emoji="🌿"
        title="Vous avez tout vu pour l'instant"
        message="Élargissez votre distance ou votre tranche d'âge, ou revenez plus tard : de nouvelles personnes arrivent chaque jour."
        actionLabel="Actualiser"
        onAction={deck.reload}
      />
    );
  else
    body = <SwipeDeck profiles={deck.queue} onSwipe={(p, a) => void deck.swipe(p, a)} onInfo={openProfile} />;

  return (
    <Screen>
      <View style={styles.header}>
        <Text variant="title">Découvrir</Text>
      </View>
      <View style={styles.body}>{body}</View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  header: { paddingBottom: spacing.sm },
  body: { flex: 1 }
});
