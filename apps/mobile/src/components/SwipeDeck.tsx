import React, { forwardRef, useImperativeHandle, useMemo, useRef } from "react";
import { Animated, PanResponder, StyleSheet, View, useWindowDimensions } from "react-native";
import type { PublicProfile, SwipeAction } from "../api/types";
import { decideSwipe } from "../lib/swipe";
import { spacing, useTheme } from "../theme";
import { ProfileCard } from "./ProfileCard";
import { Text } from "./Text";

export type SwipeDeckHandle = { swipe: (action: SwipeAction) => void };

type Props = {
  profiles: PublicProfile[];
  onSwipe: (profile: PublicProfile, action: SwipeAction) => void;
  onOpenProfile: (profile: PublicProfile) => void;
};

export const SwipeDeck = forwardRef<SwipeDeckHandle, Props>(function SwipeDeck({ profiles, onSwipe, onOpenProfile }, ref) {
  const { colors } = useTheme();
  const { width: screenWidth } = useWindowDimensions();
  const cardWidth = screenWidth - spacing.lg * 2;
  const pan = useRef(new Animated.ValueXY()).current;
  const busy = useRef(false);
  const latest = useRef({ top: profiles[0], onSwipe, screenWidth });
  latest.current = { top: profiles[0], onSwipe, screenWidth };

  const commit = (action: SwipeAction): void => {
    const { top, onSwipe: handler, screenWidth: w } = latest.current;
    if (!top || busy.current) return;
    busy.current = true;
    Animated.timing(pan, {
      toValue: { x: action === "like" ? w * 1.4 : -w * 1.4, y: 0 },
      duration: 220,
      useNativeDriver: false
    }).start(() => {
      pan.setValue({ x: 0, y: 0 });
      busy.current = false;
      handler(top, action);
    });
  };
  const commitRef = useRef(commit);
  commitRef.current = commit;

  useImperativeHandle(ref, () => ({ swipe: (action) => commitRef.current(action) }), []);

  const responder = useMemo(
    () =>
      PanResponder.create({
        onMoveShouldSetPanResponder: (_, g) => Math.abs(g.dx) > 10 && Math.abs(g.dx) > Math.abs(g.dy),
        onPanResponderMove: Animated.event([null, { dx: pan.x, dy: pan.y }], { useNativeDriver: false }),
        onPanResponderRelease: (_, g) => {
          const decision = decideSwipe(g.dx, g.vx, latest.current.screenWidth);
          if (decision) {
            commitRef.current(decision);
          } else {
            Animated.spring(pan, { toValue: { x: 0, y: 0 }, useNativeDriver: false, friction: 6 }).start();
          }
        },
        onPanResponderTerminate: () => {
          Animated.spring(pan, { toValue: { x: 0, y: 0 }, useNativeDriver: false }).start();
        }
      }),
    [pan]
  );

  const top = profiles[0];
  const second = profiles[1];
  if (!top) return null;

  const rotate = pan.x.interpolate({ inputRange: [-250, 0, 250], outputRange: ["-10deg", "0deg", "10deg"] });
  const likeOpacity = pan.x.interpolate({ inputRange: [0, 100], outputRange: [0, 1], extrapolate: "clamp" });
  const passOpacity = pan.x.interpolate({ inputRange: [-100, 0], outputRange: [1, 0], extrapolate: "clamp" });
  const underScale = pan.x.interpolate({ inputRange: [-200, 0, 200], outputRange: [1, 0.94, 1], extrapolate: "clamp" });

  return (
    <View style={styles.deck}>
      {second ? (
        <Animated.View key={second.userId} style={[StyleSheet.absoluteFill, { transform: [{ scale: underScale }] }]} pointerEvents="none">
          <ProfileCard profile={second} width={cardWidth} />
        </Animated.View>
      ) : null}
      <Animated.View
        key={top.userId}
        {...responder.panHandlers}
        style={[StyleSheet.absoluteFill, { transform: [{ translateX: pan.x }, { translateY: pan.y }, { rotate }] }]}
      >
        <ProfileCard profile={top} width={cardWidth} onOpenProfile={() => onOpenProfile(top)} />
        <Animated.View pointerEvents="none" style={[styles.stamp, styles.stampLeft, { opacity: likeOpacity, borderColor: colors.like }]}>
          <Text variant="heading" style={{ color: colors.like }}>
            LIKE
          </Text>
        </Animated.View>
        <Animated.View pointerEvents="none" style={[styles.stamp, styles.stampRight, { opacity: passOpacity, borderColor: "#fff" }]}>
          <Text variant="heading" tone="inverse">
            PASS
          </Text>
        </Animated.View>
      </Animated.View>
    </View>
  );
});

const styles = StyleSheet.create({
  deck: { flex: 1 },
  stamp: { position: "absolute", top: 40, borderWidth: 3, borderRadius: 10, paddingHorizontal: 12, paddingVertical: 4 },
  stampLeft: { left: 24, transform: [{ rotate: "-12deg" }] },
  stampRight: { right: 24, transform: [{ rotate: "12deg" }] }
});
