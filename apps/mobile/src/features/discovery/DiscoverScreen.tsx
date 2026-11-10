import React, { useCallback, useEffect, useRef, useState } from "react";
import { Platform, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import * as Haptics from "expo-haptics";
import { useQueryClient } from "@tanstack/react-query";
import { ApiError, errorMessage } from "../../api/client";
import { discoveryApi, matchesApi } from "../../api/endpoints";
import type { ProfileCard, SwipeAction } from "../../api/types";
import { keys, useMyProfile } from "../../hooks/queries";
import { useLocationSync } from "../../hooks/useLocation";
import { celebrateMatch } from "../../hooks/useRealtime";
import type { MainStackParamList } from "../../navigation/types";
import { useTheme } from "../../theme/ThemeProvider";
import { EmptyState, ErrorState, Header, IconButton, LoadingState, Screen, Text, showToast } from "../../ui";
import { SwipeDeck, type SwipeDeckHandle } from "./SwipeDeck";

const PREFETCH_BELOW = 3;

export function DiscoverScreen() {
  const theme = useTheme();
  const qc = useQueryClient();
  const navigation = useNavigation<NativeStackNavigationProp<MainStackParamList>>();
  const me = useMyProfile();
  const syncLocation = useLocationSync();
  const deckRef = useRef<SwipeDeckHandle>(null);
  const [cards, setCards] = useState<ProfileCard[]>([]);
  const [status, setStatus] = useState<"loading" | "ready" | "error" | "incomplete">("loading");
  const [exhausted, setExhausted] = useState(false);
  const [locating, setLocating] = useState(false);
  const seen = useRef<string[]>([]);
  const loadingMore = useRef(false);

  const hasLocation = me.data?.profile?.hasLocation ?? false;

  const load = useCallback(
    async (reset = false) => {
      if (loadingMore.current) return;
      loadingMore.current = true;
      if (reset) setStatus("loading");
      try {
        const current = reset ? [] : cards;
        const exclude = [...seen.current, ...current.map((c) => c.userId)];
        const fresh = await discoveryApi.deck(exclude);
        setCards((prev) => {
          const base = reset ? [] : prev;
          const known = new Set([...base.map((c) => c.userId), ...seen.current]);
          return [...base, ...fresh.filter((c) => !known.has(c.userId))];
        });
        setExhausted(fresh.length === 0);
        setStatus("ready");
      } catch (e) {
        setStatus(e instanceof ApiError && e.code === "profile_incomplete" ? "incomplete" : "error");
      } finally {
        loadingMore.current = false;
      }
    },
    [cards]
  );

  useEffect(() => {
    if (me.data && hasLocation) void load(true);
    else if (me.data) setStatus("incomplete");
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [me.data?.profile?.hasLocation, me.data?.photos.length]);

  useEffect(() => {
    if (status === "ready" && !exhausted && cards.length < PREFETCH_BELOW) void load();
  }, [cards.length, exhausted, load, status]);

  const onSwipe = useCallback(
    async (card: ProfileCard, action: SwipeAction) => {
      seen.current = [...seen.current.slice(-80), card.userId];
      setCards((prev) => prev.filter((c) => c.userId !== card.userId));
      if (Platform.OS !== "web") void Haptics.selectionAsync();
      try {
        const res = await discoveryApi.swipe(card.userId, action);
        if (res.matched && res.matchId) {
          if (Platform.OS !== "web") void Haptics.notificationAsync(Haptics.NotificationFeedbackType.Success);
          void qc.invalidateQueries({ queryKey: keys.matches });
          celebrateMatch(await matchesApi.get(res.matchId));
        }
      } catch (e) {
        if (!(e instanceof ApiError && (e.code === "already_swiped" || e.code === "profile_unavailable"))) {
          showToast(errorMessage(e), "error");
        }
      }
    },
    [qc]
  );

  const enableLocation = async () => {
    setLocating(true);
    const outcome = await syncLocation();
    setLocating(false);
    if (outcome === "updated") {
      await qc.invalidateQueries({ queryKey: keys.me });
    } else {
      showToast(outcome === "denied" ? "Autorisez la localisation dans les réglages de votre appareil." : "Position indisponible pour le moment.", "error");
    }
  };

  let content: React.ReactNode;
  if (me.isLoading || status === "loading") {
    content = <LoadingState label="Nous cherchons des profils pour vous…" />;
  } else if (!hasLocation || status === "incomplete") {
    content = (
      <EmptyState
        actionLabel={locating ? "Localisation…" : "Activer la localisation"}
        icon="location-outline"
        message="Votre position (arrondie à environ 1 km, jamais affichée) sert uniquement à vous proposer des personnes proches."
        onAction={enableLocation}
        title="Où êtes-vous ?"
      />
    );
  } else if (status === "error") {
    content = <ErrorState onRetry={() => load(true)} />;
  } else if (cards.length === 0) {
    content = (
      <EmptyState
        actionLabel="Élargir mes critères"
        icon="sparkles-outline"
        message="Vous avez vu tous les profils correspondant à vos critères. Revenez plus tard ou élargissez la distance et l'âge."
        onAction={() => navigation.navigate("Preferences")}
        title="C'est tout pour l'instant"
      />
    );
  } else {
    content = (
      <>
        <View style={styles.deck}>
          <SwipeDeck cards={cards} onOpen={(c) => navigation.navigate("ProfileDetail", { userId: c.userId })} onSwipe={onSwipe} ref={deckRef} />
        </View>
        <View style={styles.actions}>
          <IconButton
            filled
            icon="close"
            label={`Passer ${cards[0]?.firstName ?? ""}`}
            onPress={() => deckRef.current?.swipe("pass")}
            size={64}
            testID="action-pass"
          />
          <IconButton
            filled
            icon="person-outline"
            label="Voir le profil"
            onPress={() => cards[0] && navigation.navigate("ProfileDetail", { userId: cards[0].userId })}
            size={48}
            tone="muted"
          />
          <IconButton
            filled
            icon="heart"
            label={`J'aime ${cards[0]?.firstName ?? ""}`}
            onPress={() => deckRef.current?.swipe("like")}
            size={64}
            testID="action-like"
            tone="primary"
          />
        </View>
      </>
    );
  }

  return (
    <Screen>
      <Header
        large
        right={<IconButton icon="options-outline" label="Préférences de rencontre" onPress={() => navigation.navigate("Preferences")} />}
        title="Découvrir"
      />
      {me.data?.profile && !me.data.profile.isDiscoverable ? (
        <View style={[styles.banner, { backgroundColor: theme.colors.accentSoft }]}>
          <Text tone="accent" variant="small">
            Votre profil est en pause : vous restez invisible tant que vous ne le réactivez pas (Profil › Confidentialité).
          </Text>
        </View>
      ) : null}
      {content}
    </Screen>
  );
}

const styles = StyleSheet.create({
  deck: { flex: 1, marginTop: 4, width: "100%", maxWidth: 480, alignSelf: "center" },
  actions: { flexDirection: "row", alignItems: "center", justifyContent: "center", gap: 28, paddingVertical: 18 },
  banner: { borderRadius: 14, padding: 12, marginBottom: 8 }
});
