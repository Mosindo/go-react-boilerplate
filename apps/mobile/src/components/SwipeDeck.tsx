import React, { forwardRef, useImperativeHandle, useMemo, useRef } from "react";
import { Animated, PanResponder, StyleSheet, View, useWindowDimensions } from "react-native";
import type { ProfileCard, SwipeAction } from "../api/types";
import { colors, radii, shadows, spacing } from "../theme/tokens";
import { Text } from "../ui";
import { ProfileCardView } from "./ProfileCardView";

export type SwipeDeckHandle = { swipe: (action: SwipeAction) => void };

type Props = {
  cards: ProfileCard[];
  onSwipe: (card: ProfileCard, action: SwipeAction) => void;
  onOpen: (card: ProfileCard) => void;
};

const SWIPE_THRESHOLD = 110;

/**
 * Card stack with drag-to-decide. Built on RN's PanResponder + Animated so it
 * works on iOS, Android and web without extra native modules. Buttons call
 * the same path through the imperative `swipe` handle (accessibility).
 */
export const SwipeDeck = forwardRef<SwipeDeckHandle, Props>(function SwipeDeck({ cards, onSwipe, onOpen }, ref) {
  const { width } = useWindowDimensions();
  const position = useRef(new Animated.ValueXY()).current;
  const animating = useRef(false);
  const top = cards[0];
  const latest = useRef({ top, onSwipe });
  latest.current = { top, onSwipe };

  const finish = (action: SwipeAction, velocity = 0) => {
    const card = latest.current.top;
    if (!card || animating.current) {
      return;
    }
    animating.current = true;
    const direction = action === "like" ? 1 : -1;
    Animated.timing(position, {
      toValue: { x: direction * (width + 200), y: 40 },
      duration: Math.max(160, 280 - Math.abs(velocity) * 60),
      useNativeDriver: false
    }).start(() => {
      position.setValue({ x: 0, y: 0 });
      animating.current = false;
      latest.current.onSwipe(card, action);
    });
  };

  useImperativeHandle(ref, () => ({ swipe: (action) => finish(action) }));

  const responder = useMemo(
    () =>
      PanResponder.create({
        onMoveShouldSetPanResponder: (_, g) => Math.abs(g.dx) > 8 && Math.abs(g.dx) > Math.abs(g.dy),
        onPanResponderMove: (_, g) => position.setValue({ x: g.dx, y: g.dy * 0.25 }),
        onPanResponderRelease: (_, g) => {
          if (g.dx > SWIPE_THRESHOLD || g.vx > 0.8) {
            finish("like", g.vx);
          } else if (g.dx < -SWIPE_THRESHOLD || g.vx < -0.8) {
            finish("pass", g.vx);
          } else {
            Animated.spring(position, { toValue: { x: 0, y: 0 }, friction: 6, useNativeDriver: false }).start();
          }
        },
        onPanResponderTerminate: () => Animated.spring(position, { toValue: { x: 0, y: 0 }, useNativeDriver: false }).start()
      }),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [position, width]
  );

  const rotate = position.x.interpolate({ inputRange: [-width, 0, width], outputRange: ["-12deg", "0deg", "12deg"] });
  const likeOpacity = position.x.interpolate({ inputRange: [20, SWIPE_THRESHOLD], outputRange: [0, 1], extrapolate: "clamp" });
  const passOpacity = position.x.interpolate({ inputRange: [-SWIPE_THRESHOLD, -20], outputRange: [1, 0], extrapolate: "clamp" });
  const nextScale = position.x.interpolate({ inputRange: [-width, 0, width], outputRange: [1, 0.95, 1], extrapolate: "clamp" });

  return (
    <View style={styles.root}>
      {cards
        .slice(0, 2)
        .reverse()
        .map((card, reversedIndex, arr) => {
          const isTop = reversedIndex === arr.length - 1;
          if (!isTop) {
            return (
              <Animated.View key={card.userId} style={[styles.card, { transform: [{ scale: nextScale }] }]} pointerEvents="none">
                <ProfileCardView card={card} />
              </Animated.View>
            );
          }
          return (
            <Animated.View
              key={card.userId}
              testID="deck-top-card"
              accessibilityLabel={`${card.firstName}, ${card.age} ans`}
              style={[styles.card, shadows.card, { transform: [{ translateX: position.x }, { translateY: position.y }, { rotate }] }]}
              {...responder.panHandlers}
            >
              <ProfileCardView card={card} onOpen={() => onOpen(card)} />
              <Animated.View style={[styles.stamp, styles.likeStamp, { opacity: likeOpacity }]} pointerEvents="none">
                <Text variant="heading" style={{ color: colors.accent }}>
                  J&apos;AIME
                </Text>
              </Animated.View>
              <Animated.View style={[styles.stamp, styles.passStamp, { opacity: passOpacity }]} pointerEvents="none">
                <Text variant="heading" tone="muted">
                  PASSER
                </Text>
              </Animated.View>
            </Animated.View>
          );
        })}
    </View>
  );
});

const styles = StyleSheet.create({
  root: { flex: 1 },
  card: { ...StyleSheet.absoluteFillObject, borderRadius: radii.xl, backgroundColor: colors.surface },
  stamp: {
    position: "absolute",
    top: spacing.xxxl,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.xs,
    borderWidth: 2,
    borderRadius: radii.sm,
    zIndex: 5
  },
  likeStamp: { left: spacing.xl, borderColor: colors.accent, transform: [{ rotate: "-12deg" }] },
  passStamp: { right: spacing.xl, borderColor: colors.textMuted, transform: [{ rotate: "12deg" }] }
});
