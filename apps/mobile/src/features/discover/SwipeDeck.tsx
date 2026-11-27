import React, { useEffect, useMemo, useRef } from "react";
import { AccessibilityInfo, Animated, PanResponder, StyleSheet, useWindowDimensions, View } from "react-native";
import type { ProfileCard, SwipeAction } from "../../api/types";
import { useTheme } from "../../theme/ThemeProvider";
import { radii, shadow, spacing } from "../../theme/palette";
import { IconButton, Text } from "../../ui";
import { ProfileCardView } from "./ProfileCardView";

type Props = {
  cards: ProfileCard[];
  onSwipe: (card: ProfileCard, action: SwipeAction) => void;
  onOpenDetails: (card: ProfileCard) => void;
};

const SWIPE_THRESHOLD = 110;

/**
 * Two-card deck: drag the top card (or use the buttons, which are also the
 * accessible path) to like or pass. Built on PanResponder + Animated.
 */
export function SwipeDeck({ cards, onSwipe, onOpenDetails }: Props) {
  const { colors, scheme } = useTheme();
  const { width } = useWindowDimensions();
  const position = useRef(new Animated.ValueXY()).current;
  const busy = useRef(false);
  const top = cards[0];
  const next = cards[1];

  const topRef = useRef(top);
  topRef.current = top;
  const onSwipeRef = useRef(onSwipe);
  onSwipeRef.current = onSwipe;

  useEffect(() => {
    position.setValue({ x: 0, y: 0 });
    busy.current = false;
  }, [top?.userId, position]);

  const fling = (action: SwipeAction) => {
    const card = topRef.current;
    if (!card || busy.current) {
      return;
    }
    busy.current = true;
    const toX = (action === "like" ? 1 : -1) * (width + 120);
    Animated.timing(position, { toValue: { x: toX, y: 40 }, duration: 220, useNativeDriver: false }).start(() => {
      onSwipeRef.current(card, action);
    });
    AccessibilityInfo.announceForAccessibility?.(action === "like" ? `Tu aimes ${card.firstName}` : `${card.firstName} passé`);
  };
  const flingRef = useRef(fling);
  flingRef.current = fling;

  const responder = useMemo(
    () =>
      PanResponder.create({
        onMoveShouldSetPanResponder: (_, g) => Math.abs(g.dx) > 8 && Math.abs(g.dx) > Math.abs(g.dy),
        onPanResponderMove: (_, g) => position.setValue({ x: g.dx, y: g.dy * 0.25 }),
        onPanResponderRelease: (_, g) => {
          if (g.dx > SWIPE_THRESHOLD || g.vx > 0.9) {
            flingRef.current("like");
          } else if (g.dx < -SWIPE_THRESHOLD || g.vx < -0.9) {
            flingRef.current("pass");
          } else {
            Animated.spring(position, { toValue: { x: 0, y: 0 }, friction: 6, useNativeDriver: false }).start();
          }
        },
        onPanResponderTerminate: () => {
          Animated.spring(position, { toValue: { x: 0, y: 0 }, friction: 6, useNativeDriver: false }).start();
        }
      }),
    [position]
  );

  const rotate = position.x.interpolate({ inputRange: [-width, 0, width], outputRange: ["-12deg", "0deg", "12deg"] });
  const likeOpacity = position.x.interpolate({ inputRange: [20, SWIPE_THRESHOLD], outputRange: [0, 1], extrapolate: "clamp" });
  const passOpacity = position.x.interpolate({ inputRange: [-SWIPE_THRESHOLD, -20], outputRange: [1, 0], extrapolate: "clamp" });
  const nextScale = position.x.interpolate({ inputRange: [-width, 0, width], outputRange: [1, 0.95, 1], extrapolate: "clamp" });

  if (!top) {
    return null;
  }

  return (
    <View style={styles.wrap}>
      <View style={styles.stack}>
        {next ? (
          <Animated.View key={next.userId} style={[styles.card, { transform: [{ scale: nextScale }] }]} pointerEvents="none">
            <ProfileCardView card={next} active={false} />
          </Animated.View>
        ) : null}
        <Animated.View
          key={top.userId}
          style={[styles.card, shadow(scheme, 2), { transform: [{ translateX: position.x }, { translateY: position.y }, { rotate }] }]}
          {...responder.panHandlers}
        >
          <ProfileCardView card={top} onOpenDetails={() => onOpenDetails(top)} />
          <Animated.View style={[styles.stamp, styles.stampLike, { borderColor: colors.success, opacity: likeOpacity }]} pointerEvents="none">
            <Text variant="heading" style={{ color: colors.success }}>
              J’AIME
            </Text>
          </Animated.View>
          <Animated.View style={[styles.stamp, styles.stampPass, { borderColor: "#FFFFFF", opacity: passOpacity }]} pointerEvents="none">
            <Text variant="heading" tone="inverse">
              PASSER
            </Text>
          </Animated.View>
        </Animated.View>
      </View>

      <View style={styles.actions}>
        <IconButton
          icon="close"
          variant="pass"
          size={64}
          iconSize={30}
          accessibilityLabel={`Passer ${top.firstName}`}
          onPress={() => fling("pass")}
          style={shadow(scheme, 1)}
          testID="swipe-pass"
        />
        <IconButton
          icon="information"
          variant="surface"
          size={48}
          accessibilityLabel={`Voir le profil de ${top.firstName}`}
          onPress={() => onOpenDetails(top)}
          style={shadow(scheme, 1)}
        />
        <IconButton
          icon="heart"
          variant="primary"
          size={64}
          iconSize={30}
          accessibilityLabel={`Aimer ${top.firstName}`}
          onPress={() => fling("like")}
          style={shadow(scheme, 1)}
          testID="swipe-like"
        />
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { flex: 1, gap: spacing.md },
  stack: { flex: 1 },
  card: { ...StyleSheet.absoluteFillObject, borderRadius: radii.xl },
  stamp: { position: "absolute", top: 40, borderWidth: 3, borderRadius: radii.sm, paddingHorizontal: 12, paddingVertical: 4 },
  stampLike: { left: 24, transform: [{ rotate: "-14deg" }] },
  stampPass: { right: 24, transform: [{ rotate: "14deg" }] },
  actions: { flexDirection: "row", justifyContent: "center", alignItems: "center", gap: spacing.xl, paddingBottom: spacing.xs }
});
