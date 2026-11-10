import React, { forwardRef, useImperativeHandle, useMemo, useRef } from "react";
import { Animated, PanResponder, StyleSheet, View, useWindowDimensions } from "react-native";
import type { ProfileCard, SwipeAction } from "../../api/types";
import { useTheme } from "../../theme/ThemeProvider";
import { Text } from "../../ui";
import { ProfileCardView } from "./ProfileCardView";

export type SwipeDeckHandle = { swipe: (action: SwipeAction) => void };

type Props = {
  cards: ProfileCard[];
  onSwipe: (card: ProfileCard, action: SwipeAction) => void;
  onOpen: (card: ProfileCard) => void;
};

const SWIPE_THRESHOLD = 110;

/**
 * Two-card stack with a drag gesture (PanResponder + Animated; JS driver
 * because pan gestures cannot be driven natively without extra libraries).
 * Buttons call `swipe()` through the ref so gestures and buttons share the
 * exact same animation and callback.
 */
export const SwipeDeck = forwardRef<SwipeDeckHandle, Props>(function SwipeDeck({ cards, onSwipe, onOpen }, ref) {
  const theme = useTheme();
  const { width } = useWindowDimensions();
  const position = useRef(new Animated.ValueXY()).current;
  const animating = useRef(false);
  const top = cards[0];
  const latest = useRef({ top, onSwipe });
  latest.current = { top, onSwipe };

  const fling = (action: SwipeAction, velocity = 0) => {
    const current = latest.current.top;
    if (!current || animating.current) return;
    animating.current = true;
    const toX = (action === "like" ? 1 : -1) * (width + 200);
    Animated.timing(position, {
      toValue: { x: toX, y: 40 },
      duration: Math.max(160, 280 - Math.abs(velocity) * 60),
      useNativeDriver: false
    }).start(() => {
      position.setValue({ x: 0, y: 0 });
      animating.current = false;
      latest.current.onSwipe(current, action);
    });
  };

  useImperativeHandle(ref, () => ({ swipe: (action) => fling(action) }));

  const responder = useMemo(
    () =>
      PanResponder.create({
        onMoveShouldSetPanResponder: (_, g) => Math.abs(g.dx) > 8 && Math.abs(g.dx) > Math.abs(g.dy),
        onPanResponderMove: Animated.event([null, { dx: position.x, dy: position.y }], { useNativeDriver: false }),
        onPanResponderRelease: (_, g) => {
          if (g.dx > SWIPE_THRESHOLD || g.vx > 0.8) fling("like", g.vx);
          else if (g.dx < -SWIPE_THRESHOLD || g.vx < -0.8) fling("pass", g.vx);
          else Animated.spring(position, { toValue: { x: 0, y: 0 }, friction: 6, useNativeDriver: false }).start();
        },
        onPanResponderTerminate: () => Animated.spring(position, { toValue: { x: 0, y: 0 }, useNativeDriver: false }).start()
      }),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [position, width]
  );

  const rotate = position.x.interpolate({ inputRange: [-width, 0, width], outputRange: ["-12deg", "0deg", "12deg"], extrapolate: "clamp" });
  const likeOpacity = position.x.interpolate({ inputRange: [20, SWIPE_THRESHOLD], outputRange: [0, 1], extrapolate: "clamp" });
  const passOpacity = position.x.interpolate({ inputRange: [-SWIPE_THRESHOLD, -20], outputRange: [1, 0], extrapolate: "clamp" });
  const nextScale = position.x.interpolate({ inputRange: [-width, 0, width], outputRange: [1, 0.95, 1], extrapolate: "clamp" });

  return (
    <View style={styles.wrap}>
      {cards[1] ? (
        <Animated.View key={cards[1].userId} pointerEvents="none" style={[styles.card, { transform: [{ scale: nextScale }] }]}>
          <ProfileCardView card={cards[1]} interactive={false} />
        </Animated.View>
      ) : null}
      {top ? (
        <Animated.View
          key={top.userId}
          {...responder.panHandlers}
          style={[styles.card, theme.shadow(3), { transform: [{ translateX: position.x }, { translateY: position.y }, { rotate }] }]}
          testID="swipe-card"
        >
          <ProfileCardView card={top} onOpen={() => onOpen(top)} />
          <Animated.View pointerEvents="none" style={[styles.stamp, styles.stampLike, { opacity: likeOpacity, borderColor: theme.colors.success }]}>
            <Text color={theme.colors.success} variant="title">
              J’AIME
            </Text>
          </Animated.View>
          <Animated.View pointerEvents="none" style={[styles.stamp, styles.stampPass, { opacity: passOpacity, borderColor: "#1F1A1D" }]}>
            <Text color="#1F1A1D" variant="title">
              PASSER
            </Text>
          </Animated.View>
        </Animated.View>
      ) : null}
    </View>
  );
});

const styles = StyleSheet.create({
  wrap: { flex: 1 },
  card: { ...StyleSheet.absoluteFillObject },
  stamp: {
    position: "absolute",
    top: 36,
    borderWidth: 3,
    borderRadius: 12,
    paddingHorizontal: 12,
    paddingVertical: 4,
    backgroundColor: "rgba(255,255,255,0.88)"
  },
  stampLike: { left: 24, transform: [{ rotate: "-12deg" }] },
  stampPass: { right: 24, transform: [{ rotate: "12deg" }] }
});
