import React, { useCallback, useEffect, useMemo, useRef } from "react";
import { Animated, Dimensions, PanResponder, StyleSheet, View } from "react-native";
import type { Profile } from "../../api/types";
import { radii, spacing, useTheme } from "../../theme";
import { Button, Text } from "../../ui";
import { ProfileCard } from "./ProfileCard";

export type SwipeAction = "like" | "pass";

const SCREEN_WIDTH = Dimensions.get("window").width;
const SWIPE_THRESHOLD = SCREEN_WIDTH * 0.28;
const EXIT_DISTANCE = SCREEN_WIDTH * 1.4;

export function resolveSwipe(dx: number, threshold: number = SWIPE_THRESHOLD): SwipeAction | null {
  if (dx > threshold) return "like";
  if (dx < -threshold) return "pass";
  return null;
}

type Props = {
  profiles: Profile[];
  onSwipe: (profile: Profile, action: SwipeAction) => void;
  onOpen: (profile: Profile) => void;
};

export function SwipeDeck({ profiles, onSwipe, onOpen }: Props) {
  const { colors } = useTheme();
  const top = profiles[0];
  const second = profiles[1];
  const position = useRef(new Animated.ValueXY()).current;
  const busy = useRef(false);
  const latest = useRef({ top, onSwipe });
  latest.current = { top, onSwipe };

  const fling = useCallback(
    (action: SwipeAction) => {
      if (busy.current || !latest.current.top) return;
      busy.current = true;
      Animated.timing(position, {
        toValue: { x: action === "like" ? EXIT_DISTANCE : -EXIT_DISTANCE, y: 0 },
        duration: 220,
        useNativeDriver: true
      }).start(() => {
        const current = latest.current;
        position.setValue({ x: 0, y: 0 });
        busy.current = false;
        if (current.top) current.onSwipe(current.top, action);
      });
    },
    [position]
  );

  const responder = useMemo(
    () =>
      PanResponder.create({
        onMoveShouldSetPanResponder: (_e, g) => Math.abs(g.dx) > 8 && Math.abs(g.dx) > Math.abs(g.dy),
        onPanResponderMove: Animated.event([null, { dx: position.x, dy: position.y }], {
          useNativeDriver: false
        }),
        onPanResponderRelease: (_e, g) => {
          const action = resolveSwipe(g.dx);
          if (action) {
            fling(action);
          } else {
            Animated.spring(position, { toValue: { x: 0, y: 0 }, useNativeDriver: true }).start();
          }
        }
      }),
    [fling, position]
  );

  useEffect(() => {
    position.setValue({ x: 0, y: 0 });
  }, [top?.userId, position]);

  const rotate = position.x.interpolate({
    inputRange: [-SCREEN_WIDTH, 0, SCREEN_WIDTH],
    outputRange: ["-12deg", "0deg", "12deg"]
  });
  const likeOpacity = position.x.interpolate({
    inputRange: [0, SWIPE_THRESHOLD],
    outputRange: [0, 1],
    extrapolate: "clamp"
  });
  const passOpacity = position.x.interpolate({
    inputRange: [-SWIPE_THRESHOLD, 0],
    outputRange: [1, 0],
    extrapolate: "clamp"
  });
  const behindScale = position.x.interpolate({
    inputRange: [-SWIPE_THRESHOLD, 0, SWIPE_THRESHOLD],
    outputRange: [1, 0.94, 1],
    extrapolate: "clamp"
  });

  if (!top) return null;

  return (
    <View style={styles.wrap}>
      <View style={styles.stack}>
        {second ? (
          <Animated.View style={[styles.layer, { transform: [{ scale: behindScale }] }]} pointerEvents="none">
            <ProfileCard profile={second} onOpen={() => undefined} />
          </Animated.View>
        ) : null}
        <Animated.View
          key={top.userId}
          style={[
            styles.layer,
            { transform: [{ translateX: position.x }, { translateY: position.y }, { rotate }] }
          ]}
          {...responder.panHandlers}
        >
          <ProfileCard profile={top} onOpen={() => onOpen(top)} />
          <Animated.View
            style={[styles.stamp, styles.stampLike, { opacity: likeOpacity, borderColor: colors.success }]}
            pointerEvents="none"
          >
            <Text variant="heading" style={{ color: colors.success }}>
              LIKE
            </Text>
          </Animated.View>
          <Animated.View
            style={[styles.stamp, styles.stampPass, { opacity: passOpacity, borderColor: colors.danger }]}
            pointerEvents="none"
          >
            <Text variant="heading" style={{ color: colors.danger }}>
              PASS
            </Text>
          </Animated.View>
        </Animated.View>
      </View>
      <View style={styles.buttons}>
        <Button
          label="Pass"
          variant="secondary"
          onPress={() => fling("pass")}
          testID="pass-button"
          accessibilityLabel={`Pass on ${top.firstName}`}
          style={styles.button}
        />
        <Button
          label="Like"
          onPress={() => fling("like")}
          testID="like-button"
          accessibilityLabel={`Like ${top.firstName}`}
          style={styles.button}
        />
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { flex: 1 },
  stack: { flex: 1 },
  layer: { ...StyleSheet.absoluteFillObject },
  buttons: { flexDirection: "row", justifyContent: "space-between", paddingTop: spacing.lg },
  button: { flex: 1, marginHorizontal: spacing.xs },
  stamp: {
    position: "absolute",
    top: 40,
    borderWidth: 3,
    borderRadius: radii.sm,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.xs
  },
  stampLike: { left: 24, transform: [{ rotate: "-14deg" }] },
  stampPass: { right: 24, transform: [{ rotate: "14deg" }] }
});
