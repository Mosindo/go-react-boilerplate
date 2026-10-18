import React, { useCallback, useEffect, useMemo, useRef } from "react";
import { Animated, PanResponder, Platform, Pressable, StyleSheet, View } from "react-native";
import type { PublicProfile } from "../api/types";
import { Text, colors, radii, shadows, spacing } from "../shared/ui";
import { ProfileCard } from "./ProfileCard";

type SwipeAction = "like" | "pass";

type SwipeDeckProps = {
  profiles: PublicProfile[];
  disabled?: boolean;
  onSwipe: (profile: PublicProfile, action: SwipeAction) => void;
  onOpenProfile: (profile: PublicProfile) => void;
};

const SWIPE_THRESHOLD = 110;
const OFF_SCREEN = 520;
const useNativeDriver = Platform.OS !== "web";

/**
 * Card stack with drag-to-swipe plus explicit buttons, so every action stays
 * reachable without gestures (accessibility, mouse and keyboard users).
 */
export function SwipeDeck({ profiles, disabled = false, onSwipe, onOpenProfile }: SwipeDeckProps) {
  const position = useRef(new Animated.ValueXY()).current;
  const top = profiles[0];
  const next = profiles[1];

  // The pan responder is created once, so it reads the latest props through a ref.
  const latest = useRef({ top, disabled, onSwipe });
  latest.current = { top, disabled, onSwipe };
  const busy = useRef(false);

  const fling = useCallback(
    (action: SwipeAction) => {
      const { top: current, disabled: isDisabled, onSwipe: notify } = latest.current;
      if (!current || isDisabled || busy.current) {
        return;
      }
      busy.current = true;
      Animated.timing(position, {
        toValue: { x: action === "like" ? OFF_SCREEN : -OFF_SCREEN, y: 40 },
        duration: 180,
        useNativeDriver
      }).start(() => {
        notify(current, action);
        position.setValue({ x: 0, y: 0 });
        busy.current = false;
      });
    },
    [position]
  );

  const panResponder = useMemo(
    () =>
      PanResponder.create({
        onMoveShouldSetPanResponder: (_, g) =>
          !latest.current.disabled && Math.abs(g.dx) > 8 && Math.abs(g.dx) > Math.abs(g.dy),
        onPanResponderMove: Animated.event([null, { dx: position.x, dy: position.y }], { useNativeDriver: false }),
        onPanResponderRelease: (_, g) => {
          if (g.dx > SWIPE_THRESHOLD) {
            fling("like");
          } else if (g.dx < -SWIPE_THRESHOLD) {
            fling("pass");
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

  // A new top card always starts centred.
  useEffect(() => {
    position.setValue({ x: 0, y: 0 });
  }, [top?.userId, position]);

  const rotate = position.x.interpolate({
    inputRange: [-OFF_SCREEN, 0, OFF_SCREEN],
    outputRange: ["-14deg", "0deg", "14deg"]
  });
  const likeOpacity = position.x.interpolate({
    inputRange: [20, SWIPE_THRESHOLD],
    outputRange: [0, 1],
    extrapolate: "clamp"
  });
  const passOpacity = position.x.interpolate({
    inputRange: [-SWIPE_THRESHOLD, -20],
    outputRange: [1, 0],
    extrapolate: "clamp"
  });
  const nextScale = position.x.interpolate({
    inputRange: [-SWIPE_THRESHOLD, 0, SWIPE_THRESHOLD],
    outputRange: [1, 0.94, 1],
    extrapolate: "clamp"
  });

  if (!top) {
    return null;
  }

  return (
    <View style={styles.wrap}>
      <View style={styles.stack}>
        {next ? (
          <Animated.View pointerEvents="none" style={[styles.layer, { transform: [{ scale: nextScale }] }]}>
            <ProfileCard profile={next} />
          </Animated.View>
        ) : null}
        <Animated.View
          {...panResponder.panHandlers}
          style={[styles.layer, { transform: [{ translateX: position.x }, { translateY: position.y }, { rotate }] }]}
          testID="deck-card"
        >
          <ProfileCard onOpenDetails={() => onOpenProfile(top)} profile={top} />
          <Animated.View pointerEvents="none" style={[styles.stamp, styles.stampLike, { opacity: likeOpacity }]}>
            <Text style={styles.stampText} tone="success" weight="bold">
              J'AIME
            </Text>
          </Animated.View>
          <Animated.View pointerEvents="none" style={[styles.stamp, styles.stampPass, { opacity: passOpacity }]}>
            <Text style={styles.stampText} tone="muted" weight="bold">
              PASSER
            </Text>
          </Animated.View>
        </Animated.View>
      </View>

      <View style={styles.actions}>
        <RoundButton
          disabled={disabled}
          glyph="✕"
          label="Passer"
          onPress={() => fling("pass")}
          testID="swipe-pass"
          tone={colors.pass}
        />
        <RoundButton
          disabled={disabled}
          glyph="i"
          label="Voir le profil complet"
          onPress={() => onOpenProfile(top)}
          size={48}
          testID="swipe-info"
          tone={colors.secondary}
        />
        <RoundButton
          disabled={disabled}
          glyph="♥"
          label="J'aime"
          onPress={() => fling("like")}
          testID="swipe-like"
          tone={colors.like}
        />
      </View>
    </View>
  );
}

type RoundButtonProps = {
  glyph: string;
  label: string;
  tone: string;
  size?: number;
  disabled?: boolean;
  onPress: () => void;
  testID: string;
};

function RoundButton({ glyph, label, tone, size = 64, disabled, onPress, testID }: RoundButtonProps) {
  return (
    <Pressable
      accessibilityLabel={label}
      accessibilityRole="button"
      disabled={disabled}
      onPress={onPress}
      style={({ pressed }) => [
        styles.round,
        { width: size, height: size, borderRadius: size / 2, borderColor: tone },
        disabled ? styles.roundDisabled : null,
        pressed ? styles.roundPressed : null
      ]}
      testID={testID}
    >
      <Text style={{ color: tone, fontSize: size * 0.42, lineHeight: size * 0.5 }} weight="bold">
        {glyph}
      </Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  wrap: { flex: 1, gap: spacing.lg },
  stack: { flex: 1 },
  layer: { ...StyleSheet.absoluteFillObject },
  stamp: {
    position: "absolute",
    top: spacing.xxl,
    borderWidth: 3,
    borderRadius: radii.md,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.xs,
    backgroundColor: "rgba(255,255,255,0.85)"
  },
  stampLike: { left: spacing.xl, borderColor: colors.like, transform: [{ rotate: "-12deg" }] },
  stampPass: { right: spacing.xl, borderColor: colors.pass, transform: [{ rotate: "12deg" }] },
  stampText: { fontSize: 22, lineHeight: 28, letterSpacing: 1 },
  actions: { flexDirection: "row", justifyContent: "center", alignItems: "center", gap: spacing.xl },
  round: {
    alignItems: "center",
    justifyContent: "center",
    borderWidth: 2,
    backgroundColor: colors.backgroundElevated,
    ...shadows.card
  },
  roundDisabled: { opacity: 0.5 },
  roundPressed: { transform: [{ scale: 0.94 }] }
});
