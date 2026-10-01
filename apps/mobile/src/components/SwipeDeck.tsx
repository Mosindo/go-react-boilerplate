import React, { useCallback, useMemo, useRef } from "react";
import { Animated, PanResponder, StyleSheet, View, useWindowDimensions } from "react-native";
import type { PublicProfile, SwipeAction } from "../api/types";
import { IconButton, Text, radii, spacing, useTheme } from "../shared/ui";
import { swipeDecision } from "../utils/deck";
import { ProfileCard } from "./ProfileCard";

type SwipeDeckProps = {
  profiles: PublicProfile[];
  onSwipe: (profile: PublicProfile, action: SwipeAction) => void;
  onOpenProfile: (profile: PublicProfile) => void;
  disabled?: boolean;
};

/**
 * Tinder-style stack built on PanResponder + Animated (no native gesture
 * dependency). Swiping is never the only way in: the buttons below the deck
 * perform the same actions for people who cannot drag.
 */
export function SwipeDeck({ profiles, onSwipe, onOpenProfile, disabled = false }: SwipeDeckProps) {
  const { width } = useWindowDimensions();
  const { colors } = useTheme();
  const pan = useRef(new Animated.ValueXY()).current;
  const busy = useRef(false);
  const topRef = useRef<PublicProfile | undefined>(profiles[0]);
  topRef.current = profiles[0];
  const onSwipeRef = useRef(onSwipe);
  onSwipeRef.current = onSwipe;

  const commit = useCallback(
    (action: SwipeAction) => {
      if (busy.current || !topRef.current) {
        return;
      }
      busy.current = true;
      Animated.timing(pan, {
        toValue: { x: action === "like" ? width * 1.3 : -width * 1.3, y: 0 },
        duration: 200,
        useNativeDriver: false
      }).start(() => {
        const profile = topRef.current;
        pan.setValue({ x: 0, y: 0 });
        busy.current = false;
        if (profile) {
          onSwipeRef.current(profile, action);
        }
      });
    },
    [pan, width]
  );

  const responder = useMemo(
    () =>
      PanResponder.create({
        onMoveShouldSetPanResponder: (_, g) => !disabled && Math.abs(g.dx) > 8 && Math.abs(g.dx) > Math.abs(g.dy),
        onPanResponderMove: Animated.event([null, { dx: pan.x, dy: pan.y }], { useNativeDriver: false }),
        onPanResponderRelease: (_, g) => {
          const decision = swipeDecision(g.dx, g.vx, width);
          if (decision) {
            commit(decision);
          } else {
            Animated.spring(pan, { toValue: { x: 0, y: 0 }, useNativeDriver: false }).start();
          }
        },
        onPanResponderTerminate: () => {
          Animated.spring(pan, { toValue: { x: 0, y: 0 }, useNativeDriver: false }).start();
        }
      }),
    [commit, disabled, pan, width]
  );

  const rotate = pan.x.interpolate({ inputRange: [-width, 0, width], outputRange: ["-12deg", "0deg", "12deg"] });
  const likeOpacity = pan.x.interpolate({ inputRange: [20, width * 0.3], outputRange: [0, 1], extrapolate: "clamp" });
  const passOpacity = pan.x.interpolate({ inputRange: [-width * 0.3, -20], outputRange: [1, 0], extrapolate: "clamp" });
  const nextScale = pan.x.interpolate({ inputRange: [-width, 0, width], outputRange: [1, 0.95, 1], extrapolate: "clamp" });

  const top = profiles[0];
  const next = profiles[1];

  return (
    <View style={styles.root}>
      <View style={styles.stack}>
        {next ? (
          <Animated.View key={next.id} pointerEvents="none" style={[styles.cardWrap, { transform: [{ scale: nextScale }] }]}>
            <ProfileCard profile={next} />
          </Animated.View>
        ) : null}
        {top ? (
          <Animated.View
            key={top.id}
            {...responder.panHandlers}
            style={[styles.cardWrap, { transform: [{ translateX: pan.x }, { translateY: pan.y }, { rotate }] }]}
          >
            <ProfileCard profile={top} />
            <Animated.View
              pointerEvents="none"
              style={[styles.stamp, styles.stampLike, { opacity: likeOpacity, borderColor: colors.success }]}
            >
              <Text style={{ color: colors.success }} variant="heading">
                J&apos;AIME
              </Text>
            </Animated.View>
            <Animated.View pointerEvents="none" style={[styles.stamp, styles.stampPass, { opacity: passOpacity, borderColor: "#ffffff" }]}>
              <Text style={styles.stampPassText} variant="heading">
                PASSER
              </Text>
            </Animated.View>
          </Animated.View>
        ) : null}
      </View>

      <View style={styles.actions}>
        <IconButton color={colors.pass} disabled={disabled || !top} icon="close" label="Passer" onPress={() => commit("pass")} size={62} />
        <IconButton disabled={!top} icon="information" label="Voir le profil complet" onPress={() => top && onOpenProfile(top)} size={46} />
        <IconButton
          background={colors.primary}
          color={colors.primaryForeground}
          disabled={disabled || !top}
          icon="heart"
          label="J'aime"
          onPress={() => commit("like")}
          size={62}
        />
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, gap: spacing.md },
  stack: { flex: 1 },
  cardWrap: { ...StyleSheet.absoluteFillObject },
  actions: { flexDirection: "row", alignItems: "center", justifyContent: "center", gap: spacing.xl, paddingBottom: spacing.sm },
  stamp: {
    position: "absolute",
    top: spacing.xl,
    borderWidth: 3,
    borderRadius: radii.md,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.xs
  },
  stampLike: { left: spacing.lg, transform: [{ rotate: "-12deg" }] },
  stampPass: { right: spacing.lg, transform: [{ rotate: "12deg" }] },
  stampPassText: { color: "#ffffff" }
});
