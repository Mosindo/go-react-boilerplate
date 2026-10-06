import React, { forwardRef, useCallback, useImperativeHandle, useLayoutEffect, useMemo, useRef } from "react";
import { Animated, Dimensions, PanResponder, StyleSheet, View } from "react-native";
import type { PublicProfile, SwipeAction } from "../api/types";
import { Text, radii, spacing, useTheme } from "../shared/ui";
import { ProfileCard } from "./ProfileCard";

const SCREEN_WIDTH = Dimensions.get("window").width;
const SWIPE_THRESHOLD = SCREEN_WIDTH * 0.28;
const EXIT_DURATION = 220;

export type SwipeDeckHandle = {
  swipe: (action: SwipeAction) => void;
};

type SwipeDeckProps = {
  cards: PublicProfile[];
  onSwipe: (profile: PublicProfile, action: SwipeAction) => void;
  onOpenProfile: (profile: PublicProfile) => void;
};

/**
 * Two stacked cards. The top one follows the finger; release past the
 * threshold (or a fast flick) throws it away. Buttons call swipe() so the
 * deck is fully usable without gestures.
 */
export const SwipeDeck = forwardRef<SwipeDeckHandle, SwipeDeckProps>(function SwipeDeck({ cards, onSwipe, onOpenProfile }, ref) {
  const { colors } = useTheme();
  const position = useRef(new Animated.ValueXY()).current;
  const busy = useRef(false);
  const top = cards[0];
  const next = cards[1];

  const latest = useRef({ top, onSwipe });
  latest.current = { top, onSwipe };

  const exit = useCallback(
    (action: SwipeAction) => {
      const current = latest.current.top;
      if (!current || busy.current) {
        return;
      }
      busy.current = true;
      Animated.timing(position, {
        toValue: { x: (action === "like" ? 1 : -1) * SCREEN_WIDTH * 1.4, y: 40 },
        duration: EXIT_DURATION,
        useNativeDriver: true
      }).start(() => {
        latest.current.onSwipe(current, action);
      });
    },
    [position]
  );

  // Reset before paint once the next card becomes the top one.
  useLayoutEffect(() => {
    position.setValue({ x: 0, y: 0 });
    busy.current = false;
  }, [top?.userId, position]);

  useImperativeHandle(ref, () => ({ swipe: exit }), [exit]);

  const panResponder = useMemo(
    () =>
      PanResponder.create({
        onMoveShouldSetPanResponder: (_, g) => !busy.current && (Math.abs(g.dx) > 8 || Math.abs(g.dy) > 8),
        onPanResponderMove: Animated.event([null, { dx: position.x, dy: position.y }], { useNativeDriver: false }),
        onPanResponderRelease: (_, g) => {
          if (g.dx > SWIPE_THRESHOLD || g.vx > 0.9) {
            exit("like");
          } else if (g.dx < -SWIPE_THRESHOLD || g.vx < -0.9) {
            exit("pass");
          } else {
            Animated.spring(position, { toValue: { x: 0, y: 0 }, friction: 6, useNativeDriver: false }).start();
          }
        },
        onPanResponderTerminate: () => {
          Animated.spring(position, { toValue: { x: 0, y: 0 }, useNativeDriver: false }).start();
        }
      }),
    [exit, position]
  );

  if (!top) {
    return null;
  }

  const rotate = position.x.interpolate({ inputRange: [-SCREEN_WIDTH, 0, SCREEN_WIDTH], outputRange: ["-14deg", "0deg", "14deg"] });
  const likeOpacity = position.x.interpolate({ inputRange: [0, SWIPE_THRESHOLD], outputRange: [0, 1], extrapolate: "clamp" });
  const passOpacity = position.x.interpolate({ inputRange: [-SWIPE_THRESHOLD, 0], outputRange: [1, 0], extrapolate: "clamp" });
  const nextScale = position.x.interpolate({
    inputRange: [-SWIPE_THRESHOLD, 0, SWIPE_THRESHOLD],
    outputRange: [1, 0.94, 1],
    extrapolate: "clamp"
  });

  return (
    <View style={styles.root}>
      {next ? (
        <Animated.View key={next.userId} pointerEvents="none" style={[styles.card, { transform: [{ scale: nextScale }] }]}>
          <ProfileCard onOpenProfile={() => undefined} profile={next} />
        </Animated.View>
      ) : null}
      <Animated.View
        {...panResponder.panHandlers}
        key={top.userId}
        style={[styles.card, { transform: [{ translateX: position.x }, { translateY: position.y }, { rotate }] }]}
      >
        <ProfileCard onOpenProfile={() => onOpenProfile(top)} profile={top} />
        <Animated.View pointerEvents="none" style={[styles.stamp, styles.stampLike, { borderColor: colors.like, opacity: likeOpacity }]}>
          <Text style={{ color: colors.like }} variant="heading">
            J'AIME
          </Text>
        </Animated.View>
        <Animated.View pointerEvents="none" style={[styles.stamp, styles.stampPass, { borderColor: colors.pass, opacity: passOpacity }]}>
          <Text style={{ color: colors.pass }} variant="heading">
            PASSER
          </Text>
        </Animated.View>
      </Animated.View>
    </View>
  );
});

const styles = StyleSheet.create({
  root: { flex: 1 },
  card: { ...StyleSheet.absoluteFillObject },
  stamp: {
    position: "absolute",
    top: spacing.xxl,
    borderWidth: 3,
    borderRadius: radii.md,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.xs,
    backgroundColor: "rgba(255,255,255,0.85)"
  },
  stampLike: { left: spacing.xl, transform: [{ rotate: "-12deg" }] },
  stampPass: { right: spacing.xl, transform: [{ rotate: "12deg" }] }
});
