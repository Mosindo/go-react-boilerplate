import React, { forwardRef, useImperativeHandle, useMemo, useRef } from "react";
import { Animated, PanResponder, StyleSheet, View, useWindowDimensions } from "react-native";
import type { DiscoverCard, SwipeAction } from "../../api/types";
import { colors, radii, spacing } from "../../theme";
import { Text } from "../../ui";
import { ProfileCard } from "./ProfileCard";

const SWIPE_THRESHOLD = 110;

export type SwipeDeckHandle = { swipe: (action: SwipeAction) => void };

type Props = {
  cards: DiscoverCard[];
  onSwipe: (card: DiscoverCard, action: SwipeAction) => void;
  onOpenDetails: (card: DiscoverCard) => void;
};

/**
 * Card stack: drag the top card right to like, left to pass. Buttons call the same
 * animation through the imperative handle so gestures stay optional (accessibility).
 */
export const SwipeDeck = forwardRef<SwipeDeckHandle, Props>(function SwipeDeck({ cards, onSwipe, onOpenDetails }, ref) {
  const { width } = useWindowDimensions();
  const position = useRef(new Animated.ValueXY()).current;
  const animating = useRef(false);
  const top = cards[0];
  const latest = useRef({ top, onSwipe });
  latest.current = { top, onSwipe };

  const flyOut = (action: SwipeAction) => {
    const card = latest.current.top;
    if (!card || animating.current) return;
    animating.current = true;
    Animated.timing(position, {
      toValue: { x: (action === "like" ? 1 : -1) * (width + 120), y: 30 },
      duration: 220,
      useNativeDriver: false
    }).start(() => {
      position.setValue({ x: 0, y: 0 });
      animating.current = false;
      latest.current.onSwipe(card, action);
    });
  };

  useImperativeHandle(ref, () => ({ swipe: flyOut }));

  const responder = useMemo(
    () =>
      PanResponder.create({
        onMoveShouldSetPanResponder: (_, g) => Math.abs(g.dx) > 8 && Math.abs(g.dx) > Math.abs(g.dy),
        onPanResponderMove: Animated.event([null, { dx: position.x, dy: position.y }], { useNativeDriver: false }),
        onPanResponderRelease: (_, g) => {
          if (g.dx > SWIPE_THRESHOLD) flyOut("like");
          else if (g.dx < -SWIPE_THRESHOLD) flyOut("pass");
          else Animated.spring(position, { toValue: { x: 0, y: 0 }, friction: 6, useNativeDriver: false }).start();
        },
        onPanResponderTerminate: () => {
          Animated.spring(position, { toValue: { x: 0, y: 0 }, useNativeDriver: false }).start();
        }
      }),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [width]
  );

  const rotate = position.x.interpolate({ inputRange: [-width, 0, width], outputRange: ["-12deg", "0deg", "12deg"] });
  const likeOpacity = position.x.interpolate({ inputRange: [20, SWIPE_THRESHOLD], outputRange: [0, 1], extrapolate: "clamp" });
  const passOpacity = position.x.interpolate({ inputRange: [-SWIPE_THRESHOLD, -20], outputRange: [1, 0], extrapolate: "clamp" });
  const nextScale = position.x.interpolate({ inputRange: [-width, 0, width], outputRange: [1, 0.95, 1], extrapolate: "clamp" });

  return (
    <View style={styles.deck}>
      {cards[1] ? (
        <Animated.View style={[styles.layer, { transform: [{ scale: nextScale }] }]} pointerEvents="none">
          <ProfileCard card={cards[1]} onOpenDetails={() => undefined} />
        </Animated.View>
      ) : null}
      {top ? (
        <Animated.View
          key={top.userId}
          style={[styles.layer, { transform: [{ translateX: position.x }, { translateY: position.y }, { rotate }] }]}
          {...responder.panHandlers}
        >
          <ProfileCard card={top} onOpenDetails={() => onOpenDetails(top)} />
          <Animated.View style={[styles.stamp, styles.likeStamp, { opacity: likeOpacity }]} pointerEvents="none">
            <Text variant="title" tone="success">
              J’AIME
            </Text>
          </Animated.View>
          <Animated.View style={[styles.stamp, styles.passStamp, { opacity: passOpacity }]} pointerEvents="none">
            <Text variant="title" tone="muted">
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
  layer: {
    ...StyleSheet.absoluteFillObject
  },
  stamp: {
    position: "absolute",
    top: spacing.xxxl,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.xs,
    borderWidth: 3,
    borderRadius: radii.md,
    zIndex: 5
  },
  likeStamp: {
    left: spacing.xl,
    borderColor: colors.success,
    transform: [{ rotate: "-14deg" }]
  },
  passStamp: {
    right: spacing.xl,
    borderColor: colors.pass,
    transform: [{ rotate: "14deg" }]
  }
});
