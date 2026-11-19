import React, { forwardRef, useImperativeHandle, useMemo, useRef } from "react";
import { Animated, PanResponder, StyleSheet, useWindowDimensions, View } from "react-native";
import type { ProfileCard } from "../../api/types";
import { useTheme } from "../../theme/ThemeProvider";
import { radii } from "../../theme/tokens";
import { Text } from "../../ui";
import { ProfileCardView } from "./ProfileCardView";

export type Decision = "like" | "pass";
export type SwipeDeckHandle = { swipe: (decision: Decision) => void };

type Props = {
  cards: ProfileCard[];
  onDecision: (card: ProfileCard, decision: Decision) => void;
  onOpen: (card: ProfileCard) => void;
};

const THRESHOLD = 110;

/** Two-card deck: the top card follows the finger; buttons trigger the same animation. */
export const SwipeDeck = forwardRef<SwipeDeckHandle, Props>(function SwipeDeck({ cards, onDecision, onOpen }, ref) {
  const { colors } = useTheme();
  const { width } = useWindowDimensions();
  const position = useRef(new Animated.ValueXY()).current;
  const animating = useRef(false);
  const top = cards[0];
  const topRef = useRef(top);
  topRef.current = top;
  const decide = useRef(onDecision);
  decide.current = onDecision;

  const fling = (decision: Decision) => {
    const card = topRef.current;
    if (!card || animating.current) return;
    animating.current = true;
    Animated.timing(position, {
      toValue: { x: (decision === "like" ? 1 : -1) * (width + 120), y: 30 },
      duration: 240,
      useNativeDriver: false
    }).start(() => {
      position.setValue({ x: 0, y: 0 });
      animating.current = false;
      decide.current(card, decision);
    });
  };
  const flingRef = useRef(fling);
  flingRef.current = fling;

  useImperativeHandle(ref, () => ({ swipe: (d) => flingRef.current(d) }));

  const responder = useMemo(
    () =>
      PanResponder.create({
        onMoveShouldSetPanResponder: (_, g) => !animating.current && Math.abs(g.dx) > 8 && Math.abs(g.dx) > Math.abs(g.dy),
        onPanResponderMove: Animated.event([null, { dx: position.x, dy: position.y }], { useNativeDriver: false }),
        onPanResponderRelease: (_, g) => {
          if (g.dx > THRESHOLD || g.vx > 0.8) flingRef.current("like");
          else if (g.dx < -THRESHOLD || g.vx < -0.8) flingRef.current("pass");
          else Animated.spring(position, { toValue: { x: 0, y: 0 }, friction: 6, useNativeDriver: false }).start();
        },
        onPanResponderTerminate: () => Animated.spring(position, { toValue: { x: 0, y: 0 }, useNativeDriver: false }).start()
      }),
    [position]
  );

  const rotate = position.x.interpolate({ inputRange: [-width, 0, width], outputRange: ["-12deg", "0deg", "12deg"] });
  const likeOpacity = position.x.interpolate({ inputRange: [20, THRESHOLD], outputRange: [0, 1], extrapolate: "clamp" });
  const passOpacity = position.x.interpolate({ inputRange: [-THRESHOLD, -20], outputRange: [1, 0], extrapolate: "clamp" });
  const nextScale = position.x.interpolate({ inputRange: [-width, 0, width], outputRange: [1, 0.95, 1], extrapolate: "clamp" });

  return (
    <View style={styles.deck}>
      {cards[1] ? (
        <Animated.View style={[StyleSheet.absoluteFill, { transform: [{ scale: nextScale }] }]} pointerEvents="none">
          <ProfileCardView key={cards[1].userId} card={cards[1]} onOpen={() => undefined} active={false} />
        </Animated.View>
      ) : null}
      {top ? (
        <Animated.View
          key={top.userId}
          style={[StyleSheet.absoluteFill, { transform: [{ translateX: position.x }, { translateY: position.y }, { rotate }] }]}
          {...responder.panHandlers}
        >
          <ProfileCardView card={top} onOpen={() => onOpen(top)} />
          <Animated.View
            style={[styles.stamp, styles.stampLike, { opacity: likeOpacity, borderColor: colors.accent }]}
            pointerEvents="none"
          >
            <Text variant="heading" style={{ color: colors.accent }}>
              J’AIME
            </Text>
          </Animated.View>
          <Animated.View style={[styles.stamp, styles.stampPass, { opacity: passOpacity, borderColor: "#FFFFFF" }]} pointerEvents="none">
            <Text variant="heading" tone="onPhoto">
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
  stamp: { position: "absolute", top: 36, borderWidth: 3, borderRadius: radii.sm, paddingHorizontal: 12, paddingVertical: 4 },
  stampLike: { left: 24, transform: [{ rotate: "-12deg" }] },
  stampPass: { right: 24, transform: [{ rotate: "12deg" }] }
});
