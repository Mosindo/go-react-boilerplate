import { Ionicons } from "@expo/vector-icons";
import React, { useCallback, useEffect, useMemo, useRef } from "react";
import { Animated, Dimensions, PanResponder, Pressable, StyleSheet, View } from "react-native";
import type { Card, SwipeAction } from "../api/types";
import { radii, spacing, useTheme } from "../theme";
import { Text } from "../ui";
import { ProfileCard } from "./ProfileCard";

const SCREEN_WIDTH = Dimensions.get("window").width;
const SWIPE_THRESHOLD = SCREEN_WIDTH * 0.28;

type Props = {
  cards: Card[];
  onSwipe: (card: Card, action: SwipeAction) => void;
  onOpenDetails: (card: Card) => void;
  /** Disable gestures and buttons while a decision is being saved. */
  disabled?: boolean;
};

/**
 * Stack of cards. The top card can be dragged (right = like, left = pass); the two
 * buttons below do the same for people who cannot or prefer not to swipe.
 */
export function SwipeDeck({ cards, onSwipe, onOpenDetails, disabled = false }: Props) {
  const { colors } = useTheme();
  const position = useRef(new Animated.ValueXY()).current;
  const top = cards[0];
  const next = cards[1];
  const latest = useRef({ top, onSwipe, disabled });
  latest.current = { top, onSwipe, disabled };

  const fling = useCallback(
    (action: SwipeAction) => {
      const current = latest.current;
      if (!current.top || current.disabled) {
        return;
      }
      const card = current.top;
      Animated.timing(position, {
        toValue: { x: action === "like" ? SCREEN_WIDTH * 1.3 : -SCREEN_WIDTH * 1.3, y: 0 },
        duration: 220,
        useNativeDriver: true
      }).start(() => {
        position.setValue({ x: 0, y: 0 });
        current.onSwipe(card, action);
      });
    },
    [position]
  );

  const pan = useMemo(
    () =>
      PanResponder.create({
        onMoveShouldSetPanResponder: (_, g) => !latest.current.disabled && Math.abs(g.dx) > 8 && Math.abs(g.dx) > Math.abs(g.dy),
        onPanResponderMove: Animated.event([null, { dx: position.x, dy: position.y }], { useNativeDriver: false }),
        onPanResponderRelease: (_, g) => {
          if (g.dx > SWIPE_THRESHOLD) {
            fling("like");
          } else if (g.dx < -SWIPE_THRESHOLD) {
            fling("pass");
          } else {
            Animated.spring(position, { toValue: { x: 0, y: 0 }, useNativeDriver: false, friction: 6 }).start();
          }
        },
        onPanResponderTerminate: () => {
          Animated.spring(position, { toValue: { x: 0, y: 0 }, useNativeDriver: false }).start();
        }
      }),
    [fling, position]
  );

  useEffect(() => {
    position.setValue({ x: 0, y: 0 });
  }, [top?.userId, position]);

  const rotate = position.x.interpolate({ inputRange: [-SCREEN_WIDTH, 0, SCREEN_WIDTH], outputRange: ["-12deg", "0deg", "12deg"] });
  const likeOpacity = position.x.interpolate({ inputRange: [0, SWIPE_THRESHOLD], outputRange: [0, 1], extrapolate: "clamp" });
  const passOpacity = position.x.interpolate({ inputRange: [-SWIPE_THRESHOLD, 0], outputRange: [1, 0], extrapolate: "clamp" });
  const nextScale = position.x.interpolate({
    inputRange: [-SWIPE_THRESHOLD, 0, SWIPE_THRESHOLD],
    outputRange: [1, 0.94, 1],
    extrapolate: "clamp"
  });

  if (!top) {
    return null;
  }

  return (
    <View style={styles.flex}>
      <View style={styles.stack}>
        {next ? (
          <Animated.View style={[styles.layer, { transform: [{ scale: nextScale }] }]} pointerEvents="none">
            <ProfileCard card={next} />
          </Animated.View>
        ) : null}
        <Animated.View
          key={top.userId}
          {...pan.panHandlers}
          style={[styles.layer, { transform: [{ translateX: position.x }, { translateY: position.y }, { rotate }] }]}
        >
          <ProfileCard card={top} onOpenDetails={() => onOpenDetails(top)} />
          <Animated.View style={[styles.stamp, styles.likeStamp, { opacity: likeOpacity, borderColor: colors.like }]} pointerEvents="none">
            <Text variant="heading" style={{ color: colors.like }}>
              J'AIME
            </Text>
          </Animated.View>
          <Animated.View
            style={[styles.stamp, styles.passStamp, { opacity: passOpacity, borderColor: colors.danger }]}
            pointerEvents="none"
          >
            <Text variant="heading" style={{ color: colors.danger }}>
              PASSER
            </Text>
          </Animated.View>
        </Animated.View>
      </View>

      <View style={styles.actions}>
        <Pressable
          accessibilityRole="button"
          accessibilityLabel={`Passer ${top.firstName}`}
          disabled={disabled}
          onPress={() => fling("pass")}
          style={[styles.action, { backgroundColor: colors.surface, borderColor: colors.border, opacity: disabled ? 0.5 : 1 }]}
        >
          <Ionicons name="close" size={32} color={colors.danger} />
        </Pressable>
        <Pressable
          accessibilityRole="button"
          accessibilityLabel={`Aimer ${top.firstName}`}
          disabled={disabled}
          onPress={() => fling("like")}
          style={[styles.action, styles.likeAction, { backgroundColor: colors.primary, opacity: disabled ? 0.5 : 1 }]}
        >
          <Ionicons name="heart" size={32} color={colors.onPrimary} />
        </Pressable>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1, gap: spacing.lg },
  stack: { flex: 1 },
  layer: { ...StyleSheet.absoluteFillObject },
  stamp: {
    position: "absolute",
    top: 40,
    borderWidth: 4,
    borderRadius: radii.md,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.xs
  },
  likeStamp: { left: 24, transform: [{ rotate: "-14deg" }] },
  passStamp: { right: 24, transform: [{ rotate: "14deg" }] },
  actions: { flexDirection: "row", justifyContent: "center", gap: spacing.xl, paddingBottom: spacing.sm },
  action: { width: 68, height: 68, borderRadius: 34, borderWidth: 1.5, alignItems: "center", justifyContent: "center" },
  likeAction: { width: 76, height: 76, borderRadius: 38, borderWidth: 0 }
});
