import React, { useCallback, useEffect, useRef, useState } from "react";
import { StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useQueryClient } from "@tanstack/react-query";
import { ApiError, errorMessage } from "../api/client";
import { fetchDiscover, resetPasses, swipe } from "../api/discovery";
import type { MatchInfo, PublicProfile } from "../api/types";
import { MatchOverlay } from "../components/MatchOverlay";
import { Screen } from "../components/Screen";
import { SwipeDeck } from "../components/SwipeDeck";
import { APP_NAME } from "../components/labels";
import { onProfileSwiped } from "../hooks/deckBus";
import { queryKeys, useMyProfile } from "../hooks/queries";
import type { MainStackParamList } from "../navigation/types";
import { showToast } from "../shared/feedback";
import { Button, Loader, Text, spacing } from "../shared/ui";

const PAGE_SIZE = 10;
const PREFETCH_AT = 3;

type Nav = NativeStackNavigationProp<MainStackParamList>;
type Status = "loading" | "ready" | "empty" | "error" | "incomplete";

export default function HomeScreen() {
  const navigation = useNavigation<Nav>();
  const client = useQueryClient();
  const { data: profile } = useMyProfile();

  const [deck, setDeck] = useState<PublicProfile[]>([]);
  const [status, setStatus] = useState<Status>("loading");
  const [error, setError] = useState<string | null>(null);
  const [match, setMatch] = useState<MatchInfo | null>(null);
  const fetching = useRef(false);
  const exhausted = useRef(false);
  const seen = useRef(new Set<string>());

  const loadMore = useCallback(async () => {
    if (fetching.current || exhausted.current) {
      return;
    }
    fetching.current = true;
    try {
      const batch = await fetchDiscover(PAGE_SIZE);
      const fresh = batch.filter((p) => !seen.current.has(p.userId));
      fresh.forEach((p) => seen.current.add(p.userId));
      if (fresh.length === 0) {
        exhausted.current = true;
      }
      setDeck((current) => {
        const next = [...current, ...fresh];
        setStatus(next.length > 0 ? "ready" : "empty");
        return next;
      });
    } catch (err) {
      if (err instanceof ApiError && err.code === "profile_incomplete") {
        setStatus("incomplete");
      } else {
        setError(errorMessage(err));
        setStatus((s) => (s === "ready" ? s : "error"));
      }
    } finally {
      fetching.current = false;
    }
  }, []);

  const refresh = useCallback(() => {
    exhausted.current = false;
    seen.current = new Set(deck.map((p) => p.userId));
    setError(null);
    setStatus(deck.length > 0 ? "ready" : "loading");
    void loadMore();
  }, [deck, loadMore]);

  // First load, and again whenever the profile (preferences, location, visibility) changes:
  // the cards on screen were chosen with the old criteria.
  const profileStamp = profile?.updatedAt;
  const lastStamp = useRef<string | undefined>(undefined);
  useEffect(() => {
    if (lastStamp.current !== undefined && lastStamp.current !== profileStamp) {
      exhausted.current = false;
      seen.current = new Set();
      setDeck([]);
      setStatus("loading");
    }
    lastStamp.current = profileStamp ?? "";
    void loadMore();
  }, [loadMore, profileStamp]);

  useEffect(() => {
    if (status === "ready" && deck.length <= PREFETCH_AT) {
      void loadMore();
    }
  }, [deck.length, status, loadMore]);

  const handleResult = useCallback(
    (result: { matched: boolean; match?: MatchInfo }) => {
      if (result.matched && result.match) {
        setMatch(result.match);
        void client.invalidateQueries({ queryKey: queryKeys.conversations });
        void client.invalidateQueries({ queryKey: queryKeys.notifications });
      }
    },
    [client]
  );

  // Swipes made from the profile detail screen remove the card from the deck too.
  useEffect(
    () =>
      onProfileSwiped((userId, result) => {
        setDeck((current) => {
          const next = current.filter((p) => p.userId !== userId);
          if (next.length === 0) setStatus("empty");
          return next;
        });
        handleResult(result);
      }),
    [handleResult]
  );

  const onSwipe = useCallback(
    (target: PublicProfile, action: "like" | "pass") => {
      setDeck((current) => {
        const next = current.filter((p) => p.userId !== target.userId);
        if (next.length === 0) setStatus(exhausted.current ? "empty" : "loading");
        return next;
      });
      swipe(target.userId, action)
        .then(handleResult)
        .catch((err) => {
          if (err instanceof ApiError && (err.status === 409 || err.status === 404)) {
            return; // already handled elsewhere, or no longer available
          }
          showToast(errorMessage(err, "Action impossible pour le moment."), "error");
          setDeck((current) => [target, ...current]);
          setStatus("ready");
        });
    },
    [handleResult]
  );

  const reviewPasses = async () => {
    try {
      const count = await resetPasses();
      showToast(count > 0 ? `${count} profil(s) remis dans votre sélection.` : "Aucun profil passé à revoir.", "info");
      seen.current = new Set();
      exhausted.current = false;
      setStatus("loading");
      void loadMore();
    } catch (err) {
      showToast(errorMessage(err), "error");
    }
  };

  const openProfile = (target: PublicProfile) => navigation.navigate("Person", { userId: target.userId });

  return (
    <Screen
      edges={["top", "left", "right"]}
      header={
        <View style={styles.header}>
          <Text style={styles.brand} tone="primary" weight="bold">
            {APP_NAME}
          </Text>
          <Button
            label="Préférences"
            onPress={() => navigation.navigate("EditProfile")}
            size="sm"
            testID="open-preferences"
            variant="ghost"
          />
        </View>
      }
      contentStyle={styles.content}
      testID="home-screen"
    >
      {status === "loading" ? <Loader fullScreen label="Recherche de profils…" /> : null}

      {status === "ready" ? <SwipeDeck onOpenProfile={openProfile} onSwipe={onSwipe} profiles={deck} /> : null}

      {status === "empty" ? (
        <View style={styles.center} testID="deck-empty">
          <Text style={styles.centerTitle} variant="heading" weight="bold">
            Vous avez tout vu pour l'instant
          </Text>
          <Text style={styles.centerText} tone="muted">
            De nouvelles personnes arrivent chaque jour. Élargissez vos préférences ou revoyez les profils que vous avez
            passés.
          </Text>
          <Button label="Actualiser" onPress={refresh} testID="deck-refresh" />
          <Button
            label="Revoir les profils passés"
            onPress={reviewPasses}
            testID="deck-review-passes"
            variant="outline"
          />
          <Button label="Modifier mes préférences" onPress={() => navigation.navigate("EditProfile")} variant="ghost" />
        </View>
      ) : null}

      {status === "error" ? (
        <View style={styles.center}>
          <Text style={styles.centerTitle} variant="heading" weight="bold">
            Chargement impossible
          </Text>
          <Text style={styles.centerText} tone="muted">
            {error ?? "Vérifiez votre connexion."}
          </Text>
          <Button label="Réessayer" onPress={refresh} testID="deck-retry" />
        </View>
      ) : null}

      {status === "incomplete" ? (
        <View style={styles.center}>
          <Text style={styles.centerTitle} variant="heading" weight="bold">
            Complétez votre profil
          </Text>
          <Text style={styles.centerText} tone="muted">
            Ajoutez au moins une photo pour découvrir des profils.
          </Text>
          <Button label="Modifier mon profil" onPress={() => navigation.navigate("EditProfile")} />
        </View>
      ) : null}

      <MatchOverlay
        match={match}
        myName={profile?.firstName}
        myPhotoPath={profile?.photos[0]?.thumbUrl}
        onClose={() => setMatch(null)}
        onMessage={(m) => {
          setMatch(null);
          navigation.navigate("Conversation", {
            conversationId: m.conversationId,
            matchId: m.id,
            userId: m.user.userId,
            name: m.user.firstName
          });
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
    paddingHorizontal: spacing.lg,
    paddingTop: spacing.sm
  },
  brand: { fontSize: 28, lineHeight: 34, letterSpacing: -1 },
  content: { paddingTop: spacing.sm, paddingBottom: 90 },
  center: { flex: 1, alignItems: "center", justifyContent: "center", gap: spacing.md, padding: spacing.lg },
  centerTitle: { textAlign: "center" },
  centerText: { textAlign: "center", maxWidth: 360 }
});
