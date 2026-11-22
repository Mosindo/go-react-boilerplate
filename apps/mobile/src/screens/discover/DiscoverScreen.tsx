import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { StyleSheet, View } from "react-native";
import { Image } from "expo-image";
import { useFocusEffect, useNavigation, type CompositeNavigationProp } from "@react-navigation/native";
import type { BottomTabNavigationProp } from "@react-navigation/bottom-tabs";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { ApiError, errorMessage, mediaUrl } from "../../api/client";
import { getDiscovery, resetPasses, swipe } from "../../api/dating";
import type { Match, ProfileCard, SwipeAction } from "../../api/types";
import { Logo } from "../../components/Logo";
import { profileDecided } from "../../lib/events";
import { MatchModal } from "../../components/MatchModal";
import { SwipeCard, type SwipeCardHandle } from "../../components/SwipeCard";
import { qk, useMyProfile, useNotifications } from "../../hooks/queries";
import type { MainStackParamList, TabParamList } from "../../navigation/types";
import { EmptyState, ErrorState, IconButton, LoadingState, Screen, Text, colors, radii, showToast, spacing } from "../../ui";

type Nav = CompositeNavigationProp<BottomTabNavigationProp<TabParamList, "Discover">, NativeStackNavigationProp<MainStackParamList>>;

export default function DiscoverScreen() {
  const navigation = useNavigation<Nav>();
  const client = useQueryClient();
  const me = useMyProfile();
  const notifications = useNotifications();
  const discovery = useQuery({ queryKey: qk.discovery, queryFn: () => getDiscovery(10), staleTime: 0 });
  const [deck, setDeck] = useState<ProfileCard[]>([]);
  const [match, setMatch] = useState<Match | null>(null);
  const seen = useRef(new Set<string>());
  const topCard = useRef<SwipeCardHandle>(null);

  // Merge freshly fetched candidates into the local deck, never re-adding seen ones.
  useEffect(() => {
    const fresh = discovery.data?.profiles ?? [];
    if (fresh.length === 0) {
      return;
    }
    setDeck((current) => {
      const ids = new Set(current.map((c) => c.userId));
      const additions = fresh.filter((c) => !ids.has(c.userId) && !seen.current.has(c.userId));
      return additions.length ? [...current, ...additions] : current;
    });
  }, [discovery.data]);

  // Decisions taken elsewhere (profile detail, block, report) leave the deck too.
  useEffect(
    () =>
      profileDecided.subscribe((userId) => {
        seen.current.add(userId);
        setDeck((current) => current.filter((c) => c.userId !== userId));
      }),
    []
  );

  // Prefetch upcoming photos so the next card appears instantly.
  useEffect(() => {
    const urls = deck.slice(0, 3).flatMap((c) => c.photos.slice(0, 2).map((p) => mediaUrl(p.url)));
    if (urls.length) {
      void Image.prefetch(urls);
    }
  }, [deck]);

  const refill = useCallback(() => {
    if (!discovery.isFetching) {
      void discovery.refetch();
    }
  }, [discovery]);

  useFocusEffect(
    useCallback(() => {
      if (deck.length === 0) {
        refill();
      }
      // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [])
  );

  const sharedInterests = useMemo(() => new Set(me.data?.interests.map((i) => i.slug) ?? []), [me.data]);

  const onSwiped = async (card: ProfileCard, action: SwipeAction) => {
    seen.current.add(card.userId);
    setDeck((current) => current.filter((c) => c.userId !== card.userId));
    if (deck.length <= 3) {
      refill();
    }
    try {
      const result = await swipe(card.userId, action);
      if (result.matched && result.match) {
        setMatch(result.match);
        void client.invalidateQueries({ queryKey: qk.chats });
      }
      if (action === "like") {
        void client.invalidateQueries({ queryKey: qk.likes });
      }
    } catch (err) {
      if (err instanceof ApiError && (err.status === 409 || err.status === 404)) {
        return;
      }
      seen.current.delete(card.userId);
      setDeck((current) => [card, ...current]);
      showToast(errorMessage(err), "error");
    }
  };

  const unread = notifications.data?.unreadCount ?? 0;
  const status = discovery.data?.status;
  const top = deck[0];
  const next = deck[1];

  let body: React.ReactNode;
  if (discovery.isPending && deck.length === 0) {
    body = <LoadingState label="Recherche de profils près de vous…" />;
  } else if (discovery.isError && deck.length === 0) {
    body = <ErrorState message={errorMessage(discovery.error)} onAction={refill} />;
  } else if (status === "incomplete_profile") {
    body = (
      <EmptyState
        actionLabel="Compléter mon profil"
        icon="person-add-outline"
        message="Ajoutez au moins une photo et votre position pour découvrir des profils et être découvert·e."
        onAction={() => navigation.navigate("Onboarding")}
        testID="discover-incomplete"
        title="Votre profil n'est pas encore prêt"
      />
    );
  } else if (status === "under_review") {
    body = (
      <EmptyState
        icon="shield-checkmark-outline"
        message="Suite à des signalements, votre profil est temporairement masqué le temps d'une vérification."
        title="Profil en cours de vérification"
      />
    );
  } else if (!top) {
    body = (
      <EmptyState
        actionLabel="Revoir les profils passés"
        icon="moon-outline"
        message="Vous avez vu tout le monde autour de vous. Élargissez vos critères ou revenez un peu plus tard."
        onAction={async () => {
          try {
            const { restored } = await resetPasses();
            seen.current.clear();
            showToast(restored > 0 ? `${restored} profil(s) de retour` : "Aucun profil passé à revoir");
            refill();
          } catch (err) {
            showToast(errorMessage(err), "error");
          }
        }}
        testID="discover-empty"
        title="Plus personne pour le moment"
      />
    );
  } else {
    body = (
      <>
        <View style={styles.deck}>
          {next ? (
            <View pointerEvents="none" style={[styles.nextCard]}>
              <Image
                cachePolicy="memory-disk"
                contentFit="cover"
                source={{ uri: mediaUrl(next.photos[0]?.url ?? "") }}
                style={StyleSheet.absoluteFill}
              />
            </View>
          ) : null}
          <SwipeCard
            key={top.userId}
            onOpen={() => navigation.navigate("ProfileDetail", { userId: top.userId, context: "discover" })}
            onSwiped={(action) => void onSwiped(top, action)}
            profile={top}
            ref={topCard}
            sharedInterests={sharedInterests}
          />
        </View>
        <View style={styles.actions}>
          <IconButton
            background={colors.surfaceRaised}
            color={colors.pass}
            icon="close"
            label={`Passer ${top.firstName}`}
            onPress={() => topCard.current?.swipe("pass")}
            size={64}
            testID="discover-pass"
          />
          <IconButton
            icon="information"
            label={`Voir le profil de ${top.firstName}`}
            onPress={() => navigation.navigate("ProfileDetail", { userId: top.userId, context: "discover" })}
            size={48}
            testID="discover-info"
          />
          <IconButton
            background={colors.likeSoft}
            color={colors.like}
            icon="heart"
            label={`Liker ${top.firstName}`}
            onPress={() => topCard.current?.swipe("like")}
            size={64}
            testID="discover-like"
          />
        </View>
      </>
    );
  }

  return (
    <Screen testID="discover-screen">
      <View style={styles.header}>
        <Logo size={28} />
        <View style={styles.headerActions}>
          <View>
            <IconButton
              icon="notifications-outline"
              label="Notifications"
              onPress={() => navigation.navigate("Notifications")}
              testID="open-notifications"
            />
            {unread > 0 ? <View style={styles.dot} /> : null}
          </View>
          <IconButton icon="options-outline" label="Préférences de découverte" onPress={() => navigation.navigate("Preferences")} />
        </View>
      </View>
      {status === "paused" ? (
        <View style={styles.banner}>
          <Text tone="muted" variant="caption">
            Votre profil est en pause : vous pouvez découvrir, mais personne ne vous voit.
          </Text>
        </View>
      ) : null}
      {body}
      <MatchModal
        match={match}
        myName={me.data?.firstName ?? ""}
        myPhotoUrl={me.data?.photos[0]?.url}
        onClose={() => setMatch(null)}
        onMessage={(m) => {
          setMatch(null);
          navigation.navigate("ChatThread", { userId: m.user.userId, firstName: m.user.firstName, photoUrl: m.user.photos[0]?.url });
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
    marginBottom: spacing.md
  },
  headerActions: {
    flexDirection: "row",
    gap: spacing.xs
  },
  dot: {
    position: "absolute",
    top: 6,
    right: 6,
    width: 10,
    height: 10,
    borderRadius: 5,
    backgroundColor: colors.accent,
    borderWidth: 2,
    borderColor: colors.background
  },
  banner: {
    padding: spacing.sm,
    borderRadius: radii.md,
    backgroundColor: colors.surface,
    marginBottom: spacing.sm
  },
  deck: {
    flex: 1,
    minHeight: 360
  },
  nextCard: {
    ...StyleSheet.absoluteFillObject,
    borderRadius: radii.xl,
    overflow: "hidden",
    transform: [{ scale: 0.95 }, { translateY: 12 }],
    opacity: 0.55,
    backgroundColor: colors.surfaceHigh
  },
  actions: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "center",
    gap: spacing.xl,
    paddingVertical: spacing.md
  }
});
