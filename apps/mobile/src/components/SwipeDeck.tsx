import React, { forwardRef, useCallback, useImperativeHandle, useLayoutEffect, useRef } from "react";
import { Animated, Dimensions, PanResponder, StyleSheet, View } from "react-native";
import type { PublicProfile, SwipeAction } from "../api/types";
import { Text, colors, radii, spacing } from "../shared/ui";
import { ProfileHero } from "./ProfileHero";

export type SwipeDeckHandle = {
  /** Same effect as dragging the top card away; used by the like/pass buttons. */
  swipe: (action: SwipeAction) => void;
};

type SwipeDeckProps = {
  profiles: PublicProfile[];
  onSwipe: (profile: PublicProfile, action: SwipeAction) => void;
  onOpen: (profile: PublicProfile) => void;
};

const SCREEN_WIDTH = Dimensions.get("window").width;
const DISTANCE_THRESHOLD = SCREEN_WIDTH * 0.28;
const VELOCITY_THRESHOLD = 0.9;

export const SwipeDeck = forwardRef<SwipeDeckHandle, SwipeDeckProps>(function SwipeDeck({ profiles, onSwipe, onOpen }, ref) {
  const position = useRef(new Animated.ValueXY()).current;
  const top = profiles[0];
  const next = profiles[1];

  const latest = useRef({ top, onSwipe });
  latest.current = { top, onSwipe };
  const busy = useRef(false);

  // The next card takes the top slot centred: reset before the new frame is painted.
  useLayoutEffect(() => {
    position.setValue({ x: 0, y: 0 });
    busy.current = false;
  }, [position, top?.id]);

  const fling = useCallback(
    (action: SwipeAction) => {
      const current = latest.current;
      if (!current.top || busy.current) {
        return;
      }
      busy.current = true;
      Animated.timing(position, {
        toValue: { x: action === "like" ? SCREEN_WIDTH * 1.4 : -SCREEN_WIDTH * 1.4, y: 0 },
        duration: 200,
        useNativeDriver: false
      }).start(() => current.onSwipe(current.top, action));
    },
    [position]
  );
  useImperativeHandle(ref, () => ({ swipe: fling }), [fling]);

  const pan = useRef(
    PanResponder.create({
      onMoveShouldSetPanResponder: (_event, gesture) => Math.abs(gesture.dx) > 8 && Math.abs(gesture.dx) > Math.abs(gesture.dy),
      onPanResponderMove: Animated.event([null, { dx: position.x, dy: position.y }], { useNativeDriver: false }),
      onPanResponderRelease: (_event, gesture) => {
        if (gesture.dx > DISTANCE_THRESHOLD || gesture.vx > VELOCITY_THRESHOLD) {
          fling("like");
        } else if (gesture.dx < -DISTANCE_THRESHOLD || gesture.vx < -VELOCITY_THRESHOLD) {
          fling("pass");
        } else {
          Animated.spring(position, { toValue: { x: 0, y: 0 }, useNativeDriver: false, friction: 6 }).start();
        }
      }
    })
  ).current;

  const rotate = position.x.interpolate({ inputRange: [-SCREEN_WIDTH, 0, SCREEN_WIDTH], outputRange: ["-12deg", "0deg", "12deg"] });
  const likeOpacity = position.x.interpolate({ inputRange: [0, DISTANCE_THRESHOLD], outputRange: [0, 1], extrapolate: "clamp" });
  const passOpacity = position.x.interpolate({ inputRange: [-DISTANCE_THRESHOLD, 0], outputRange: [1, 0], extrapolate: "clamp" });
  const nextScale = position.x.interpolate({
    inputRange: [-DISTANCE_THRESHOLD, 0, DISTANCE_THRESHOLD],
    outputRange: [1, 0.94, 1],
    extrapolate: "clamp"
  });

  return (
    <View style={styles.deck} testID="swipe-deck">
      {next ? (
        <Animated.View pointerEvents="none" style={[styles.card, { transform: [{ scale: nextScale }] }]}>
          <ProfileHero maxInterests={0} profile={next} />
        </Animated.View>
      ) : null}
      {top ? (
        <Animated.View
          {...pan.panHandlers}
          key={top.id}
          style={[styles.card, { transform: [{ translateX: position.x }, { translateY: position.y }, { rotate }] }]}
          testID={`card-${top.id}`}
        >
          <ProfileHero onCenterPress={() => onOpen(top)} profile={top} />
          <Animated.View pointerEvents="none" style={[styles.stamp, styles.likeStamp, { opacity: likeOpacity }]}>
            <Text style={styles.likeText} variant="heading" weight="bold">
              J'AIME
            </Text>
          </Animated.View>
          <Animated.View pointerEvents="none" style={[styles.stamp, styles.passStamp, { opacity: passOpacity }]}>
            <Text style={styles.passText} variant="heading" weight="bold">
              PASSER
            </Text>
          </Animated.View>
        </Animated.View>
      ) : null}
    </View>
  );
});

const styles = StyleSheet.create({
  deck: { flex: 1 },
  card: {
    ...StyleSheet.absoluteFillObject,
    borderRadius: radii.xl,
    shadowColor: "#2A2124",
    shadowOpacity: 0.18,
    shadowRadius: 18,
    shadowOffset: { width: 0, height: 8 },
    elevation: 6,
    backgroundColor: colors.surfaceSubtle
  },
  stamp: {
    position: "absolute",
    top: spacing.xxl,
    borderWidth: 3,
    borderRadius: radii.md,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.xs,
    backgroundColor: "rgba(255,255,255,0.85)"
  },
  likeStamp: { left: spacing.xl, borderColor: colors.like, transform: [{ rotate: "-12deg" }] },
  passStamp: { right: spacing.xl, borderColor: colors.pass, transform: [{ rotate: "12deg" }] },
  likeText: { color: colors.like },
  passText: { color: colors.pass }
});
