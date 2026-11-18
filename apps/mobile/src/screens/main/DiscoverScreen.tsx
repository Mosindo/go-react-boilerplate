import React, { useCallback, useEffect, useRef, useState } from "react";
import { StyleSheet, View } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import { discoveryApi } from "../../api/endpoints";
import { errorMessage, isApiError } from "../../api/errors";
import type { Match, ProfileCard, SwipeDecision } from "../../api/types";
import { MatchCelebration } from "../../components/MatchCelebration";
import { SwipeCard, type SwipeCardHandle } from "../../components/SwipeCard";
import { afterModalDismiss, useFeedback } from "../../feedback";
import { useMyProfile } from "../../hooks/queries";
import { discoveryEvents } from "../../hooks/discoveryEvents";
import { queryKeys } from "../../hooks/queryClient";
import { useRealtime } from "../../hooks/useRealtime";
import type { TabScreenProps } from "../../navigation/types";
import { layout, spacing } from "../../theme";
import { Card, EmptyState, ErrorState, Header, IconButton, LoadingState, Screen, Text } from "../../ui";

const REFILL_BELOW = 3;

export default function DiscoverScreen({ navigation }: TabScreenProps<"Discover">) {
  const qc = useQueryClient();
  const { toast } = useFeedback();
  const { onMatch } = useRealtime();
  const me = useMyProfile();
  const [deck, setDeck] = useState<ProfileCard[]>([]);
  const [status, setStatus] = useState<"loading" | "ready" | "error">("loading");
  const [error, setError] = useState<string | null>(null);
  const [exhausted, setExhausted] = useState(false);
  const [celebrate, setCelebrate] = useState<Match | null>(null);
  const seen = useRef(new Set<string>());
  const loading = useRef(false);
  const topCard = useRef<SwipeCardHandle>(null);

  const load = useCallback(async (reset = false) => {
    if (loading.current) {
      return;
    }
    loading.current = true;
    try {
      const profiles = await discoveryApi.feed(10);
      if (reset) {
        seen.current.clear();
      }
      const fresh = profiles.filter((p) => !seen.current.has(p.userId));
      fresh.forEach((p) => seen.current.add(p.userId));
      setDeck((prev) => (reset ? fresh : [...prev, ...fresh]));
      setExhausted(fresh.length === 0);
      setStatus("ready");
      setError(null);
    } catch (e) {
      setError(errorMessage(e));
      setStatus((s) => (s === "loading" ? "error" : s));
    } finally {
      loading.current = false;
    }
  }, []);

  useEffect(() => {
    void load(true);
  }, [load]);

  // Matches created by the other member while browsing.
  useEffect(() => onMatch((match) => setCelebrate(match)), [onMatch]);

  // Decisions taken from the full profile screen.
  useEffect(
    () =>
      discoveryEvents.subscribe(({ userId, match }) => {
        setDeck((prev) => prev.filter((p) => p.userId !== userId));
        if (match) {
          setCelebrate(match);
        }
      }),
    []
  );

  useEffect(() => {
    if (status === "ready" && deck.length < REFILL_BELOW && !exhausted) {
      void load();
    }
  }, [deck.length, exhausted, load, status]);

  const decide = useCallback(
    async (profile: ProfileCard, decision: SwipeDecision) => {
      setDeck((prev) => prev.filter((p) => p.userId !== profile.userId));
      try {
        const result = await discoveryApi.swipe(profile.userId, decision);
        if (result.matched && result.match) {
          setCelebrate(result.match);
          void qc.invalidateQueries({ queryKey: queryKeys.matches });
        }
      } catch (e) {
        if (!isApiError(e, "already_swiped") && !isApiError(e, "not_found")) {
          toast(errorMessage(e), "error");
        }
      }
    },
    [qc, toast]
  );

  const current = deck[0];
  const next = deck[1];
  const paused = me.data?.profile && !me.data.profile.isDiscoverable;

  let body: React.ReactNode;
  if (status === "loading") {
    body = <LoadingState label="On cherche des profils pour vous…" />;
  } else if (status === "error" && deck.length === 0) {
    body = <ErrorState message={error ?? "Impossible de charger les profils."} onRetry={() => void load(true)} />;
  } else if (!current) {
    body = (
      <EmptyState
        actionLabel="Ajuster mes critères"
        icon="compass-outline"
        message="Vous avez vu tous les profils qui correspondent à vos critères. Revenez plus tard ou élargissez votre recherche."
        onAction={() => navigation.navigate("Preferences")}
        testID="discover-empty"
        title="C’est tout pour le moment"
      />
    );
  } else {
    body = (
      <View style={styles.deckArea}>
        <View style={styles.deck}>
          {next ? (
            <View pointerEvents="none" style={[StyleSheet.absoluteFill, styles.behind]}>
              <SwipeCard key={next.userId} onDecision={() => undefined} onOpen={() => undefined} profile={next} />
            </View>
          ) : null}
          <SwipeCard
            key={current.userId}
            ref={topCard}
            onDecision={(decision) => void decide(current, decision)}
            onOpen={() => navigation.navigate("ProfileDetail", { userId: current.userId, source: "discovery" })}
            profile={current}
          />
        </View>
        <View style={styles.actions}>
          <IconButton
            icon="close"
            label={`Passer ${current.firstName}`}
            onPress={() => topCard.current?.swipe("pass")}
            size={64}
            testID="discover-pass"
            tone="surface"
            color="textMuted"
          />
          <IconButton
            icon="person-outline"
            label="Voir le profil complet"
            onPress={() => navigation.navigate("ProfileDetail", { userId: current.userId, source: "discovery" })}
            size={48}
            tone="surface"
          />
          <IconButton
            icon="heart"
            label={`Aimer ${current.firstName}`}
            onPress={() => topCard.current?.swipe("like")}
            size={64}
            testID="discover-like"
            tone="accent"
          />
        </View>
      </View>
    );
  }

  return (
    <Screen testID="discover-screen">
      <Header
        right={
          <IconButton
            icon="options-outline"
            label="Critères de recherche"
            onPress={() => navigation.navigate("Preferences")}
            tone="surface"
          />
        }
        title="Découvrir"
      />
      {paused ? (
        <Card>
          <Text color="textMuted" variant="caption">
            Votre profil est en pause : vous pouvez découvrir des profils, mais personne ne vous voit.
          </Text>
        </Card>
      ) : null}
      {me.data?.profile?.underReview ? (
        <Card>
          <Text color="textMuted" variant="caption">
            Votre profil est temporairement masqué le temps d’une vérification par notre équipe.
          </Text>
        </Card>
      ) : null}
      <View style={styles.flex}>{body}</View>
      <MatchCelebration
        match={celebrate}
        myName={me.data?.profile?.firstName ?? ""}
        myPhotoUrl={me.data?.photos[0]?.url}
        onClose={() => setCelebrate(null)}
        onMessage={(match) => {
          setCelebrate(null);
          void afterModalDismiss().then(() =>
            navigation.navigate("Chat", { userId: match.user.userId, firstName: match.user.firstName, photoUrl: match.user.photoUrl })
          );
        }}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1, paddingBottom: 84 },
  deckArea: { flex: 1, gap: spacing.lg, width: "100%", maxWidth: layout.maxContentWidth, alignSelf: "center", paddingTop: spacing.sm },
  deck: { flex: 1 },
  behind: { transform: [{ scale: 0.96 }, { translateY: 10 }], opacity: 0.9 },
  actions: { flexDirection: "row", alignItems: "center", justifyContent: "center", gap: spacing.xl }
});
