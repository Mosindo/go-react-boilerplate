import React, { useCallback, useEffect, useMemo, useRef } from "react";
import { Animated, Dimensions, PanResponder, StyleSheet, View } from "react-native";
import type { Card } from "../api/types";
import { Button, Text, colors, spacing } from "../shared/ui";
import { ProfileCardView } from "./ProfileCardView";

const SCREEN_WIDTH = Dimensions.get("window").width;
const SWIPE_THRESHOLD = SCREEN_WIDTH * 0.28;

type SwipeDeckProps = {
  cards: Card[];
  onLike: (card: Card) => void;
  onPass: (card: Card) => void;
  onOpenProfile: (card: Card) => void;
  disabled?: boolean;
};

/**
 * Top card is draggable (swipe right = like, left = pass). The same actions
 * are available as buttons, which keeps the screen usable without gestures.
 */
export function SwipeDeck({ cards, onLike, onPass, onOpenProfile, disabled = false }: SwipeDeckProps) {
  const position = useRef(new Animated.ValueXY()).current;
  const top = cards[0];
  const next = cards[1];

  // Handlers are read through a ref so the PanResponder (created once) never goes stale.
  const handlers = useRef({ top, onLike, onPass, disabled });
  handlers.current = { top, onLike, onPass, disabled };

  const fling = useCallback(
    (direction: 1 | -1) => {
      const { top: card, onLike: like, onPass: pass } = handlers.current;
      if (!card) return;
      Animated.timing(position, { toValue: { x: direction * SCREEN_WIDTH * 1.4, y: 0 }, duration: 180, useNativeDriver: false }).start(
        () => {
          position.setValue({ x: 0, y: 0 });
          if (direction === 1) like(card);
          else pass(card);
        }
      );
    },
    [position]
  );

  const responder = useMemo(
    () =>
      PanResponder.create({
        onMoveShouldSetPanResponder: (_e, g) => !handlers.current.disabled && Math.abs(g.dx) > 8 && Math.abs(g.dx) > Math.abs(g.dy),
        onPanResponderMove: Animated.event([null, { dx: position.x, dy: position.y }], { useNativeDriver: false }),
        onPanResponderRelease: (_e, g) => {
          if (g.dx > SWIPE_THRESHOLD) fling(1);
          else if (g.dx < -SWIPE_THRESHOLD) fling(-1);
          else Animated.spring(position, { toValue: { x: 0, y: 0 }, useNativeDriver: false }).start();
        }
      }),
    [fling, position]
  );

  useEffect(() => {
    position.setValue({ x: 0, y: 0 });
  }, [top?.id, position]);

  if (!top) return null;

  const rotate = position.x.interpolate({ inputRange: [-SCREEN_WIDTH, 0, SCREEN_WIDTH], outputRange: ["-12deg", "0deg", "12deg"] });
  const likeOpacity = position.x.interpolate({ inputRange: [0, SWIPE_THRESHOLD], outputRange: [0, 1], extrapolate: "clamp" });
  const passOpacity = position.x.interpolate({ inputRange: [-SWIPE_THRESHOLD, 0], outputRange: [1, 0], extrapolate: "clamp" });

  return (
    <View style={styles.container}>
      <View style={styles.stack}>
        {next ? (
          <View pointerEvents="none" style={[styles.layer, styles.behind]}>
            <ProfileCardView card={next} />
          </View>
        ) : null}
        <Animated.View
          {...responder.panHandlers}
          style={[styles.layer, { transform: [{ translateX: position.x }, { translateY: position.y }, { rotate }] }]}
        >
          <ProfileCardView card={top} onOpenProfile={() => onOpenProfile(top)} />
          <Animated.View pointerEvents="none" style={[styles.stamp, styles.stampLike, { opacity: likeOpacity }]}>
            <Text style={styles.stampText} tone="success" variant="heading" weight="bold">
              J'AIME
            </Text>
          </Animated.View>
          <Animated.View pointerEvents="none" style={[styles.stamp, styles.stampPass, { opacity: passOpacity }]}>
            <Text style={styles.stampText} tone="danger" variant="heading" weight="bold">
              PASSER
            </Text>
          </Animated.View>
        </Animated.View>
      </View>
      <View style={styles.actions}>
        <Button
          accessibilityLabel="Passer"
          disabled={disabled}
          label="Passer"
          onPress={() => fling(-1)}
          style={styles.action}
          variant="outline"
        />
        <Button
          accessibilityLabel="Voir le profil"
          disabled={disabled}
          label="Profil"
          onPress={() => onOpenProfile(top)}
          style={styles.action}
          variant="ghost"
        />
        <Button
          accessibilityLabel="J'aime"
          disabled={disabled}
          label="J'aime"
          onPress={() => fling(1)}
          style={styles.action}
          variant="primary"
        />
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, gap: spacing.lg },
  stack: { width: "100%" },
  layer: { width: "100%" },
  behind: { position: "absolute", top: 0, transform: [{ scale: 0.95 }], opacity: 0.7 },
  actions: { flexDirection: "row", gap: spacing.md, alignItems: "center" },
  action: { flex: 1 },
  stamp: {
    position: "absolute",
    top: spacing.xl,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.xs,
    borderWidth: 3,
    borderRadius: 10,
    backgroundColor: "rgba(255,255,255,0.85)"
  },
  stampLike: { left: spacing.xl, borderColor: colors.success, transform: [{ rotate: "-12deg" }] },
  stampPass: { right: spacing.xl, borderColor: colors.danger, transform: [{ rotate: "12deg" }] },
  stampText: { letterSpacing: 1 }
});
