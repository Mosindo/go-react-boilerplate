import React, { forwardRef, useCallback, useImperativeHandle, useMemo, useRef, type ReactNode } from "react";
import { Animated, Dimensions, PanResponder, StyleSheet, View } from "react-native";
import { Text } from "../../shared/ui";
import type { SwipeAction } from "../../api/types";

export type SwipeableCardHandle = { swipe: (action: SwipeAction) => void };

type SwipeableCardProps = {
  children: ReactNode;
  onSwiped: (action: SwipeAction) => void;
  disabled?: boolean;
};

const SCREEN_WIDTH = Dimensions.get("window").width;
const THRESHOLD = 110;
const EXIT_DISTANCE = Math.max(SCREEN_WIDTH, 600) * 1.3;

/** Drag right to like, left to pass; the handle lets buttons trigger the same animation. */
export const SwipeableCard = forwardRef<SwipeableCardHandle, SwipeableCardProps>(function SwipeableCard(
  { children, onSwiped, disabled },
  ref
) {
  const position = useRef(new Animated.ValueXY()).current;
  const done = useRef(false);

  const fling = useCallback(
    (action: SwipeAction) => {
      if (done.current) {
        return;
      }
      done.current = true;
      Animated.timing(position, {
        toValue: { x: action === "like" ? EXIT_DISTANCE : -EXIT_DISTANCE, y: 40 },
        duration: 220,
        useNativeDriver: true
      }).start(() => onSwiped(action));
    },
    [onSwiped, position]
  );

  useImperativeHandle(ref, () => ({ swipe: fling }), [fling]);

  const responder = useMemo(
    () =>
      PanResponder.create({
        onMoveShouldSetPanResponder: (_e, g) => !disabled && Math.abs(g.dx) > 8 && Math.abs(g.dx) > Math.abs(g.dy),
        onPanResponderMove: (_e, g) => position.setValue({ x: g.dx, y: g.dy * 0.3 }),
        onPanResponderRelease: (_e, g) => {
          if (g.dx > THRESHOLD || g.vx > 0.9) {
            fling("like");
          } else if (g.dx < -THRESHOLD || g.vx < -0.9) {
            fling("pass");
          } else {
            Animated.spring(position, { toValue: { x: 0, y: 0 }, useNativeDriver: true, friction: 6 }).start();
          }
        },
        onPanResponderTerminate: () => {
          Animated.spring(position, { toValue: { x: 0, y: 0 }, useNativeDriver: true }).start();
        }
      }),
    [disabled, fling, position]
  );

  const rotate = position.x.interpolate({ inputRange: [-250, 0, 250], outputRange: ["-9deg", "0deg", "9deg"] });
  const likeOpacity = position.x.interpolate({ inputRange: [20, THRESHOLD], outputRange: [0, 1], extrapolate: "clamp" });
  const passOpacity = position.x.interpolate({ inputRange: [-THRESHOLD, -20], outputRange: [1, 0], extrapolate: "clamp" });

  return (
    <Animated.View
      {...responder.panHandlers}
      style={[styles.card, { transform: [{ translateX: position.x }, { translateY: position.y }, { rotate }] }]}
      testID="swipe-card"
    >
      {children}
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
  );
});

export function CardBackdrop({ children }: { children: ReactNode }) {
  return (
    <View aria-hidden importantForAccessibility="no-hide-descendants" pointerEvents="none" style={styles.backdrop}>
      {children}
    </View>
  );
}

const styles = StyleSheet.create({
  card: { ...StyleSheet.absoluteFillObject },
  backdrop: { ...StyleSheet.absoluteFillObject, transform: [{ scale: 0.95 }], opacity: 0.9 },
  stamp: { position: "absolute", top: 36, borderWidth: 3, borderRadius: 10, paddingHorizontal: 12, paddingVertical: 4 },
  likeStamp: { left: 24, borderColor: "#7DDC9B", transform: [{ rotate: "-12deg" }] },
  passStamp: { right: 24, borderColor: "#FFB4A6", transform: [{ rotate: "12deg" }] },
  likeText: { color: "#7DDC9B" },
  passText: { color: "#FFB4A6" }
});
