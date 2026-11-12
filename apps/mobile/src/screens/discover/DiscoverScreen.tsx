import React, { useCallback, useEffect, useRef, useState } from "react";
import { Animated, PanResponder, Pressable, StyleSheet, View, useWindowDimensions } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useQueryClient } from "@tanstack/react-query";
import { ApiError, errorMessage } from "../../api/client";
import { matchingApi } from "../../api/endpoints";
import type { Match, PublicProfile } from "../../api/types";
import { keys, useOwnProfile } from "../../hooks/queries";
import type { AppStackParamList } from "../../navigation/types";
import { spacing, useTheme } from "../../theme";
import { EmptyState, ErrorState, Icon, Loading, Screen, Text, showToast } from "../../ui";
import { MatchModal } from "./MatchModal";
import { ProfileCard } from "./ProfileCard";

const SWIPE_THRESHOLD = 110;
const REFILL_AT = 3;

type Decision = "like" | "pass";

export default function DiscoverScreen() {
  const { colors } = useTheme();
  const navigation = useNavigation<NativeStackNavigationProp<AppStackParamList>>();
  const qc = useQueryClient();
  const profile = useOwnProfile();
  const { width } = useWindowDimensions();
  const [deck, setDeck] = useState<PublicProfile[]>([]);
  const [status, setStatus] = useState<"loading" | "ready" | "error">("loading");
  const [error, setError] = useState<string>("");
  const [exhausted, setExhausted] = useState(false);
  const [match, setMatch] = useState<Match | null>(null);
  const fetching = useRef(false);
  const position = useRef(new Animated.ValueXY()).current;
  const deckRef = useRef<PublicProfile[]>([]);
  deckRef.current = deck;

  const load = useCallback(async (reset = false) => {
    if (fetching.current) return;
    fetching.current = true;
    if (reset) setStatus("loading");
    try {
      const current = reset ? [] : deckRef.current;
      const next = await matchingApi.discover(
        current.map((p) => p.userId),
        10
      );
      setDeck((d) => {
        const base = reset ? [] : d;
        const seen = new Set(base.map((p) => p.userId));
        return [...base, ...next.filter((p) => !seen.has(p.userId))];
      });
      setExhausted(next.length === 0);
      setStatus("ready");
    } catch (e) {
      if (reset || deckRef.current.length === 0) {
        setError(errorMessage(e));
        setStatus("error");
      }
    } finally {
      fetching.current = false;
    }
  }, []);

  useEffect(() => {
    void load(true);
  }, [load]);

  useEffect(() => {
    if (status === "ready" && !exhausted && deck.length > 0 && deck.length <= REFILL_AT) void load();
  }, [deck.length, exhausted, load, status]);

  const commit = useCallback(
    async (target: PublicProfile, decision: Decision) => {
      setDeck((d) => d.filter((p) => p.userId !== target.userId));
      position.setValue({ x: 0, y: 0 });
      try {
        const res = await matchingApi.swipe(target.userId, decision);
        if (res.matched && res.match) {
          setMatch(res.match);
          void qc.invalidateQueries({ queryKey: keys.matches });
        }
      } catch (e) {
        if (e instanceof ApiError && (e.status === 409 || e.status === 404)) return;
        showToast(errorMessage(e), "error");
        setDeck((d) => [target, ...d]);
      }
    },
    [position, qc]
  );

  const animateOut = useCallback(
    (decision: Decision) => {
      const top = deckRef.current[0];
      if (!top) return;
      Animated.timing(position, { toValue: { x: (decision === "like" ? 1 : -1) * width * 1.3, y: 30 }, duration: 220, useNativeDriver: true }).start(
        () => void commit(top, decision)
      );
    },
    [commit, position, width]
  );

  const animateOutRef = useRef(animateOut);
  animateOutRef.current = animateOut;
  const pan = useRef(
    PanResponder.create({
      onMoveShouldSetPanResponder: (_, g) => Math.abs(g.dx) > 8 && Math.abs(g.dx) > Math.abs(g.dy),
      onPanResponderMove: (_, g) => position.setValue({ x: g.dx, y: g.dy * 0.2 }),
      onPanResponderRelease: (_, g) => {
        if (g.dx > SWIPE_THRESHOLD) animateOutRef.current("like");
        else if (g.dx < -SWIPE_THRESHOLD) animateOutRef.current("pass");
        else Animated.spring(position, { toValue: { x: 0, y: 0 }, useNativeDriver: true, friction: 6 }).start();
      },
      onPanResponderTerminate: () => Animated.spring(position, { toValue: { x: 0, y: 0 }, useNativeDriver: true }).start()
    })
  ).current;

  const rotate = position.x.interpolate({ inputRange: [-width, 0, width], outputRange: ["-12deg", "0deg", "12deg"] });
  const likeOpacity = position.x.interpolate({ inputRange: [0, SWIPE_THRESHOLD], outputRange: [0, 1], extrapolate: "clamp" });
  const passOpacity = position.x.interpolate({ inputRange: [-SWIPE_THRESHOLD, 0], outputRange: [1, 0], extrapolate: "clamp" });
  const nextScale = position.x.interpolate({ inputRange: [-width, 0, width], outputRange: [1, 0.95, 1], extrapolate: "clamp" });

  const top = deck[0];
  const next = deck[1];

  return (
    <Screen>
      <View style={styles.header}>
        <Text variant="title" serif accessibilityRole="header">
          Lueur
        </Text>
      </View>
      <View style={styles.deck}>
        {status === "loading" ? (
          <Loading label="On cherche des personnes pour vous…" />
        ) : status === "error" ? (
          <ErrorState message={error} onRetry={() => void load(true)} />
        ) : !top ? (
          <EmptyState
            icon="sparkles-outline"
            title="Vous avez fait le tour"
            message="Revenez un peu plus tard, ou élargissez vos critères d'âge et de distance."
            actionLabel="Modifier mes critères"
            onAction={() => navigation.navigate("EditPreferences")}
          />
        ) : (
          <>
            {next ? (
              <Animated.View style={[styles.cardWrap, { transform: [{ scale: nextScale }] }]} importantForAccessibility="no-hide-descendants">
                <ProfileCard profile={next} active={false} onOpen={() => undefined} />
              </Animated.View>
            ) : null}
            <Animated.View
              key={top.userId}
              style={[styles.cardWrap, { transform: [{ translateX: position.x }, { translateY: position.y }, { rotate }] }]}
              {...pan.panHandlers}
              testID="discover-card"
            >
              <ProfileCard profile={top} active onOpen={() => navigation.navigate("ProfileView", { userId: top.userId })} />
              <Animated.View style={[styles.stamp, styles.stampLike, { opacity: likeOpacity, borderColor: colors.primary }]} pointerEvents="none">
                <Text variant="heading" style={{ color: colors.primary }}>
                  J’AIME
                </Text>
              </Animated.View>
              <Animated.View style={[styles.stamp, styles.stampPass, { opacity: passOpacity, borderColor: "#FFFFFF" }]} pointerEvents="none">
                <Text variant="heading" tone="onPhoto">
                  PASSER
                </Text>
              </Animated.View>
            </Animated.View>
          </>
        )}
      </View>
      {top ? (
        <View style={styles.actions}>
          <ActionButton icon="close" label={`Passer ${top.firstName}`} onPress={() => animateOut("pass")} testID="discover-pass" />
          <ActionButton icon="heart" label={`J'aime ${top.firstName}`} primary onPress={() => animateOut("like")} testID="discover-like" />
        </View>
      ) : null}
      <MatchModal
        match={match}
        myPhoto={profile.data?.photos[0]?.url}
        onClose={() => setMatch(null)}
        onMessage={() => {
          const m = match;
          setMatch(null);
          if (m) navigation.navigate("Chat", { userId: m.user.userId, firstName: m.user.firstName });
        }}
      />
    </Screen>
  );
}

function ActionButton({
  icon,
  label,
  onPress,
  primary,
  testID
}: {
  icon: "close" | "heart";
  label: string;
  onPress: () => void;
  primary?: boolean;
  testID: string;
}) {
  const { colors } = useTheme();
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      onPress={onPress}
      testID={testID}
      style={({ pressed }) => [
        styles.action,
        {
          width: primary ? 76 : 64,
          height: primary ? 76 : 64,
          backgroundColor: primary ? colors.primary : colors.surface,
          borderColor: primary ? colors.primary : colors.border,
          transform: [{ scale: pressed ? 0.93 : 1 }]
        }
      ]}
    >
      <Icon name={icon} size={primary ? 36 : 30} color={primary ? colors.onPrimary : colors.textMuted} />
    </Pressable>
  );
}

const styles = StyleSheet.create({
  header: { flexDirection: "row", alignItems: "center", justifyContent: "space-between", paddingVertical: spacing.xs },
  deck: { flex: 1, marginTop: spacing.xs },
  cardWrap: { ...StyleSheet.absoluteFillObject },
  stamp: { position: "absolute", top: 36, paddingHorizontal: 14, paddingVertical: 6, borderWidth: 3, borderRadius: 10 },
  stampLike: { left: 24, transform: [{ rotate: "-14deg" }] },
  stampPass: { right: 24, transform: [{ rotate: "14deg" }] },
  actions: { flexDirection: "row", justifyContent: "center", alignItems: "center", gap: spacing.xxl, paddingVertical: spacing.md },
  action: {
    borderRadius: 999,
    borderWidth: 1,
    alignItems: "center",
    justifyContent: "center",
    shadowColor: "#000",
    shadowOpacity: 0.1,
    shadowRadius: 12,
    shadowOffset: { width: 0, height: 6 },
    elevation: 4
  }
});
