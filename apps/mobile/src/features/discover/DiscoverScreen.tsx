import React, { useCallback, useEffect, useRef, useState } from "react";
import { StyleSheet, View } from "react-native";
import { errorMessage } from "../../api/client";
import type { MatchSummary, ProfileCard, SwipeAction } from "../../api/types";
import { useMyProfile } from "../../hooks/queries";
import { markMatchAnnounced } from "../../hooks/useRealtime";
import { spacing } from "../../theme/palette";
import { EmptyState, ErrorState, Header, IconButton, Screen, Skeleton, toast } from "../../ui";
import type { TabScreenProps } from "../../navigation/types";
import { MatchModal } from "./MatchModal";
import { SwipeDeck } from "./SwipeDeck";
import { useDeck } from "./useDeck";

export function DiscoverScreen({ navigation }: TabScreenProps<"Discover">) {
  const me = useMyProfile();
  const [match, setMatch] = useState<MatchSummary | null>(null);
  const onMatch = useCallback((m: MatchSummary) => {
    markMatchAnnounced(m.id);
    setMatch(m);
  }, []);
  const deck = useDeck(onMatch);

  // Preferences changed (e.g. from the filters screen): rebuild the deck.
  const prefsKey = JSON.stringify(me.data?.preferences ?? null) + String(me.data?.hasLocation);
  const lastPrefs = useRef(prefsKey);
  useEffect(() => {
    if (lastPrefs.current !== prefsKey) {
      lastPrefs.current = prefsKey;
      deck.reload();
    }
  }, [prefsKey, deck]);

  const handleSwipe = useCallback(
    (card: ProfileCard, action: SwipeAction) => {
      deck.swipe(card, action).catch((e) => toast(errorMessage(e), "error"));
    },
    [deck]
  );

  let body: React.ReactNode;
  if (deck.loading && deck.queue.length === 0) {
    body = (
      <View style={styles.skeleton}>
        <Skeleton height={0} style={styles.flex} radius={28} />
        <Skeleton height={64} width={220} radius={32} style={styles.center} />
      </View>
    );
  } else if (deck.error && deck.queue.length === 0) {
    body = <ErrorState message={errorMessage(deck.error)} onRetry={deck.reload} />;
  } else if (deck.queue.length === 0) {
    body = (
      <EmptyState
        icon="sparkles-outline"
        title="Tu as fait le tour pour l'instant"
        message="De nouvelles personnes s'inscrivent chaque jour. Élargis tes critères ou repasse un peu plus tard."
        actionLabel="Ajuster mes critères"
        onAction={() => navigation.navigate("Preferences")}
        testID="discover-empty"
      />
    );
  } else {
    body = (
      <SwipeDeck cards={deck.queue} onSwipe={handleSwipe} onOpenDetails={(card) => navigation.navigate("ProfileDetail", { userId: card.userId })} />
    );
  }

  return (
    <Screen testID="discover-screen">
      <Header
        title="Découvrir"
        large
        right={
          <View style={styles.headerActions}>
            <IconButton icon="refresh" accessibilityLabel="Actualiser" onPress={deck.reload} />
            <IconButton icon="options-outline" accessibilityLabel="Critères de recherche" onPress={() => navigation.navigate("Preferences")} />
          </View>
        }
      />
      <View style={styles.body}>{body}</View>
      <MatchModal
        match={match}
        myName={me.data?.firstName ?? ""}
        myPhotoUrl={me.data?.photos[0]?.url}
        onClose={() => setMatch(null)}
        onMessage={(m) => {
          setMatch(null);
          navigation.navigate("Chat", { matchId: m.id });
        }}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  body: { flex: 1, paddingBottom: spacing.sm, maxWidth: 560, width: "100%", alignSelf: "center" },
  headerActions: { flexDirection: "row" },
  skeleton: { flex: 1, gap: spacing.md },
  center: { alignSelf: "center" }
});
