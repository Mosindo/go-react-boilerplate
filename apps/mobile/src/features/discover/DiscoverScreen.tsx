import { useMutation, useQueryClient } from "@tanstack/react-query";
import React, { useCallback, useEffect, useRef, useState } from "react";
import { AccessibilityInfo, Animated, PanResponder, StyleSheet, View, useWindowDimensions } from "react-native";
import { discoveryApi } from "../../api/discovery";
import type { MatchInfo, PublicProfile, SwipeAction } from "../../api/types";
import type { TabScreenProps } from "../../navigation/types";
import { useMyProfile } from "../../state/queries";
import { queryKeys } from "../../state/queryClient";
import { expectMatchFrom } from "../../state/realtime";
import { useTheme } from "../../theme/ThemeProvider";
import { EmptyState, ErrorState, IconButton, LoadingState, Screen, Text, useToast } from "../../ui";
import { describeError, isApiError } from "../../utils/errors";
import { MatchModal } from "./MatchModal";
import { SwipeCard } from "./SwipeCard";

const SWIPE_THRESHOLD = 110;

type LoadState = "loading" | "ready" | "error" | "incomplete";

export default function DiscoverScreen({ navigation }: TabScreenProps<"Discover">) {
  const { colors, spacing } = useTheme();
  const { width } = useWindowDimensions();
  const toast = useToast();
  const client = useQueryClient();
  const me = useMyProfile();

  const [queue, setQueue] = useState<PublicProfile[]>([]);
  const [state, setState] = useState<LoadState>("loading");
  const [error, setError] = useState<string | null>(null);
  const [match, setMatch] = useState<MatchInfo | null>(null);
  const seen = useRef(new Set<string>());
  const loading = useRef(false);
  const exhausted = useRef(false);
  const reduceMotion = useRef(false);
  const position = useRef(new Animated.ValueXY()).current;

  useEffect(() => {
    void AccessibilityInfo.isReduceMotionEnabled().then((v) => {
      reduceMotion.current = v;
    });
  }, []);

  const load = useCallback(async (reset = false) => {
    if (loading.current) return;
    loading.current = true;
    if (reset) {
      exhausted.current = false;
      setState("loading");
    }
    try {
      const batch = await discoveryApi.next(10);
      const fresh = batch.filter((p) => !seen.current.has(p.userId));
      fresh.forEach((p) => seen.current.add(p.userId));
      exhausted.current = fresh.length === 0;
      setQueue((q) => (reset ? fresh : [...q, ...fresh]));
      setState("ready");
    } catch (e) {
      if (isApiError(e, 409)) {
        setState("incomplete");
      } else {
        setError(describeError(e));
        setState("error");
      }
    } finally {
      loading.current = false;
    }
  }, []);

  useEffect(() => {
    void load(true);
  }, [load]);

  // Prefetch before the deck runs out.
  useEffect(() => {
    if (state === "ready" && queue.length < 3 && !exhausted.current) {
      void load();
    }
  }, [load, queue.length, state]);

  // Reload the deck when search criteria, location or completeness change.
  const criteria = me.data
    ? JSON.stringify([me.data.preferences, me.data.hasLocation, me.data.isComplete, me.data.gender])
    : null;
  const lastCriteria = useRef<string | null>(null);
  useEffect(() => {
    if (criteria === null) return;
    if (lastCriteria.current !== null && lastCriteria.current !== criteria) {
      seen.current.clear();
      void load(true);
    }
    lastCriteria.current = criteria;
  }, [criteria, load]);

  const swipe = useMutation({
    mutationFn: ({ profile, action }: { profile: PublicProfile; action: SwipeAction }) =>
      discoveryApi.swipe(profile.userId, action),
    onSuccess: (result) => {
      if (result.matched && result.match) {
        setMatch(result.match);
        void client.invalidateQueries({ queryKey: queryKeys.conversations });
      }
    },
    onError: (e) => {
      if (!isApiError(e, 404) && !isApiError(e, 409)) {
        toast.show(describeError(e), "error");
      }
    }
  });

  const commit = useCallback(
    (action: SwipeAction) => {
      const top = queue[0];
      if (!top) return;
      if (action === "like") {
        expectMatchFrom(top.userId);
      }
      const finish = () => {
        position.setValue({ x: 0, y: 0 });
        setQueue((q) => q.slice(1));
        swipe.mutate({ profile: top, action });
      };
      if (reduceMotion.current) {
        finish();
        return;
      }
      Animated.timing(position, {
        toValue: { x: (action === "like" ? 1 : -1) * width * 1.3, y: 0 },
        duration: 220,
        useNativeDriver: false
      }).start(finish);
    },
    [position, queue, swipe, width]
  );

  const commitRef = useRef(commit);
  commitRef.current = commit;

  const responder = useRef(
    PanResponder.create({
      onMoveShouldSetPanResponder: (_, g) => Math.abs(g.dx) > 8 && Math.abs(g.dx) > Math.abs(g.dy),
      onPanResponderMove: Animated.event([null, { dx: position.x, dy: position.y }], { useNativeDriver: false }),
      onPanResponderRelease: (_, g) => {
        if (g.dx > SWIPE_THRESHOLD) {
          commitRef.current("like");
        } else if (g.dx < -SWIPE_THRESHOLD) {
          commitRef.current("pass");
        } else {
          Animated.spring(position, { toValue: { x: 0, y: 0 }, friction: 6, useNativeDriver: false }).start();
        }
      },
      onPanResponderTerminate: () => {
        Animated.spring(position, { toValue: { x: 0, y: 0 }, useNativeDriver: false }).start();
      }
    })
  ).current;

  const rotate = position.x.interpolate({ inputRange: [-width, 0, width], outputRange: ["-12deg", "0deg", "12deg"] });
  const likeOpacity = position.x.interpolate({
    inputRange: [0, SWIPE_THRESHOLD],
    outputRange: [0, 1],
    extrapolate: "clamp"
  });
  const passOpacity = position.x.interpolate({
    inputRange: [-SWIPE_THRESHOLD, 0],
    outputRange: [1, 0],
    extrapolate: "clamp"
  });
  const nextScale = position.x.interpolate({
    inputRange: [-width, 0, width],
    outputRange: [1, 0.95, 1],
    extrapolate: "clamp"
  });

  const header = (
    <View style={[styles.header, { paddingVertical: spacing.sm }]}>
      <Text variant="title" tone="primary" accessibilityRole="header">
        Aimant
      </Text>
      <IconButton
        icon="options-outline"
        accessibilityLabel="Préférences de recherche"
        onPress={() => navigation.navigate("Preferences")}
        testID="discover-preferences"
      />
    </View>
  );

  let body: React.ReactNode;
  if (state === "loading") {
    body = <LoadingState label="On cherche des profils pour vous…" />;
  } else if (state === "incomplete") {
    body = (
      <EmptyState
        icon="person-add-outline"
        title="Votre profil est presque prêt"
        message="Ajoutez les informations manquantes (photo, position…) pour découvrir des personnes et être vu·e."
        actionLabel="Compléter mon profil"
        onAction={() => navigation.navigate("CompleteProfile")}
        testID="discover-incomplete"
      />
    );
  } else if (state === "error") {
    body = <ErrorState message={error ?? ""} onAction={() => void load(true)} />;
  } else if (queue.length === 0) {
    body = (
      <EmptyState
        icon="compass-outline"
        title="Vous avez fait le tour"
        message="Revenez plus tard ou élargissez vos critères de distance et d'âge."
        actionLabel="Ajuster mes critères"
        onAction={() => navigation.navigate("Preferences")}
        testID="discover-empty"
      />
    );
  } else {
    const [top, next] = queue;
    body = (
      <View style={styles.flex}>
        <View style={styles.deck} testID="discover-deck">
          {next ? (
            <Animated.View
              style={[StyleSheet.absoluteFill, { transform: [{ scale: nextScale }] }]}
              pointerEvents="none"
            >
              <SwipeCard profile={next} onOpen={() => undefined} />
            </Animated.View>
          ) : null}
          <Animated.View
            key={top.userId}
            {...responder.panHandlers}
            style={[
              StyleSheet.absoluteFill,
              { transform: [{ translateX: position.x }, { translateY: position.y }, { rotate }] }
            ]}
          >
            <SwipeCard profile={top} onOpen={() => navigation.navigate("ProfileDetail", { userId: top.userId })} />
            <Animated.View
              style={[styles.stamp, styles.stampLike, { opacity: likeOpacity, borderColor: colors.accent }]}
              pointerEvents="none"
            >
              <Text variant="heading" weight="bold" style={{ color: colors.accent }}>
                J’AIME
              </Text>
            </Animated.View>
            <Animated.View
              style={[styles.stamp, styles.stampPass, { opacity: passOpacity, borderColor: "#FFFFFF" }]}
              pointerEvents="none"
            >
              <Text variant="heading" weight="bold" style={{ color: "#FFFFFF" }}>
                NON
              </Text>
            </Animated.View>
          </Animated.View>
        </View>
        <View style={[styles.actions, { paddingVertical: spacing.md }]}>
          <IconButton
            icon="close"
            size={64}
            filled
            accessibilityLabel={`Passer ${top.firstName}`}
            onPress={() => commit("pass")}
            testID="discover-pass"
          />
          <IconButton
            icon="heart"
            size={72}
            filled
            tone="primary"
            accessibilityLabel={`Aimer ${top.firstName}`}
            onPress={() => commit("like")}
            testID="discover-like"
          />
        </View>
      </View>
    );
  }

  return (
    <Screen edges={["top"]} testID="discover-screen">
      {header}
      {me.data?.isPaused ? (
        <View style={[styles.banner, { backgroundColor: colors.accentSoft }]}>
          <Text variant="label" tone="accent">
            Votre profil est en pause : personne ne vous voit pour le moment.
          </Text>
        </View>
      ) : null}
      {body}
      <MatchModal
        match={match}
        myName={me.data?.firstName ?? ""}
        myPhotoUrl={me.data?.photos[0]?.url}
        onClose={() => setMatch(null)}
        onMessage={() => {
          const m = match;
          setMatch(null);
          if (m) {
            navigation.navigate("Chat", {
              userId: m.user.userId,
              firstName: m.user.firstName,
              photoUrl: m.user.photos[0]?.url
            });
          }
        }}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  header: { flexDirection: "row", alignItems: "center", justifyContent: "space-between" },
  deck: { flex: 1, maxWidth: 520, width: "100%", alignSelf: "center" },
  actions: { flexDirection: "row", justifyContent: "center", alignItems: "center", gap: 36 },
  stamp: { position: "absolute", top: 36, paddingHorizontal: 14, paddingVertical: 6, borderWidth: 3, borderRadius: 10 },
  stampLike: { left: 24, transform: [{ rotate: "-14deg" }] },
  stampPass: { right: 24, transform: [{ rotate: "14deg" }] },
  banner: { padding: 10, borderRadius: 12, marginBottom: 8 }
});
