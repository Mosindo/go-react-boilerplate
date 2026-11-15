import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { StyleSheet, View } from "react-native";
import { useFocusEffect, useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useQuery } from "@tanstack/react-query";
import * as Haptics from "expo-haptics";
import { matchingApi } from "../../api/matching";
import { notificationsApi } from "../../api/notifications";
import { queryKeys } from "../../api/queryKeys";
import type { MatchSummary, ProfileCard, SwipeAction } from "../../api/types";
import { errorMessage, isApiError } from "../../lib/errors";
import type { AppStackParamList } from "../../navigation/types";
import { celebratedMatches, useRealtimeConnected } from "../../realtime/RealtimeProvider";
import { radii, spacing, useTheme } from "../../theme";
import { EmptyState, ErrorState, Header, Icon, IconButton, Loading, Screen, Text, toast } from "../../ui";
import { useMyProfile } from "../profile/hooks";
import { swipedIds, useSwipe } from "./hooks";
import { MatchModal } from "./MatchModal";
import { SwipeCard, type SwipeCardHandle } from "./SwipeCard";

type Nav = NativeStackNavigationProp<AppStackParamList>;

export function DiscoveryScreen() {
  const navigation = useNavigation<Nav>();
  const { colors } = useTheme();
  const me = useMyProfile();
  const connected = useRealtimeConnected();
  const badges = useQuery({ queryKey: queryKeys.badges, queryFn: notificationsApi.badges, refetchInterval: connected ? false : 20_000 });
  const discovery = useQuery({ queryKey: queryKeys.discovery, queryFn: () => matchingApi.discovery(10), staleTime: 0 });
  const swipe = useSwipe();
  const [queue, setQueue] = useState<ProfileCard[]>([]);
  const [match, setMatch] = useState<MatchSummary | null>(null);
  const handled = useRef(new Set<string>());
  const topCard = useRef<SwipeCardHandle>(null);
  const queueRef = useRef(queue);
  queueRef.current = queue;

  // Merge freshly fetched candidates into the local deck, skipping anyone
  // already handled in this session. Auto-refill only while fetches keep
  // bringing new people, so an exhausted pool never loops.
  const [canRefill, setCanRefill] = useState(true);
  useEffect(() => {
    if (!discovery.data) return;
    const known = new Set(queueRef.current.map((c) => c.userId));
    const fresh = discovery.data.profiles.filter((p) => !known.has(p.userId) && !handled.current.has(p.userId) && !swipedIds.has(p.userId));
    setCanRefill(fresh.length > 0);
    if (fresh.length) setQueue((current) => [...current, ...fresh]);
  }, [discovery.data]);

  const { refetch, isFetching } = discovery;
  useEffect(() => {
    if (queue.length < 3 && canRefill && !isFetching && discovery.data) {
      void refetch();
    }
  }, [queue.length, canRefill, isFetching, refetch, discovery.data]);

  // A like/pass made from the full profile screen removes the card here too.
  useFocusEffect(
    useCallback(() => {
      setQueue((q) => (q.some((c) => swipedIds.has(c.userId)) ? q.filter((c) => !swipedIds.has(c.userId)) : q));
    }, [])
  );

  const myInterests = useMemo(() => me.data?.profile.interests.map((i) => i.id) ?? [], [me.data]);

  const onSwiped = useCallback(
    (card: ProfileCard, action: SwipeAction) => {
      handled.current.add(card.userId);
      setQueue((q) => q.filter((c) => c.userId !== card.userId));
      if (action === "like") void Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light).catch(() => undefined);
      swipe.mutate(
        { userId: card.userId, action },
        {
          onSuccess: (result) => {
            if (result.matched && result.match) {
              celebratedMatches.add(result.match.id);
              setMatch(result.match);
            }
          },
          onError: (e) => {
            if (isApiError(e, "already_swiped") || isApiError(e, "profile_unavailable")) return;
            toast(errorMessage(e), "error");
          }
        }
      );
    },
    [swipe]
  );

  const header = (
    <Header
      title="Découvrir"
      large
      right={
        <>
          <IconButton
            icon="options-outline"
            label="Préférences de rencontre"
            onPress={() => navigation.navigate("Preferences")}
            testID="open-preferences"
          />
          <IconButton
            icon="notifications-outline"
            label="Notifications"
            badge={badges.data?.unreadNotifications}
            onPress={() => navigation.navigate("Notifications")}
            testID="open-notifications"
          />
        </>
      }
    />
  );

  let body: React.ReactNode;
  if (discovery.isPending) {
    body = <Loading label="Recherche de profils…" />;
  } else if (discovery.isError && queue.length === 0) {
    body = isApiError(discovery.error, "profile_incomplete") ? (
      <EmptyState
        icon="person-circle-outline"
        title="Profil incomplet"
        message={errorMessage(discovery.error)}
        actionLabel="Compléter"
        onAction={() => navigation.navigate("EditProfile")}
      />
    ) : (
      <ErrorState message={errorMessage(discovery.error)} onRetry={() => void refetch()} />
    );
  } else if (queue.length === 0) {
    body = (
      <EmptyState
        icon="compass-outline"
        title="Vous avez fait le tour"
        message="Revenez un peu plus tard ou élargissez vos critères d'âge et de distance."
        actionLabel="Élargir mes critères"
        onAction={() => navigation.navigate("Preferences")}
        testID="discovery-empty"
      />
    );
    body = (
      <View style={styles.flex}>
        {body}
        <IconButton
          icon="refresh"
          label="Actualiser"
          onPress={() => {
            setCanRefill(true);
            void refetch();
          }}
          style={styles.refresh}
        />
      </View>
    );
  } else {
    const [current, nextCard] = queue;
    body = (
      <>
        <View style={styles.deck}>
          {nextCard ? (
            <View style={[StyleSheet.absoluteFill, styles.behind]} pointerEvents="none">
              <SwipeCard
                key={nextCard.userId}
                card={nextCard}
                sharedCount={0}
                active={false}
                onSwiped={() => undefined}
                onOpen={() => undefined}
              />
            </View>
          ) : null}
          <SwipeCard
            key={current.userId}
            ref={topCard}
            card={current}
            active
            sharedCount={current.interests.filter((i) => myInterests.includes(i.id)).length}
            onSwiped={(action) => onSwiped(current, action)}
            onOpen={() => navigation.navigate("ProfileDetail", { userId: current.userId, source: "discovery" })}
          />
        </View>
        <View style={styles.actions}>
          <IconButton
            icon="close"
            label={`Passer ${current.firstName}`}
            size={64}
            tone="danger"
            background="surface"
            onPress={() => topCard.current?.swipe("pass")}
            testID="swipe-pass"
            style={[styles.action, { borderColor: colors.border }]}
          />
          <IconButton
            icon="person-outline"
            label="Voir le profil complet"
            size={48}
            tone="textMuted"
            background="surface"
            onPress={() => navigation.navigate("ProfileDetail", { userId: current.userId, source: "discovery" })}
            style={[styles.action, { borderColor: colors.border }]}
          />
          <IconButton
            icon="heart"
            label={`Aimer ${current.firstName}`}
            size={64}
            tone="onPrimary"
            background="primary"
            onPress={() => topCard.current?.swipe("like")}
            testID="swipe-like"
          />
        </View>
      </>
    );
  }

  return (
    <Screen testID="discovery-screen">
      {header}
      {me.data?.profile.isPaused ? (
        <View style={[styles.paused, { backgroundColor: colors.primarySoft }]}>
          <Icon name="pause-circle-outline" size={18} color="primary" />
          <Text variant="caption" color="primary" style={styles.flex}>
            Votre profil est en pause : personne ne vous voit dans la découverte.
          </Text>
        </View>
      ) : null}
      <View style={styles.flex}>{body}</View>
      <MatchModal
        match={match}
        myPhoto={me.data?.profile.photos[0]}
        myName={me.data?.profile.firstName ?? ""}
        onClose={() => setMatch(null)}
        onMessage={(m) => {
          setMatch(null);
          navigation.navigate("Chat", { conversationId: m.conversationId });
        }}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  deck: { flex: 1, marginTop: spacing.xs },
  behind: { transform: [{ scale: 0.95 }, { translateY: 14 }], opacity: 0.6 },
  actions: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "center",
    gap: spacing.lg,
    paddingVertical: spacing.md,
    paddingBottom: spacing.lg
  },
  action: { borderWidth: 1 },
  refresh: { alignSelf: "center", marginBottom: spacing.xl },
  paused: {
    flexDirection: "row",
    gap: spacing.xs,
    alignItems: "center",
    padding: spacing.sm,
    borderRadius: radii.md,
    marginBottom: spacing.xs
  }
});
