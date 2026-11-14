import React, { useCallback, useEffect, useRef, useState } from "react";
import { StyleSheet, View } from "react-native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useFocusEffect, useNavigation } from "@react-navigation/native";
import { useQueryClient } from "@tanstack/react-query";
import { errorMessage, isApiError } from "../../api/client";
import { matchingApi } from "../../api/endpoints";
import type { Match, PublicProfile, SwipeAction } from "../../api/types";
import type { MainStackParamList } from "../../navigation/types";
import { keys } from "../../state/queryClient";
import { useMyProfile } from "../../state/queries";
import { spacing } from "../../theme/palette";
import { useTheme } from "../../theme/ThemeProvider";
import { EmptyState, ErrorState, IconButton, LoadingState, Screen, Text, useToast } from "../../ui";
import { deckEvents } from "./deckEvents";
import { MatchModal } from "./MatchModal";
import { SwipeDeck, type SwipeDeckHandle } from "./SwipeDeck";

type Nav = NativeStackNavigationProp<MainStackParamList>;

const BATCH = 10;
const REFILL_AT = 3;

export function DiscoverScreen() {
  const navigation = useNavigation<Nav>();
  const { colors } = useTheme();
  const toast = useToast();
  const qc = useQueryClient();
  const { data: me } = useMyProfile();
  const deckRef = useRef<SwipeDeckHandle>(null);
  const [deck, setDeck] = useState<PublicProfile[]>([]);
  const [status, setStatus] = useState<"loading" | "ready" | "error">("loading");
  const [error, setError] = useState<string | null>(null);
  const [exhausted, setExhausted] = useState(false);
  const [match, setMatch] = useState<Match | null>(null);
  const seen = useRef(new Set<string>());
  const fetching = useRef(false);

  const load = useCallback(async (reset = false) => {
    if (fetching.current) return;
    fetching.current = true;
    if (reset) {
      seen.current.clear();
      setStatus("loading");
    }
    try {
      const batch = await matchingApi.discover(BATCH);
      const fresh = batch.filter((p) => !seen.current.has(p.userId));
      fresh.forEach((p) => seen.current.add(p.userId));
      setDeck((d) => (reset ? fresh : [...d, ...fresh]));
      setExhausted(fresh.length === 0 || batch.length < BATCH);
      setStatus("ready");
    } catch (e) {
      setError(errorMessage(e));
      setStatus((s) => (s === "loading" ? "error" : s));
    } finally {
      fetching.current = false;
    }
  }, []);

  useEffect(() => {
    void load(true);
  }, [load]);

  // Preferences or location changed elsewhere → rebuild the deck.
  useFocusEffect(
    useCallback(() => {
      if (deckEvents.consumeStale()) void load(true);
    }, [load])
  );

  useEffect(() => {
    if (status === "ready" && !exhausted && deck.length <= REFILL_AT) void load();
  }, [deck.length, exhausted, load, status]);

  const decide = useCallback(
    async (profile: PublicProfile, action: SwipeAction) => {
      setDeck((d) => d.filter((p) => p.userId !== profile.userId));
      try {
        const result = await matchingApi.swipe(profile.userId, action);
        if (result.matched && result.match) {
          setMatch(result.match);
          void qc.invalidateQueries({ queryKey: keys.conversations });
        }
      } catch (e) {
        if (!isApiError(e, "already_swiped") && !isApiError(e, "profile_not_found")) {
          toast({ title: "Action non enregistrée", body: errorMessage(e), tone: "error" });
          seen.current.delete(profile.userId);
        }
      }
    },
    [qc, toast]
  );

  useEffect(
    () =>
      deckEvents.subscribe(({ userId, action }) => {
        const target = deck.find((p) => p.userId === userId);
        if (!target) return;
        if (deck[0]?.userId === userId) {
          deckRef.current?.swipe(action);
        } else {
          void decide(target, action);
        }
      }),
    [deck, decide]
  );

  const open = (profile: PublicProfile) => navigation.navigate("ProfileDetail", { userId: profile.userId, profile });

  let body: React.ReactNode;
  if (status === "loading") {
    body = <LoadingState label="On vous cherche de belles rencontres…" />;
  } else if (status === "error") {
    body = <ErrorState message={error ?? "Impossible de charger les profils."} onRetry={() => load(true)} />;
  } else if (deck.length === 0) {
    body = (
      <EmptyState
        actionLabel="Élargir mes critères"
        body="Vous avez vu tous les profils qui correspondent à vos critères pour le moment. Revenez bientôt ou élargissez votre recherche."
        icon="compass-outline"
        onAction={() => navigation.navigate("Preferences")}
        title="Plus personne pour l'instant"
      />
    );
  } else {
    body = (
      <>
        <View style={styles.deck}>
          <SwipeDeck onOpen={open} onSwipe={decide} profiles={deck} ref={deckRef} />
        </View>
        <View style={styles.actions}>
          <IconButton icon="close" label="Passer" onPress={() => deckRef.current?.swipe("pass")} size={64} testID="discover-pass" tone="surface" />
          <IconButton icon="person-outline" label="Voir le profil complet" onPress={() => deck[0] && open(deck[0])} size={48} tone="surface" />
          <IconButton icon="heart" label="J'aime" onPress={() => deckRef.current?.swipe("like")} size={64} testID="discover-like" tone="primary" />
        </View>
      </>
    );
  }

  return (
    <Screen>
      <View style={styles.header}>
        <Text accessibilityRole="header" variant="title">
          Découvrir
        </Text>
        <IconButton icon="options-outline" label="Préférences de rencontre" onPress={() => navigation.navigate("Preferences")} />
      </View>
      {me?.profile && !me.profile.discoverable ? (
        <View style={[styles.hidden, { backgroundColor: colors.accentSoft }]}>
          <Text tone="accent" variant="caption">
            Votre profil est masqué : les autres ne vous voient pas. Réactivez-le dans Confidentialité.
          </Text>
        </View>
      ) : null}
      {body}
      <MatchModal
        match={match}
        myPhoto={me?.photos[0]}
        onClose={() => setMatch(null)}
        onMessage={(m) => {
          setMatch(null);
          navigation.navigate("Conversation", { matchId: m.id, user: m.user });
        }}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  header: { flexDirection: "row", alignItems: "center", justifyContent: "space-between", marginBottom: spacing.sm },
  hidden: { padding: spacing.sm, borderRadius: 12, marginBottom: spacing.sm },
  deck: { flex: 1, marginBottom: spacing.md },
  actions: { flexDirection: "row", alignItems: "center", justifyContent: "center", gap: spacing.xl, paddingBottom: spacing.md }
});
