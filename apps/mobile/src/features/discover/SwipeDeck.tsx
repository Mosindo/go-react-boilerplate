import React, { forwardRef, useCallback, useImperativeHandle, useLayoutEffect, useMemo, useRef, useState } from "react";
import { Animated, PanResponder, Pressable, StyleSheet, View } from "react-native";
import type { PublicProfile, SwipeAction } from "../../api/types";
import { useReducedMotion } from "../../lib/motion";
import { decideSwipe } from "../../lib/swipe";
import { MIN_TOUCH, radii, spacing, useColors } from "../../theme";
import { Icon, Text } from "../../ui";
import { usePressScale } from "../../ui/Button";
import { ProfileCardView } from "./ProfileCardView";

export type SwipeDeckHandle = { swipe: (action: SwipeAction) => void };

type Props = {
  cards: PublicProfile[];
  onSwipe: (profile: PublicProfile, action: SwipeAction) => void;
  onOpenProfile: (profile: PublicProfile) => void;
};

/**
 * Card stack. The top card follows the finger (PanResponder + Animated); the one behind it
 * is rendered scaled-down. Like/Pass buttons trigger the same fling animation.
 */
export const SwipeDeck = forwardRef<SwipeDeckHandle, Props>(function SwipeDeck({ cards, onSwipe, onOpenProfile }, ref) {
  const colors = useColors();
  const reducedMotion = useReducedMotion();
  const position = useRef(new Animated.ValueXY()).current;
  const [width, setWidth] = useState(360);
  const flinging = useRef(false);

  const top = cards[0];
  const next = cards[1];
  const topId = top?.userId;

  // Latest values for the (long-lived) PanResponder callbacks.
  const live = useRef({ top, onSwipe, width, reducedMotion });
  live.current = { top, onSwipe, width, reducedMotion };

  // Reset the transform once the next card has been committed as the new top card.
  useLayoutEffect(() => {
    position.setValue({ x: 0, y: 0 });
    flinging.current = false;
  }, [topId, position]);

  const fling = useCallback(
    (action: SwipeAction) => {
      const { top: current, width: w, reducedMotion: reduce } = live.current;
      if (!current || flinging.current) return;
      flinging.current = true;
      Animated.timing(position, {
        toValue: { x: (action === "like" ? 1 : -1) * w * 1.4, y: 30 },
        duration: reduce ? 0 : 220,
        useNativeDriver: false
      }).start(() => live.current.onSwipe(current, action));
    },
    [position]
  );

  useImperativeHandle(ref, () => ({ swipe: fling }), [fling]);

  const panResponder = useMemo(
    () =>
      PanResponder.create({
        onMoveShouldSetPanResponder: (_e, g) =>
          !flinging.current && Math.abs(g.dx) > 8 && Math.abs(g.dx) > Math.abs(g.dy),
        onPanResponderMove: Animated.event([null, { dx: position.x, dy: position.y }], { useNativeDriver: false }),
        onPanResponderRelease: (_e, g) => {
          const decision = decideSwipe(g.dx, g.vx, live.current.width);
          if (decision) fling(decision);
          else Animated.spring(position, { toValue: { x: 0, y: 0 }, useNativeDriver: false, bounciness: 6 }).start();
        },
        onPanResponderTerminate: () => {
          Animated.spring(position, { toValue: { x: 0, y: 0 }, useNativeDriver: false }).start();
        }
      }),
    [fling, position]
  );

  const rotate = position.x.interpolate({
    inputRange: [-width, 0, width],
    outputRange: ["-12deg", "0deg", "12deg"]
  });
  const likeOpacity = position.x.interpolate({
    inputRange: [0, width * 0.25],
    outputRange: [0, 1],
    extrapolate: "clamp"
  });
  const passOpacity = position.x.interpolate({
    inputRange: [-width * 0.25, 0],
    outputRange: [1, 0],
    extrapolate: "clamp"
  });
  const nextScale = position.x.interpolate({
    inputRange: [-width, 0, width],
    outputRange: [1, 0.94, 1],
    extrapolate: "clamp"
  });

  return (
    <View style={styles.wrap}>
      <View onLayout={(e) => setWidth(e.nativeEvent.layout.width)} style={styles.stack}>
        {next ? (
          <Animated.View
            key={next.userId}
            pointerEvents="none"
            style={[styles.cardLayer, { transform: [{ scale: nextScale }] }]}
          >
            <ProfileCardView onOpen={() => undefined} profile={next} />
          </Animated.View>
        ) : null}
        {top ? (
          <Animated.View
            key={top.userId}
            accessibilityActions={[
              { name: "like", label: "Like" },
              { name: "pass", label: "Pass" }
            ]}
            onAccessibilityAction={(event) => {
              if (event.nativeEvent.actionName === "like") fling("like");
              if (event.nativeEvent.actionName === "pass") fling("pass");
            }}
            style={[
              styles.cardLayer,
              { transform: [{ translateX: position.x }, { translateY: position.y }, { rotate }] }
            ]}
            testID="deck-top-card"
            {...panResponder.panHandlers}
          >
            <ProfileCardView onOpen={() => onOpenProfile(top)} profile={top} />
            <Animated.View
              pointerEvents="none"
              style={[styles.stamp, styles.stampLike, { opacity: likeOpacity, borderColor: colors.success }]}
            >
              <Text style={{ color: colors.success }} variant="title" weight="bold">
                LIKE
              </Text>
            </Animated.View>
            <Animated.View
              pointerEvents="none"
              style={[styles.stamp, styles.stampPass, { opacity: passOpacity, borderColor: colors.danger }]}
            >
              <Text style={{ color: colors.danger }} variant="title" weight="bold">
                PASS
              </Text>
            </Animated.View>
          </Animated.View>
        ) : null}
      </View>

      <View style={styles.actions}>
        <RoundButton color={colors.danger} disabled={!top} icon="close" label="Pass" onPress={() => fling("pass")} />
        <RoundButton
          color={colors.primary}
          disabled={!top}
          icon="heart"
          label="Like"
          large
          onPress={() => fling("like")}
        />
      </View>
    </View>
  );
});

type RoundButtonProps = {
  icon: "close" | "heart";
  label: string;
  color: string;
  onPress: () => void;
  disabled?: boolean;
  large?: boolean;
};

function RoundButton({ icon, label, color, onPress, disabled, large }: RoundButtonProps) {
  const colors = useColors();
  const press = usePressScale(0.9);
  const size = large ? 68 : 58;
  return (
    <Animated.View style={{ transform: [{ scale: press.scale }] }}>
      <Pressable
        accessibilityLabel={label}
        accessibilityRole="button"
        accessibilityState={{ disabled: !!disabled }}
        disabled={disabled}
        onPress={onPress}
        onPressIn={press.onPressIn}
        onPressOut={press.onPressOut}
        style={[
          styles.round,
          {
            width: Math.max(size, MIN_TOUCH),
            height: Math.max(size, MIN_TOUCH),
            borderRadius: size / 2,
            backgroundColor: large ? color : colors.surface,
            borderColor: color,
            opacity: disabled ? 0.4 : 1
          }
        ]}
      >
        <Icon color={large ? colors.onPrimary : color} name={icon} size={large ? 34 : 30} />
      </Pressable>
    </Animated.View>
  );
}

const styles = StyleSheet.create({
  wrap: { flex: 1, gap: spacing.md },
  stack: { flex: 1 },
  cardLayer: { ...StyleSheet.absoluteFillObject },
  actions: {
    flexDirection: "row",
    justifyContent: "center",
    alignItems: "center",
    gap: spacing.xl,
    paddingBottom: spacing.xs
  },
  round: { borderWidth: 2, alignItems: "center", justifyContent: "center" },
  stamp: {
    position: "absolute",
    top: spacing.xl,
    borderWidth: 3,
    borderRadius: radii.md,
    paddingHorizontal: spacing.sm,
    backgroundColor: "rgba(255,255,255,0.85)"
  },
  stampLike: { left: spacing.md, transform: [{ rotate: "-14deg" }] },
  stampPass: { right: spacing.md, transform: [{ rotate: "14deg" }] }
});
