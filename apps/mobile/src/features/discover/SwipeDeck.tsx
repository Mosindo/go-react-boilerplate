import React, { forwardRef, useImperativeHandle, useMemo, useRef } from "react";
import { Animated, PanResponder, StyleSheet, View, useWindowDimensions } from "react-native";
import { Text } from "../../design/components";
import { useTheme } from "../../design/ThemeProvider";
import type { PublicProfile } from "../../lib/api/types";
import { ProfileCard } from "./ProfileCard";

export type SwipeDirection = "like" | "pass";
export type SwipeDeckHandle = { swipe: (direction: SwipeDirection) => void };

type Props = {
  profiles: PublicProfile[];
  onSwipe: (profile: PublicProfile, direction: SwipeDirection) => void;
  onOpenDetails: (profile: PublicProfile) => void;
};

const SWIPE_THRESHOLD = 110;

export const SwipeDeck = forwardRef<SwipeDeckHandle, Props>(function SwipeDeck({ profiles, onSwipe, onOpenDetails }, ref) {
  const { width } = useWindowDimensions();
  const { colors } = useTheme();
  const position = useRef(new Animated.ValueXY()).current;
  const animating = useRef(false);
  const top = profiles[0];
  // Latest values for the gesture handlers, which are created once.
  const latest = useRef({ top, onSwipe, width });
  latest.current = { top, onSwipe, width };

  // The pan gesture drives `position` from JS, so every animation on it must
  // use the JS driver too (mixing drivers on one value throws).
  const fling = useRef((direction: SwipeDirection, velocity = 0) => {
    const { top: current, onSwipe: notify, width: w } = latest.current;
    if (!current || animating.current) return;
    animating.current = true;
    Animated.timing(position, {
      toValue: { x: (direction === "like" ? 1 : -1) * w * 1.3, y: 0 },
      duration: Math.max(160, 280 - Math.abs(velocity) * 60),
      useNativeDriver: false
    }).start(() => {
      position.setValue({ x: 0, y: 0 });
      animating.current = false;
      notify(current, direction);
    });
  }).current;

  useImperativeHandle(ref, () => ({ swipe: (direction) => fling(direction) }), [fling]);

  const responder = useMemo(
    () =>
      PanResponder.create({
        onMoveShouldSetPanResponder: (_, g) => Math.abs(g.dx) > 8 && Math.abs(g.dx) > Math.abs(g.dy),
        onPanResponderMove: Animated.event([null, { dx: position.x, dy: position.y }], { useNativeDriver: false }),
        onPanResponderRelease: (_, g) => {
          if (g.dx > SWIPE_THRESHOLD || g.vx > 0.8) {
            fling("like", g.vx);
          } else if (g.dx < -SWIPE_THRESHOLD || g.vx < -0.8) {
            fling("pass", g.vx);
          } else {
            Animated.spring(position, { toValue: { x: 0, y: 0 }, friction: 6, useNativeDriver: false }).start();
          }
        },
        onPanResponderTerminate: () => {
          Animated.spring(position, { toValue: { x: 0, y: 0 }, useNativeDriver: false }).start();
        }
      }),
    [fling, position]
  );

  const rotate = position.x.interpolate({ inputRange: [-width, 0, width], outputRange: ["-12deg", "0deg", "12deg"] });
  const likeOpacity = position.x.interpolate({ inputRange: [20, SWIPE_THRESHOLD], outputRange: [0, 1], extrapolate: "clamp" });
  const passOpacity = position.x.interpolate({ inputRange: [-SWIPE_THRESHOLD, -20], outputRange: [1, 0], extrapolate: "clamp" });
  const nextScale = position.x.interpolate({ inputRange: [-width, 0, width], outputRange: [1, 0.95, 1], extrapolate: "clamp" });

  return (
    <View style={styles.deck}>
      {profiles[1] ? (
        <Animated.View key={profiles[1].userId} pointerEvents="none" style={[styles.card, { transform: [{ scale: nextScale }] }]}>
          <ProfileCard onOpenDetails={() => undefined} profile={profiles[1]} />
        </Animated.View>
      ) : null}
      {top ? (
        <Animated.View
          key={top.userId}
          {...responder.panHandlers}
          style={[styles.card, { transform: [{ translateX: position.x }, { translateY: position.y }, { rotate }] }]}
        >
          <ProfileCard onOpenDetails={() => onOpenDetails(top)} profile={top} />
          <Animated.View pointerEvents="none" style={[styles.stamp, styles.stampLike, { borderColor: colors.success, opacity: likeOpacity }]}>
            <Text style={{ color: colors.success }} variant="title">
              J’AIME
            </Text>
          </Animated.View>
          <Animated.View pointerEvents="none" style={[styles.stamp, styles.stampPass, { borderColor: "#fff", opacity: passOpacity }]}>
            <Text style={styles.passText} variant="title">
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
  card: { ...StyleSheet.absoluteFillObject },
  stamp: { position: "absolute", top: 40, paddingHorizontal: 14, paddingVertical: 6, borderWidth: 3, borderRadius: 10 },
  stampLike: { left: 24, transform: [{ rotate: "-12deg" }] },
  stampPass: { right: 24, transform: [{ rotate: "12deg" }] },
  passText: { color: "#fff" }
});
