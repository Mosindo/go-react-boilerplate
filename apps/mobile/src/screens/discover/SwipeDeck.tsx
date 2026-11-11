import React, { forwardRef, useImperativeHandle, useMemo, useRef } from "react";
import { Animated, PanResponder, StyleSheet, View, useWindowDimensions } from "react-native";
import type { Card, SwipeDirection } from "../../api/types";
import { useTheme } from "../../theme/ThemeProvider";
import { Text } from "../../ui";
import { ProfileCard } from "./ProfileCard";

export type SwipeDeckHandle = { swipe: (direction: SwipeDirection) => void };

type Props = {
  cards: Card[];
  onSwipe: (card: Card, direction: SwipeDirection) => void;
  onOpen: (card: Card) => void;
};

const THRESHOLD = 110;

/** Drag the top card right to like, left to pass; buttons call `swipe()`. */
export const SwipeDeck = forwardRef<SwipeDeckHandle, Props>(function SwipeDeck({ cards, onSwipe, onOpen }, ref) {
  const { colors, radii } = useTheme();
  const { width } = useWindowDimensions();
  const position = useRef(new Animated.ValueXY()).current;
  const top = cards[0];
  const below = cards[1];
  const topRef = useRef(top);
  topRef.current = top;
  const animating = useRef(false);

  const fling = (direction: SwipeDirection) => {
    const card = topRef.current;
    if (!card || animating.current) {
      return;
    }
    animating.current = true;
    Animated.timing(position, {
      toValue: { x: (direction === "like" ? 1 : -1) * width * 1.4, y: 40 },
      duration: 240,
      useNativeDriver: false
    }).start(() => {
      position.setValue({ x: 0, y: 0 });
      animating.current = false;
      onSwipe(card, direction);
    });
  };

  useImperativeHandle(ref, () => ({ swipe: fling }));

  const responder = useMemo(
    () =>
      PanResponder.create({
        onMoveShouldSetPanResponder: (_e, g) => Math.abs(g.dx) > 8 && Math.abs(g.dx) > Math.abs(g.dy),
        onPanResponderMove: Animated.event([null, { dx: position.x, dy: position.y }], { useNativeDriver: false }),
        onPanResponderRelease: (_e, g) => {
          if (g.dx > THRESHOLD) {
            fling("like");
          } else if (g.dx < -THRESHOLD) {
            fling("pass");
          } else {
            Animated.spring(position, { toValue: { x: 0, y: 0 }, friction: 6, useNativeDriver: false }).start();
          }
        },
        onPanResponderTerminate: () => {
          Animated.spring(position, { toValue: { x: 0, y: 0 }, useNativeDriver: false }).start();
        }
      }),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [position, width]
  );

  const rotate = position.x.interpolate({ inputRange: [-width, 0, width], outputRange: ["-12deg", "0deg", "12deg"] });
  const likeOpacity = position.x.interpolate({
    inputRange: [20, THRESHOLD],
    outputRange: [0, 1],
    extrapolate: "clamp"
  });
  const passOpacity = position.x.interpolate({
    inputRange: [-THRESHOLD, -20],
    outputRange: [1, 0],
    extrapolate: "clamp"
  });
  const belowScale = position.x.interpolate({
    inputRange: [-width, 0, width],
    outputRange: [1, 0.95, 1],
    extrapolate: "clamp"
  });

  return (
    <View style={styles.deck}>
      {below ? (
        <Animated.View style={[StyleSheet.absoluteFill, { transform: [{ scale: belowScale }] }]} pointerEvents="none">
          <ProfileCard card={below} interactive={false} />
        </Animated.View>
      ) : null}
      {top ? (
        <Animated.View
          key={top.userId}
          testID="swipe-top-card"
          style={[
            StyleSheet.absoluteFill,
            { transform: [{ translateX: position.x }, { translateY: position.y }, { rotate }] }
          ]}
          {...responder.panHandlers}
        >
          <ProfileCard card={top} onOpenDetails={() => onOpen(top)} />
          <Animated.View
            pointerEvents="none"
            style={[
              styles.stamp,
              styles.stampLeft,
              { opacity: likeOpacity, borderColor: colors.success, borderRadius: radii.sm }
            ]}
          >
            <Text variant="title" style={{ color: colors.success }}>
              J’AIME
            </Text>
          </Animated.View>
          <Animated.View
            pointerEvents="none"
            style={[
              styles.stamp,
              styles.stampRight,
              { opacity: passOpacity, borderColor: colors.textMuted, borderRadius: radii.sm }
            ]}
          >
            <Text variant="title" style={{ color: "#FFFFFF" }}>
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
  stamp: {
    position: "absolute",
    top: 36,
    borderWidth: 3,
    paddingHorizontal: 10,
    paddingVertical: 2,
    backgroundColor: "rgba(0,0,0,0.15)"
  },
  stampLeft: { left: 24, transform: [{ rotate: "-14deg" }] },
  stampRight: { right: 24, transform: [{ rotate: "14deg" }] }
});
