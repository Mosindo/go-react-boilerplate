import React, { useCallback, useEffect, useMemo, useRef } from "react";
import { Animated, PanResponder, Platform, StyleSheet, View, useWindowDimensions } from "react-native";
import type { PublicProfile, SwipeAction } from "../api/types";
import { IconButton, Text, radii, spacing } from "../shared/ui";
import { useTheme } from "../theme";
import { ProfileCard } from "./ProfileCard";

const SWIPE_DISTANCE = 110;
const SWIPE_VELOCITY = 0.6;
const FLY_OUT_MS = 180;

type SwipeDeckProps = {
  cards: PublicProfile[];
  onSwipe: (card: PublicProfile, action: SwipeAction) => void;
  onOpen: (card: PublicProfile) => void;
};

/**
 * Stack of profile cards. Drag the top card right to like and left to pass, or use the buttons / arrow keys.
 * Only the top card is interactive; the next one is pre-rendered underneath so the transition is instant.
 */
export function SwipeDeck({ cards, onSwipe, onOpen }: SwipeDeckProps) {
  const { colors } = useTheme();
  const { width: screenWidth } = useWindowDimensions();
  const position = useRef(new Animated.ValueXY()).current;
  const native = Platform.OS !== "web";
  const top = cards[0];
  const next = cards[1];

  const latest = useRef({ top, onSwipe, onOpen });
  latest.current = { top, onSwipe, onOpen };
  const animating = useRef(false);

  const flyOut = useCallback(
    (action: SwipeAction) => {
      const current = latest.current.top;
      if (!current || animating.current) return;
      animating.current = true;
      Animated.timing(position, {
        toValue: { x: (action === "like" ? 1 : -1) * screenWidth * 1.3, y: 40 },
        duration: FLY_OUT_MS,
        useNativeDriver: native
      }).start(() => {
        position.setValue({ x: 0, y: 0 });
        animating.current = false;
        latest.current.onSwipe(current, action);
      });
    },
    [native, position, screenWidth]
  );

  const panResponder = useMemo(
    () =>
      PanResponder.create({
        onMoveShouldSetPanResponder: (_, g) => Math.abs(g.dx) > 6 && Math.abs(g.dx) > Math.abs(g.dy),
        onPanResponderMove: Animated.event([null, { dx: position.x, dy: position.y }], { useNativeDriver: false }),
        onPanResponderRelease: (_, g) => {
          if (g.dx > SWIPE_DISTANCE || g.vx > SWIPE_VELOCITY) flyOut("like");
          else if (g.dx < -SWIPE_DISTANCE || g.vx < -SWIPE_VELOCITY) flyOut("pass");
          else Animated.spring(position, { toValue: { x: 0, y: 0 }, friction: 6, useNativeDriver: false }).start();
        },
        onPanResponderTerminate: () => Animated.spring(position, { toValue: { x: 0, y: 0 }, useNativeDriver: false }).start()
      }),
    [flyOut, position]
  );

  useEffect(() => {
    if (Platform.OS !== "web") return;
    const onKey = (event: KeyboardEvent) => {
      const target = event.target as HTMLElement | null;
      if (target && ["INPUT", "TEXTAREA"].includes(target.tagName)) return;
      if (event.key === "ArrowRight") flyOut("like");
      if (event.key === "ArrowLeft") flyOut("pass");
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [flyOut]);

  const rotate = position.x.interpolate({ inputRange: [-screenWidth, 0, screenWidth], outputRange: ["-14deg", "0deg", "14deg"] });
  const likeOpacity = position.x.interpolate({ inputRange: [20, SWIPE_DISTANCE], outputRange: [0, 1], extrapolate: "clamp" });
  const passOpacity = position.x.interpolate({ inputRange: [-SWIPE_DISTANCE, -20], outputRange: [1, 0], extrapolate: "clamp" });
  const nextScale = position.x.interpolate({ inputRange: [-SWIPE_DISTANCE, 0, SWIPE_DISTANCE], outputRange: [1, 0.94, 1], extrapolate: "clamp" });

  if (!top) return null;

  return (
    <View style={styles.wrap}>
      <View style={styles.stack}>
        {next ? (
          <Animated.View pointerEvents="none" style={[styles.layer, { transform: [{ scale: nextScale }] }]}>
            <ProfileCard interactive={false} profile={next} />
          </Animated.View>
        ) : null}
        <Animated.View
          {...panResponder.panHandlers}
          style={[styles.layer, { transform: [{ translateX: position.x }, { translateY: position.y }, { rotate }] }]}
        >
          <ProfileCard onOpen={() => latest.current.onOpen(top)} profile={top} />
          <Animated.View pointerEvents="none" style={[styles.stamp, styles.stampLike, { opacity: likeOpacity, borderColor: colors.like }]}>
            <Text style={{ color: colors.like }} variant="heading">
              J'AIME
            </Text>
          </Animated.View>
          <Animated.View pointerEvents="none" style={[styles.stamp, styles.stampPass, { opacity: passOpacity, borderColor: "#FFFFFF" }]}>
            <Text style={styles.passText} variant="heading">
              PASSER
            </Text>
          </Animated.View>
        </Animated.View>
      </View>

      <View style={styles.actions}>
        <IconButton
          background={colors.surface}
          color={colors.pass}
          icon="close"
          label="Passer"
          onPress={() => flyOut("pass")}
          size={64}
          style={[styles.actionButton, { borderColor: colors.border }]}
        />
        <IconButton
          background={colors.primary}
          color={colors.onPrimary}
          icon="heart"
          label="J'aime"
          onPress={() => flyOut("like")}
          size={72}
          style={styles.actionButton}
        />
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { flex: 1, gap: spacing.lg },
  stack: { flex: 1 },
  layer: { ...StyleSheet.absoluteFillObject },
  actions: { flexDirection: "row", justifyContent: "center", alignItems: "center", gap: spacing.xxl, paddingBottom: spacing.sm },
  actionButton: {
    borderWidth: 1,
    borderColor: "transparent",
    shadowColor: "#2B1B24",
    shadowOpacity: 0.18,
    shadowRadius: 12,
    shadowOffset: { width: 0, height: 6 },
    elevation: 6
  },
  stamp: { position: "absolute", top: 48, borderWidth: 3, borderRadius: radii.md, paddingHorizontal: spacing.md, paddingVertical: 2 },
  stampLike: { left: 24, transform: [{ rotate: "-12deg" }] },
  stampPass: { right: 24, transform: [{ rotate: "12deg" }] },
  passText: { color: "#FFFFFF" }
});
