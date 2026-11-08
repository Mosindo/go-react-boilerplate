import React, { useCallback, useEffect, useMemo, useRef } from "react";
import { Animated, PanResponder, StyleSheet, useWindowDimensions, View } from "react-native";
import type { ProfileCard, SwipeAction } from "../../api/types";
import { decideSwipe } from "../../lib/swipe";
import { radii, spacing, useTheme } from "../../theme/theme";
import { IconButton } from "../../ui/Button";
import { Text } from "../../ui/Text";
import { ProfileCardFace } from "./ProfileCardFace";

type Props = {
  cards: ProfileCard[];
  onSwipe: (card: ProfileCard, action: SwipeAction) => void;
  onOpenProfile: (card: ProfileCard) => void;
};

// Position is driven by JS (PanResponder), so native-driven animations cannot share the same Animated.Value.
const useNative = false;

export function SwipeDeck({ cards, onSwipe, onOpenProfile }: Props) {
  const { colors } = useTheme();
  const { width } = useWindowDimensions();
  const position = useRef(new Animated.ValueXY()).current;
  const top = cards[0];
  const next = cards[1];
  const busy = useRef(false);

  // Always use the latest card/callback inside the long-lived PanResponder.
  const latest = useRef({ top, onSwipe });
  latest.current = { top, onSwipe };

  useEffect(() => {
    position.setValue({ x: 0, y: 0 });
    busy.current = false;
  }, [top?.userId, position]);

  const flyOut = useCallback(
    (action: SwipeAction) => {
      if (busy.current || !latest.current.top) return;
      busy.current = true;
      const card = latest.current.top;
      Animated.timing(position, {
        toValue: { x: (action === "like" ? 1 : -1) * width * 1.3, y: 0 },
        duration: 200,
        useNativeDriver: useNative
      }).start(() => {
        latest.current.onSwipe(card, action);
      });
    },
    [position, width]
  );

  const pan = useMemo(
    () =>
      PanResponder.create({
        onMoveShouldSetPanResponder: (_, g) => Math.abs(g.dx) > 8 && Math.abs(g.dx) > Math.abs(g.dy),
        onPanResponderMove: Animated.event([null, { dx: position.x, dy: position.y }], {
          useNativeDriver: false
        }),
        onPanResponderRelease: (_, g) => {
          const decision = decideSwipe(g.dx, g.dy, g.vx);
          if (decision) flyOut(decision);
          else
            Animated.spring(position, {
              toValue: { x: 0, y: 0 },
              friction: 6,
              useNativeDriver: useNative
            }).start();
        },
        onPanResponderTerminate: () =>
          Animated.spring(position, { toValue: { x: 0, y: 0 }, useNativeDriver: useNative }).start()
      }),
    [position, flyOut]
  );

  const rotate = position.x.interpolate({
    inputRange: [-width, 0, width],
    outputRange: ["-12deg", "0deg", "12deg"]
  });
  const likeOpacity = position.x.interpolate({
    inputRange: [20, 120],
    outputRange: [0, 1],
    extrapolate: "clamp"
  });
  const passOpacity = position.x.interpolate({
    inputRange: [-120, -20],
    outputRange: [1, 0],
    extrapolate: "clamp"
  });
  const nextScale = position.x.interpolate({
    inputRange: [-width, 0, width],
    outputRange: [1, 0.94, 1],
    extrapolate: "clamp"
  });

  return (
    <View style={styles.wrap}>
      <View style={styles.stack}>
        {next ? (
          <Animated.View
            key={next.userId}
            style={[styles.card, { backgroundColor: colors.surfaceMuted, transform: [{ scale: nextScale }] }]}
            pointerEvents="none"
          >
            <ProfileCardFace card={next} onOpenProfile={() => undefined} />
          </Animated.View>
        ) : null}
        {top ? (
          <Animated.View
            key={top.userId}
            {...pan.panHandlers}
            testID="deck-top-card"
            style={[
              styles.card,
              {
                backgroundColor: colors.surfaceMuted,
                transform: [{ translateX: position.x }, { translateY: position.y }, { rotate }]
              }
            ]}
          >
            <ProfileCardFace card={top} onOpenProfile={() => onOpenProfile(top)} />
            <Animated.View
              style={[styles.stamp, styles.stampLike, { opacity: likeOpacity, borderColor: "#7DE0B0" }]}
              pointerEvents="none"
            >
              <Text variant="title" style={{ color: "#7DE0B0" }}>
                J&apos;AIME
              </Text>
            </Animated.View>
            <Animated.View
              style={[styles.stamp, styles.stampPass, { opacity: passOpacity, borderColor: "#F5E9E3" }]}
              pointerEvents="none"
            >
              <Text variant="title" style={{ color: "#F5E9E3" }}>
                PASSER
              </Text>
            </Animated.View>
          </Animated.View>
        ) : null}
      </View>

      <View style={styles.actions}>
        <IconButton
          icon="close"
          label="Passer"
          size={64}
          color={colors.pass}
          background={colors.surface}
          onPress={() => flyOut("pass")}
          disabled={!top}
          testID="deck-pass"
        />
        <IconButton
          icon="heart"
          label="J'aime"
          size={72}
          color={colors.onPrimary}
          background={colors.primary}
          onPress={() => flyOut("like")}
          disabled={!top}
          testID="deck-like"
        />
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { flex: 1 },
  stack: { flex: 1, marginHorizontal: spacing.md, marginTop: spacing.xs },
  card: { ...StyleSheet.absoluteFillObject, borderRadius: radii.lg, overflow: "hidden" },
  stamp: {
    position: "absolute",
    top: 48,
    borderWidth: 4,
    borderRadius: radii.md,
    paddingHorizontal: spacing.sm
  },
  stampLike: { left: 24, transform: [{ rotate: "-14deg" }] },
  stampPass: { right: 24, transform: [{ rotate: "14deg" }] },
  actions: {
    flexDirection: "row",
    justifyContent: "center",
    alignItems: "center",
    gap: spacing.xl,
    paddingVertical: spacing.md
  }
});
