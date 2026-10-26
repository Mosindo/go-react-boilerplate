import React, { useEffect, useMemo, useRef } from "react";
import { Animated, PanResponder, Pressable, StyleSheet, View, useWindowDimensions } from "react-native";
import type { PublicProfile } from "../api";
import { radius, spacing, useTheme } from "../theme";
import { ProfileCard } from "./ProfileCard";
import { Text } from "./Text";

type Props = {
  profiles: PublicProfile[];
  onSwipe: (profile: PublicProfile, action: "like" | "pass") => void;
  onInfo: (profile: PublicProfile) => void;
};

const SWIPE_THRESHOLD = 110;

/**
 * Card stack. Dragging a card past the threshold likes (right) or passes
 * (left); the round buttons trigger the same actions for accessibility.
 */
export function SwipeDeck({ profiles, onSwipe, onInfo }: Props) {
  const theme = useTheme();
  const { width } = useWindowDimensions();
  const top = profiles[0];
  const next = profiles[1];
  const position = useRef(new Animated.ValueXY()).current;
  const busy = useRef(false);
  const topRef = useRef(top);
  const onSwipeRef = useRef(onSwipe);

  useEffect(() => {
    topRef.current = top;
    onSwipeRef.current = onSwipe;
  });

  // A fresh card always starts centred.
  useEffect(() => {
    position.setValue({ x: 0, y: 0 });
    busy.current = false;
  }, [top?.userId, position]);

  const fling = (action: "like" | "pass") => {
    const card = topRef.current;
    if (!card || busy.current) return;
    busy.current = true;
    Animated.timing(position, {
      toValue: { x: (action === "like" ? 1 : -1) * width * 1.3, y: 40 },
      duration: 220,
      useNativeDriver: false
    }).start(() => onSwipeRef.current(card, action));
  };
  const flingRef = useRef(fling);
  useEffect(() => {
    flingRef.current = fling;
  });

  const responder = useMemo(
    () =>
      PanResponder.create({
        onMoveShouldSetPanResponder: (_, g) => Math.abs(g.dx) > 8 && Math.abs(g.dx) > Math.abs(g.dy),
        onPanResponderMove: (_, g) => position.setValue({ x: g.dx, y: g.dy * 0.2 }),
        onPanResponderRelease: (_, g) => {
          if (g.dx > SWIPE_THRESHOLD) flingRef.current("like");
          else if (g.dx < -SWIPE_THRESHOLD) flingRef.current("pass");
          else Animated.spring(position, { toValue: { x: 0, y: 0 }, useNativeDriver: false }).start();
        }
      }),
    [position]
  );

  const rotate = position.x.interpolate({
    inputRange: [-width, 0, width],
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

  if (!top) return null;

  return (
    <View style={styles.wrap}>
      <View style={styles.stack}>
        {next ? (
          <View style={[styles.layer, styles.behind]} pointerEvents="none">
            <ProfileCard profile={next} />
          </View>
        ) : null}
        <Animated.View
          key={top.userId}
          style={[
            styles.layer,
            { transform: [{ translateX: position.x }, { translateY: position.y }, { rotate }] }
          ]}
          {...responder.panHandlers}
        >
          <ProfileCard profile={top} onInfo={() => onInfo(top)} />
          <Animated.View
            style={[styles.stamp, styles.stampLike, { opacity: likeOpacity, borderColor: theme.like }]}
            pointerEvents="none"
          >
            <Text variant="title" color={theme.like}>
              J&apos;AIME
            </Text>
          </Animated.View>
          <Animated.View
            style={[styles.stamp, styles.stampPass, { opacity: passOpacity, borderColor: "#FFFFFF" }]}
            pointerEvents="none"
          >
            <Text variant="title" color="#FFFFFF">
              PASSE
            </Text>
          </Animated.View>
        </Animated.View>
      </View>

      <View style={styles.actions}>
        <RoundButton
          label="Passer"
          glyph="✕"
          color={theme.pass}
          onPress={() => fling("pass")}
          testID="pass-button"
        />
        <RoundButton
          label="Voir le profil"
          glyph="i"
          color={theme.primary}
          small
          onPress={() => onInfo(top)}
          testID="info-button"
        />
        <RoundButton
          label="Liker"
          glyph="♥"
          color={theme.like}
          onPress={() => fling("like")}
          testID="like-button"
        />
      </View>
    </View>
  );
}

function RoundButton({
  label,
  glyph,
  color,
  onPress,
  small,
  testID
}: {
  label: string;
  glyph: string;
  color: string;
  onPress: () => void;
  small?: boolean;
  testID: string;
}) {
  const theme = useTheme();
  const size = small ? 48 : 64;
  return (
    <Pressable
      testID={testID}
      accessibilityRole="button"
      accessibilityLabel={label}
      onPress={onPress}
      style={({ pressed }) => [
        styles.round,
        {
          width: size,
          height: size,
          borderRadius: size / 2,
          borderColor: color,
          backgroundColor: pressed ? theme.surfaceAlt : theme.surface
        }
      ]}
    >
      <Text
        style={{ fontSize: small ? 20 : 28, lineHeight: small ? 24 : 32, fontWeight: "700" }}
        color={color}
      >
        {glyph}
      </Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  wrap: { flex: 1, gap: spacing.lg },
  stack: { flex: 1 },
  layer: { ...StyleSheet.absoluteFillObject },
  behind: { transform: [{ scale: 0.95 }], opacity: 0.9 },
  actions: {
    flexDirection: "row",
    justifyContent: "center",
    alignItems: "center",
    gap: spacing.xl,
    paddingBottom: spacing.sm
  },
  round: { alignItems: "center", justifyContent: "center", borderWidth: 2 },
  stamp: {
    position: "absolute",
    top: 40,
    borderWidth: 3,
    borderRadius: radius.md,
    paddingHorizontal: spacing.md,
    paddingVertical: 2
  },
  stampLike: { left: 24, transform: [{ rotate: "-14deg" }] },
  stampPass: { right: 24, transform: [{ rotate: "14deg" }] }
});
