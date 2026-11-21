import React, { useCallback, useEffect, useRef, useState } from "react";
import { View } from "react-native";
import type { BottomTabScreenProps } from "@react-navigation/bottom-tabs";
import type { CompositeScreenProps } from "@react-navigation/native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useQueryClient } from "@tanstack/react-query";
import { ApiError, errorMessage } from "../../api/client";
import { discoveryApi } from "../../api/endpoints";
import { queryKeys } from "../../api/queryKeys";
import type { FeedItem, SwipeAction, UserSummary } from "../../api/types";
import type { RootStackParamList, TabParamList } from "../../navigation/types";
import { makeStyles } from "../../theme";
import { BrandMark, EmptyState, ErrorState, IconButton, LoadingState, prefetchImages, Screen, showToast, Text } from "../../ui";
import { useMyProfile } from "../profile/hooks";
import { MatchModal } from "./MatchModal";
import { SwipeCard, type SwipeCardHandle } from "./SwipeCard";

type Props = CompositeScreenProps<BottomTabScreenProps<TabParamList, "Discover">, NativeStackScreenProps<RootStackParamList>>;

type Status = "loading" | "ready" | "error" | "incomplete";

export function DiscoverScreen({ navigation, route }: Props) {
  const styles = useStyles();
  const queryClient = useQueryClient();
  const me = useMyProfile().data;
  const [queue, setQueue] = useState<FeedItem[]>([]);
  const [status, setStatus] = useState<Status>("loading");
  const [error, setError] = useState<string | null>(null);
  const [exhausted, setExhausted] = useState(false);
  const [match, setMatch] = useState<{ user: UserSummary } | null>(null);
  const loadingRef = useRef(false);
  const decidedRef = useRef(new Set<string>());
  const topCardRef = useRef<SwipeCardHandle>(null);
  const queueRef = useRef(queue);
  queueRef.current = queue;

  const load = useCallback(async (reset = false) => {
    if (loadingRef.current) return;
    loadingRef.current = true;
    if (reset) {
      setStatus("loading");
      setExhausted(false);
    }
    try {
      const current = reset ? [] : queueRef.current;
      const res = await discoveryApi.feed(
        current.map((c) => c.userId),
        10
      );
      const fresh = res.profiles.filter((p) => !decidedRef.current.has(p.userId) && !current.some((c) => c.userId === p.userId));
      setQueue(reset ? fresh : [...current, ...fresh]);
      setExhausted(res.profiles.length === 0);
      setStatus("ready");
      prefetchImages(fresh.slice(0, 3).flatMap((p) => p.photos.slice(0, 2).map((ph) => ph.url)));
    } catch (err) {
      if (err instanceof ApiError && err.code === "profile_incomplete") {
        setStatus("incomplete");
      } else if (reset || queueRef.current.length === 0) {
        setError(errorMessage(err));
        setStatus("error");
      } else {
        showToast(errorMessage(err), "error");
      }
    } finally {
      loadingRef.current = false;
    }
  }, []);

  useEffect(() => {
    void load(true);
  }, [load]);

  useEffect(() => {
    if (status === "ready" && queue.length <= 3 && !exhausted) {
      void load();
    }
  }, [queue.length, status, exhausted, load]);

  const decide = useCallback(
    async (card: FeedItem, action: SwipeAction) => {
      decidedRef.current.add(card.userId);
      setQueue((q) => q.filter((c) => c.userId !== card.userId));
      try {
        const result = await discoveryApi.swipe(card.userId, action);
        if (result.matched && result.match) {
          setMatch({ user: result.match.user });
          void queryClient.invalidateQueries({ queryKey: queryKeys.chats });
        }
      } catch (err) {
        if (err instanceof ApiError && (err.code === "already_swiped" || err.code === "profile_unavailable")) {
          return;
        }
        if (err instanceof ApiError && err.status === 0) {
          // Network issue: give the card back so the decision isn't lost.
          decidedRef.current.delete(card.userId);
          setQueue((q) => [card, ...q]);
        }
        showToast(errorMessage(err), "error");
      }
    },
    [queryClient]
  );

  // Decisions taken from the full profile screen.
  const decision = route.params?.decision;
  useEffect(() => {
    if (!decision) return;
    const top = queueRef.current[0];
    if (top && top.userId === decision.userId) {
      topCardRef.current?.swipe(decision.action);
    } else {
      const card = queueRef.current.find((c) => c.userId === decision.userId);
      if (card) void decide(card, decision.action);
    }
    navigation.setParams({ decision: undefined });
  }, [decision, decide, navigation]);

  const top = queue[0];
  const next = queue[1];
  const openProfile = (card: FeedItem) => navigation.navigate("ProfileDetail", { userId: card.userId, mode: "discovery" });

  let body: React.ReactNode;
  if (status === "loading") {
    body = <LoadingState label="On cherche des profils pour vous…" />;
  } else if (status === "error") {
    body = <ErrorState message={error ?? "Impossible de charger les profils."} onRetry={() => void load(true)} />;
  } else if (status === "incomplete") {
    body = (
      <EmptyState
        icon="images-outline"
        title="Ajoutez une photo"
        message="Votre profil a besoin d'au moins une photo pour découvrir et être découvert."
        actionLabel="Ajouter des photos"
        onAction={() => navigation.navigate("EditPhotos")}
      />
    );
  } else if (!top) {
    body = (
      <EmptyState
        icon="compass-outline"
        title="Vous avez fait le tour"
        message="Plus personne ne correspond à vos critères pour le moment. Revenez bientôt ou élargissez vos préférences."
        actionLabel="Élargir mes critères"
        onAction={() => navigation.navigate("Preferences")}
        secondaryLabel="Actualiser"
        onSecondary={() => void load(true)}
      />
    );
  } else {
    body = (
      <>
        <View style={styles.deck}>
          {next ? <SwipeCard key={next.userId} card={next} active={false} onDecision={decide} onOpen={openProfile} /> : null}
          <SwipeCard key={top.userId} ref={topCardRef} card={top} active onDecision={decide} onOpen={openProfile} />
        </View>
        <View style={styles.actions}>
          <IconButton
            testID="discovery-pass"
            icon="close"
            accessibilityLabel={`Passer ${top.firstName}`}
            size={64}
            style={styles.passButton}
            tone="pass"
            onPress={() => topCardRef.current?.swipe("pass")}
          />
          <IconButton
            testID="discovery-like"
            icon="heart"
            accessibilityLabel={`J'aime ${top.firstName}`}
            size={72}
            filled
            tone="like"
            onPress={() => topCardRef.current?.swipe("like")}
          />
        </View>
      </>
    );
  }

  return (
    <Screen padded={false}>
      <View style={styles.header}>
        <View style={styles.brand}>
          <BrandMark size={28} />
          <Text variant="title">Lueur</Text>
        </View>
        <IconButton
          icon="options-outline"
          accessibilityLabel="Préférences de rencontre"
          onPress={() => navigation.navigate("Preferences")}
        />
      </View>
      {body}
      <MatchModal
        match={match}
        myName={me?.profile?.firstName ?? ""}
        myPhotoUrl={me?.photos[0]?.thumbUrl ?? null}
        onClose={() => setMatch(null)}
        onMessage={() => {
          const user = match?.user;
          setMatch(null);
          if (user) navigation.navigate("Conversation", { userId: user.id, firstName: user.firstName, photoUrl: user.photoUrl });
        }}
      />
    </Screen>
  );
}

const useStyles = makeStyles((t) => ({
  header: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    paddingHorizontal: t.spacing.xl,
    paddingVertical: t.spacing.sm
  },
  brand: { flexDirection: "row", alignItems: "center", gap: t.spacing.sm },
  deck: { flex: 1, marginHorizontal: t.spacing.lg, marginTop: t.spacing.xs },
  actions: {
    flexDirection: "row",
    justifyContent: "center",
    alignItems: "center",
    gap: t.spacing.xxl,
    paddingVertical: t.spacing.lg,
    paddingBottom: t.spacing.xl
  },
  passButton: { backgroundColor: t.colors.surface, borderWidth: 1, borderColor: t.colors.border }
}));
