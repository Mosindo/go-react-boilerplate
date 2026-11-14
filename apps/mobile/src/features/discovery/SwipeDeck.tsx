import React, { useEffect, useImperativeHandle, useMemo, useRef, forwardRef } from "react";
import { Animated, PanResponder, StyleSheet, View, useWindowDimensions } from "react-native";
import type { PublicProfile, SwipeAction } from "../../api/types";
import { radii } from "../../theme/palette";
import { useTheme } from "../../theme/ThemeProvider";
import { Text } from "../../ui";
import { ProfileCard } from "./ProfileCard";

export type SwipeDeckHandle = { swipe: (action: SwipeAction) => void };

type Props = {
  profiles: PublicProfile[];
  onSwipe: (profile: PublicProfile, action: SwipeAction) => void;
  onOpen: (profile: PublicProfile) => void;
};

/**
 * Swipeable stack built on Animated + PanResponder (no extra native
 * dependency; JS driver because the pan gesture feeds the same value). Buttons drive the same animation through the ref, so the
 * deck is fully usable without gestures (accessibility, web, desktop).
 */
export const SwipeDeck = forwardRef<SwipeDeckHandle, Props>(function SwipeDeck({ profiles, onSwipe, onOpen }, ref) {
  const { colors } = useTheme();
  const { width } = useWindowDimensions();
  const position = useRef(new Animated.ValueXY()).current;
  const animating = useRef(false);
  const top = profiles[0];
  const threshold = Math.min(140, width * 0.28);

  const latest = useRef({ top, onSwipe });
  latest.current = { top, onSwipe };

  const fling = useMemo(
    () => (action: SwipeAction) => {
      const current = latest.current.top;
      if (!current || animating.current) return;
      animating.current = true;
      Animated.timing(position, {
        toValue: { x: (action === "like" ? 1 : -1) * width * 1.4, y: 40 },
        duration: 260,
        useNativeDriver: false
      }).start(() => {
        latest.current.onSwipe(current, action);
      });
    },
    [position, width]
  );

  // Reset the animated position once the next card is on top.
  useEffect(() => {
    position.setValue({ x: 0, y: 0 });
    animating.current = false;
  }, [top?.userId, position]);

  useImperativeHandle(ref, () => ({ swipe: fling }), [fling]);

  const responder = useMemo(
    () =>
      PanResponder.create({
        onMoveShouldSetPanResponder: (_, g) => Math.abs(g.dx) > 8 && Math.abs(g.dx) > Math.abs(g.dy),
        onPanResponderMove: Animated.event([null, { dx: position.x, dy: position.y }], { useNativeDriver: false }),
        onPanResponderRelease: (_, g) => {
          if (g.dx > threshold || g.vx > 1.2) {
            fling("like");
          } else if (g.dx < -threshold || g.vx < -1.2) {
            fling("pass");
          } else {
            Animated.spring(position, { toValue: { x: 0, y: 0 }, friction: 6, useNativeDriver: false }).start();
          }
        },
        onPanResponderTerminate: () => {
          Animated.spring(position, { toValue: { x: 0, y: 0 }, useNativeDriver: false }).start();
        }
      }),
    [fling, position, threshold]
  );

  const rotate = position.x.interpolate({ inputRange: [-width, 0, width], outputRange: ["-12deg", "0deg", "12deg"] });
  const likeOpacity = position.x.interpolate({ inputRange: [0, threshold], outputRange: [0, 1], extrapolate: "clamp" });
  const passOpacity = position.x.interpolate({ inputRange: [-threshold, 0], outputRange: [1, 0], extrapolate: "clamp" });
  const nextScale = position.x.interpolate({ inputRange: [-width, 0, width], outputRange: [1, 0.95, 1], extrapolate: "clamp" });

  return (
    <View style={styles.stack}>
      {profiles[1] ? (
        <Animated.View key={profiles[1].userId} pointerEvents="none" style={[styles.layer, { transform: [{ scale: nextScale }] }]}>
          <ProfileCard onOpen={() => undefined} profile={profiles[1]} />
        </Animated.View>
      ) : null}
      {top ? (
        <Animated.View
          key={top.userId}
          {...responder.panHandlers}
          style={[styles.layer, { transform: [{ translateX: position.x }, { translateY: position.y }, { rotate }] }]}
          testID="deck-top-card"
        >
          <ProfileCard onOpen={() => onOpen(top)} profile={top} />
          <Animated.View pointerEvents="none" style={[styles.stamp, styles.stampLike, { borderColor: colors.success, opacity: likeOpacity }]}>
            <Text style={{ color: colors.success }} variant="title">
              J’AIME
            </Text>
          </Animated.View>
          <Animated.View pointerEvents="none" style={[styles.stamp, styles.stampPass, { borderColor: "#fff", opacity: passOpacity }]}>
            <Text style={{ color: "#fff" }} variant="title">
              PLUS TARD
            </Text>
          </Animated.View>
        </Animated.View>
      ) : null}
    </View>
  );
});

const styles = StyleSheet.create({
  stack: { flex: 1 },
  layer: { ...StyleSheet.absoluteFillObject },
  stamp: { position: "absolute", top: 36, paddingHorizontal: 14, paddingVertical: 6, borderWidth: 3, borderRadius: radii.md },
  stampLike: { left: 24, transform: [{ rotate: "-12deg" }] },
  stampPass: { right: 24, transform: [{ rotate: "12deg" }] }
});
