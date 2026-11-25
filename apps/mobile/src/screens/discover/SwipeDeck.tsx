import React, { forwardRef, useImperativeHandle, useMemo, useRef } from "react";
import { Animated, PanResponder, StyleSheet, useWindowDimensions, View } from "react-native";
import type { PublicProfile, SwipeAction } from "../../api/types";
import { colors, radii, spacing } from "../../theme";
import { Text } from "../../ui";
import { ProfileCard } from "./ProfileCard";

export type SwipeDeckHandle = {
  swipe: (action: SwipeAction) => void;
};

type SwipeDeckProps = {
  profiles: PublicProfile[];
  onSwiped: (profile: PublicProfile, action: SwipeAction) => void;
  onOpen: (profile: PublicProfile) => void;
};

const SWIPE_THRESHOLD = 110;

/**
 * Card stack with a draggable top card. Gestures use the built-in
 * PanResponder + Animated (native driver) so no extra dependency is needed;
 * buttons call swipe() for an accessible alternative.
 */
export const SwipeDeck = forwardRef<SwipeDeckHandle, SwipeDeckProps>(function SwipeDeck({ profiles, onSwiped, onOpen }, ref) {
  const { width } = useWindowDimensions();
  const position = useRef(new Animated.ValueXY()).current;
  const animating = useRef(false);
  const top = profiles[0];
  const next = profiles[1];

  const finish = (action: SwipeAction) => {
    if (!top || animating.current) {
      return;
    }
    animating.current = true;
    const toX = (action === "like" ? 1 : -1) * (width + 200);
    Animated.timing(position, { toValue: { x: toX, y: 40 }, duration: 220, useNativeDriver: true }).start(() => {
      position.setValue({ x: 0, y: 0 });
      animating.current = false;
      onSwiped(top, action);
    });
  };

  const finishRef = useRef(finish);
  finishRef.current = finish;

  useImperativeHandle(ref, () => ({ swipe: (action) => finishRef.current(action) }));

  const responder = useMemo(
    () =>
      PanResponder.create({
        onMoveShouldSetPanResponder: (_, g) => Math.abs(g.dx) > 8 && Math.abs(g.dx) > Math.abs(g.dy),
        onPanResponderMove: (_, g) => position.setValue({ x: g.dx, y: g.dy * 0.3 }),
        onPanResponderRelease: (_, g) => {
          if (g.dx > SWIPE_THRESHOLD || g.vx > 0.8) {
            finishRef.current("like");
          } else if (g.dx < -SWIPE_THRESHOLD || g.vx < -0.8) {
            finishRef.current("pass");
          } else {
            Animated.spring(position, { toValue: { x: 0, y: 0 }, friction: 6, useNativeDriver: true }).start();
          }
        },
        onPanResponderTerminate: () => {
          Animated.spring(position, { toValue: { x: 0, y: 0 }, friction: 6, useNativeDriver: true }).start();
        }
      }),
    [position]
  );

  const rotate = position.x.interpolate({ inputRange: [-width, 0, width], outputRange: ["-12deg", "0deg", "12deg"] });
  const likeOpacity = position.x.interpolate({ inputRange: [0, SWIPE_THRESHOLD], outputRange: [0, 1], extrapolate: "clamp" });
  const passOpacity = position.x.interpolate({ inputRange: [-SWIPE_THRESHOLD, 0], outputRange: [1, 0], extrapolate: "clamp" });
  const nextScale = position.x.interpolate({ inputRange: [-width, 0, width], outputRange: [1, 0.95, 1], extrapolate: "clamp" });

  return (
    <View style={styles.deck}>
      {next ? (
        <Animated.View key={next.userId} style={[styles.card, { transform: [{ scale: nextScale }] }]} pointerEvents="none">
          <ProfileCard profile={next} onOpen={() => undefined} active={false} />
        </Animated.View>
      ) : null}
      {top ? (
        <Animated.View
          key={top.userId}
          {...responder.panHandlers}
          style={[styles.card, { transform: [{ translateX: position.x }, { translateY: position.y }, { rotate }] }]}
        >
          <ProfileCard profile={top} onOpen={() => onOpen(top)} active />
          <Animated.View style={[styles.stamp, styles.likeStamp, { opacity: likeOpacity }]} pointerEvents="none">
            <Text variant="heading" color="success">
              J’AIME
            </Text>
          </Animated.View>
          <Animated.View style={[styles.stamp, styles.passStamp, { opacity: passOpacity }]} pointerEvents="none">
            <Text variant="heading" color="textMuted">
              PASSER
            </Text>
          </Animated.View>
        </Animated.View>
      ) : null}
    </View>
  );
});

const styles = StyleSheet.create({
  deck: {
    flex: 1
  },
  card: {
    ...StyleSheet.absoluteFillObject
  },
  stamp: {
    position: "absolute",
    top: spacing.xxl,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.xs,
    borderRadius: radii.md,
    borderWidth: 3,
    backgroundColor: "rgba(21,17,26,0.4)"
  },
  likeStamp: {
    left: spacing.lg,
    borderColor: colors.success,
    transform: [{ rotate: "-12deg" }]
  },
  passStamp: {
    right: spacing.lg,
    borderColor: colors.textMuted,
    transform: [{ rotate: "12deg" }]
  }
});
