import React, { useCallback, useEffect, useRef, useState } from "react";
import { Animated, PanResponder, StyleSheet, View, useWindowDimensions } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import { useFocusEffect, useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { isApiError } from "../../api/client";
import { discoveryApi } from "../../api/dating";
import type { MatchSummary, PublicProfile, SwipeAction } from "../../api/types";
import { MatchCelebration } from "../../components/MatchCelebration";
import { ProfileCard } from "../../components/ProfileCard";
import { markMatchCelebrated } from "../../hooks/presence";
import { keys, useDeck, useMyProfile, useUpdateLocation } from "../../hooks/queries";
import type { AppStackParamList } from "../../navigation/types";
import { EmptyView, ErrorView, LoadingView, showToast } from "../../shared/feedback";
import { Screen } from "../../shared/layout";
import { IconButton, Text, colors, spacing } from "../../shared/ui";
import { errorMessage } from "../../utils/errors";
import { readApproximateLocation } from "../../utils/media";

type Nav = NativeStackNavigationProp<AppStackParamList>;

export default function DiscoverScreen() {
  const navigation = useNavigation<Nav>();
  const client = useQueryClient();
  const profile = useMyProfile();
  const hasLocation = profile.data?.hasLocation ?? false;
  const deck = useDeck(hasLocation);
  const updateLocation = useUpdateLocation();
  const [queue, setQueue] = useState<PublicProfile[]>([]);
  const seen = useRef(new Set<string>());
  const [match, setMatch] = useState<MatchSummary | null>(null);
  const { width } = useWindowDimensions();

  useEffect(() => {
    if (!deck.data) {
      return;
    }
    setQueue((current) => {
      const ids = new Set(current.map((p) => p.userId));
      return [...current, ...deck.data.filter((p) => !ids.has(p.userId) && !seen.current.has(p.userId))];
    });
  }, [deck.data]);

  const queueRef = useRef(queue);
  queueRef.current = queue;
  const refetchDeck = deck.refetch;

  // When coming back to an empty deck, look for new people.
  useFocusEffect(
    useCallback(() => {
      if (hasLocation && queueRef.current.length === 0) {
        void refetchDeck();
      }
    }, [hasLocation, refetchDeck])
  );

  const decide = useCallback(
    (target: PublicProfile, action: SwipeAction) => {
      seen.current.add(target.userId);
      setQueue((current) => current.filter((p) => p.userId !== target.userId));
      discoveryApi
        .swipe(target.userId, action)
        .then((result) => {
          if (result.matched && result.match) {
            markMatchCelebrated(result.match.id);
            setMatch(result.match);
            void client.invalidateQueries({ queryKey: keys.matches });
          }
        })
        .catch((error: unknown) => {
          if (!isApiError(error, "already_swiped") && !isApiError(error, "not_available")) {
            showToast(errorMessage(error), "danger");
          }
        })
        .finally(() => {
          // Refill once the decision is stored, so it is excluded server-side.
          if (queueRef.current.length < 3) {
            void refetchDeck();
          }
        });
    },
    [client, refetchDeck]
  );

  const top = queue[0];
  const position = useRef(new Animated.ValueXY()).current;
  const threshold = Math.min(140, width * 0.28);

  const fling = useCallback(
    (action: SwipeAction) => {
      if (!top) {
        return;
      }
      Animated.timing(position, {
        toValue: { x: (action === "like" ? 1 : -1) * width * 1.3, y: 40 },
        duration: 240,
        useNativeDriver: false
      }).start(() => {
        position.setValue({ x: 0, y: 0 });
        decide(top, action);
      });
    },
    [decide, position, top, width]
  );

  const responder = PanResponder.create({
    onMoveShouldSetPanResponder: (_, g) => Math.abs(g.dx) > 8 && Math.abs(g.dx) > Math.abs(g.dy),
    onPanResponderMove: (_, g) => position.setValue({ x: g.dx, y: g.dy * 0.2 }),
    onPanResponderRelease: (_, g) => {
      if (g.dx > threshold) {
        fling("like");
      } else if (g.dx < -threshold) {
        fling("pass");
      } else {
        Animated.spring(position, { toValue: { x: 0, y: 0 }, friction: 6, useNativeDriver: false }).start();
      }
    },
    onPanResponderTerminate: () => Animated.spring(position, { toValue: { x: 0, y: 0 }, useNativeDriver: false }).start()
  });

  const rotate = position.x.interpolate({ inputRange: [-width, 0, width], outputRange: ["-12deg", "0deg", "12deg"] });
  const likeOpacity = position.x.interpolate({ inputRange: [0, threshold], outputRange: [0, 1], extrapolate: "clamp" });
  const passOpacity = position.x.interpolate({ inputRange: [-threshold, 0], outputRange: [1, 0], extrapolate: "clamp" });

  const locate = async () => {
    const result = await readApproximateLocation();
    if ("error" in result) {
      showToast(result.error, "danger");
      return;
    }
    updateLocation.mutate(result);
  };

  let body: React.ReactNode;
  if (profile.isLoading) {
    body = <LoadingView />;
  } else if (!hasLocation) {
    body = (
      <EmptyView
        actionLabel="Utiliser ma position"
        glyph="◎"
        message="Ta position (arrondie à ~1 km) nous sert à te montrer des personnes près de toi. Personne ne voit ton adresse."
        onAction={() => void locate()}
        testID="discover-location"
        title="Active ta localisation"
      />
    );
  } else if (deck.isError && queue.length === 0) {
    body = <ErrorView message={errorMessage(deck.error)} onRetry={() => void deck.refetch()} />;
  } else if (deck.isLoading && queue.length === 0) {
    body = <LoadingView label="On cherche des profils pour toi…" />;
  } else if (!top) {
    body = (
      <EmptyView
        actionLabel="Élargir mes critères"
        glyph="✦"
        message="Tu as vu tout le monde près de toi pour l'instant. Reviens plus tard ou élargis ta distance et ta tranche d'âge."
        onAction={() => navigation.navigate("Preferences")}
        testID="discover-empty"
        title="Plus personne pour le moment"
      />
    );
  } else {
    body = (
      <>
        <View style={styles.deck}>
          {queue[1] ? (
            <View style={[StyleSheet.absoluteFill, styles.under]}>
              <ProfileCard onOpen={() => undefined} profile={queue[1]} />
            </View>
          ) : null}
          <Animated.View
            key={top.userId}
            style={[StyleSheet.absoluteFill, { transform: [{ translateX: position.x }, { translateY: position.y }, { rotate }] }]}
            {...responder.panHandlers}
          >
            <ProfileCard onOpen={() => navigation.navigate("ProfileDetail", { userId: top.userId, fromDiscover: true })} profile={top} />
            <Animated.View pointerEvents="none" style={[styles.stamp, styles.stampLike, { opacity: likeOpacity }]}>
              <Text tone="primary" variant="heading" weight="bold">
                J&apos;AIME
              </Text>
            </Animated.View>
            <Animated.View pointerEvents="none" style={[styles.stamp, styles.stampPass, { opacity: passOpacity }]}>
              <Text tone="muted" variant="heading" weight="bold">
                PASSER
              </Text>
            </Animated.View>
          </Animated.View>
        </View>
        <View style={styles.actions}>
          <IconButton
            accessibilityLabel={`Passer ${top.firstName}`}
            glyph="✕"
            onPress={() => fling("pass")}
            size={64}
            testID="swipe-pass"
            tone="muted"
          />
          <IconButton
            accessibilityLabel={`Aimer ${top.firstName}`}
            filled
            glyph="♥"
            onPress={() => fling("like")}
            size={72}
            testID="swipe-like"
            tone="primary"
          />
        </View>
      </>
    );
  }

  return (
    <Screen contentStyle={styles.content} scroll={false} testID="discover-screen">
      <View style={styles.header}>
        <Text variant="heading" weight="bold">
          Hearth
        </Text>
        {profile.data && !profile.data.settings.isDiscoverable ? (
          <Text tone="accent" variant="caption">
            Profil en pause · invisible
          </Text>
        ) : null}
        <IconButton accessibilityLabel="Critères de recherche" glyph="⚙" onPress={() => navigation.navigate("Preferences")} size={40} />
      </View>
      {body}
      <MatchCelebration
        match={match}
        myPhoto={profile.data?.photos[0]}
        onClose={() => setMatch(null)}
        onMessage={(m) => {
          setMatch(null);
          navigation.navigate("Chat", {
            conversationId: m.conversationId,
            matchId: m.id,
            userId: m.user.userId,
            firstName: m.user.firstName
          });
        }}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  content: {
    paddingBottom: spacing.lg,
    gap: spacing.md
  },
  header: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    gap: spacing.md
  },
  deck: {
    flex: 1
  },
  under: {
    transform: [{ scale: 0.96 }, { translateY: 10 }],
    opacity: 0.7
  },
  actions: {
    flexDirection: "row",
    justifyContent: "center",
    alignItems: "center",
    gap: spacing.xxl
  },
  stamp: {
    position: "absolute",
    top: spacing.xxl,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.xs,
    borderWidth: 3,
    borderRadius: 10
  },
  stampLike: {
    left: spacing.xl,
    borderColor: colors.primary,
    transform: [{ rotate: "-12deg" }]
  },
  stampPass: {
    right: spacing.xl,
    borderColor: colors.textMuted,
    transform: [{ rotate: "12deg" }]
  }
});
