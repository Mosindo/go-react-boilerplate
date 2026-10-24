import React, { forwardRef, useImperativeHandle, useMemo, useRef } from "react";
import { Animated, PanResponder, StyleSheet, useWindowDimensions } from "react-native";
import { radius, spacing } from "../../theme/theme";
import { Text } from "../../ui";

export type SwipeDirection = "left" | "right";
export type SwipeCardHandle = { swipe: (direction: SwipeDirection) => void };

type Props = {
  children: React.ReactNode;
  onSwiped: (direction: SwipeDirection) => void;
};

const COMMIT_RATIO = 0.28;
const COMMIT_VELOCITY = 0.8;

/** Drag the card sideways: right = like, left = pass. The same handle powers the buttons. */
export const SwipeCard = forwardRef<SwipeCardHandle, Props>(function SwipeCard(
  { children, onSwiped },
  ref,
) {
  const { width } = useWindowDimensions();
  const position = useRef(new Animated.ValueXY()).current;
  const finished = useRef(false);
  const onSwipedRef = useRef(onSwiped);
  onSwipedRef.current = onSwiped;

  const fling = (direction: SwipeDirection) => {
    if (finished.current) return;
    finished.current = true;
    Animated.timing(position, {
      toValue: { x: (direction === "right" ? 1 : -1) * width * 1.4, y: 0 },
      duration: 220,
      useNativeDriver: true,
    }).start(() => onSwipedRef.current(direction));
  };

  const responder = useMemo(
    () =>
      PanResponder.create({
        onMoveShouldSetPanResponder: (_, g) =>
          Math.abs(g.dx) > 8 && Math.abs(g.dx) > Math.abs(g.dy),
        onPanResponderMove: Animated.event([null, { dx: position.x, dy: position.y }], {
          useNativeDriver: false,
        }),
        onPanResponderRelease: (_, g) => {
          if (g.dx > width * COMMIT_RATIO || g.vx > COMMIT_VELOCITY) fling("right");
          else if (g.dx < -width * COMMIT_RATIO || g.vx < -COMMIT_VELOCITY) fling("left");
          else
            Animated.spring(position, {
              toValue: { x: 0, y: 0 },
              useNativeDriver: true,
              friction: 6,
            }).start();
        },
      }),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [width],
  );

  useImperativeHandle(ref, () => ({ swipe: fling }));

  const rotate = position.x.interpolate({
    inputRange: [-width, 0, width],
    outputRange: ["-12deg", "0deg", "12deg"],
  });
  const likeOpacity = position.x.interpolate({
    inputRange: [0, width * 0.3],
    outputRange: [0, 1],
    extrapolate: "clamp",
  });
  const passOpacity = position.x.interpolate({
    inputRange: [-width * 0.3, 0],
    outputRange: [1, 0],
    extrapolate: "clamp",
  });

  return (
    <Animated.View
      {...responder.panHandlers}
      style={[
        styles.fill,
        { transform: [{ translateX: position.x }, { translateY: position.y }, { rotate }] },
      ]}
    >
      {children}
      <Animated.View
        pointerEvents="none"
        style={[styles.stamp, styles.like, { opacity: likeOpacity }]}
      >
        <Text variant="heading" color="#2E7D4F">
          J&apos;AIME
        </Text>
      </Animated.View>
      <Animated.View
        pointerEvents="none"
        style={[styles.stamp, styles.pass, { opacity: passOpacity }]}
      >
        <Text variant="heading" color="#B3261E">
          PASSER
        </Text>
      </Animated.View>
    </Animated.View>
  );
});

const styles = StyleSheet.create({
  fill: { flex: 1 },
  stamp: {
    position: "absolute",
    top: spacing.xl,
    borderWidth: 3,
    borderRadius: radius.md,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.xs,
    backgroundColor: "rgba(255,255,255,0.85)",
  },
  like: { left: spacing.xl, borderColor: "#2E7D4F", transform: [{ rotate: "-12deg" }] },
  pass: { right: spacing.xl, borderColor: "#B3261E", transform: [{ rotate: "12deg" }] },
});
