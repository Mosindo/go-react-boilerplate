import React, { useEffect, useMemo, useRef } from "react";
import { Animated, PanResponder, StyleSheet, useWindowDimensions, View } from "react-native";
import type { ProfileCard, SwipeAction } from "../../api/types";
import { useTheme } from "../../theme";
import { IconButton, Text } from "../../ui";
import { ProfileCardView } from "./ProfileCardView";

type Props = {
  cards: ProfileCard[];
  onDecision: (card: ProfileCard, action: SwipeAction) => void;
  onOpen: (card: ProfileCard) => void;
};

const SWIPE_THRESHOLD = 110;

/**
 * Swipe right to like, left to pass. Large buttons offer the same actions
 * for accessibility and for people who prefer tapping.
 */
export function SwipeDeck({ cards, onDecision, onOpen }: Props) {
  const { colors, spacing } = useTheme();
  const { width } = useWindowDimensions();
  const position = useRef(new Animated.ValueXY()).current;
  const top = cards[0];
  const topRef = useRef(top);
  topRef.current = top;

  useEffect(() => {
    position.setValue({ x: 0, y: 0 });
  }, [top?.userId, position]);

  const fling = (action: SwipeAction) => {
    const card = topRef.current;
    if (!card) {
      return;
    }
    Animated.timing(position, {
      toValue: { x: (action === "like" ? 1 : -1) * width * 1.3, y: 40 },
      duration: 220,
      useNativeDriver: true
    }).start(() => onDecision(card, action));
  };
  const flingRef = useRef(fling);
  flingRef.current = fling;

  const responder = useMemo(
    () =>
      PanResponder.create({
        onMoveShouldSetPanResponder: (_, g) => Math.abs(g.dx) > 8 && Math.abs(g.dx) > Math.abs(g.dy),
        onPanResponderMove: Animated.event([null, { dx: position.x, dy: position.y }], { useNativeDriver: false }),
        onPanResponderRelease: (_, g) => {
          if (g.dx > SWIPE_THRESHOLD || g.vx > 1.2) {
            flingRef.current("like");
          } else if (g.dx < -SWIPE_THRESHOLD || g.vx < -1.2) {
            flingRef.current("pass");
          } else {
            Animated.spring(position, { toValue: { x: 0, y: 0 }, friction: 6, useNativeDriver: true }).start();
          }
        }
      }),
    [position]
  );

  const rotate = position.x.interpolate({ inputRange: [-width, 0, width], outputRange: ["-12deg", "0deg", "12deg"] });
  const likeOpacity = position.x.interpolate({ inputRange: [0, SWIPE_THRESHOLD], outputRange: [0, 1], extrapolate: "clamp" });
  const passOpacity = position.x.interpolate({ inputRange: [-SWIPE_THRESHOLD, 0], outputRange: [1, 0], extrapolate: "clamp" });
  const nextScale = position.x.interpolate({ inputRange: [-width, 0, width], outputRange: [1, 0.95, 1], extrapolate: "clamp" });

  const second = cards[1];
  return (
    <View style={styles.flex}>
      <View style={[styles.stack, { marginHorizontal: spacing.md }]}>
        {second ? (
          <Animated.View key={second.userId} style={[StyleSheet.absoluteFill, { transform: [{ scale: nextScale }] }]}>
            <ProfileCardView card={second} />
          </Animated.View>
        ) : null}
        {top ? (
          <Animated.View
            key={top.userId}
            style={[StyleSheet.absoluteFill, { transform: [{ translateX: position.x }, { translateY: position.y }, { rotate }] }]}
            testID="discovery-card"
            {...responder.panHandlers}
          >
            <ProfileCardView card={top} onOpen={() => onOpen(top)} />
            <Animated.View pointerEvents="none" style={[styles.stamp, styles.stampLike, { borderColor: colors.success, opacity: likeOpacity }]}>
              <Text style={{ color: colors.success }} variant="title">
                J’AIME
              </Text>
            </Animated.View>
            <Animated.View pointerEvents="none" style={[styles.stamp, styles.stampPass, { borderColor: colors.textMuted, opacity: passOpacity }]}>
              <Text style={{ color: "#FFFFFF" }} variant="title">
                PASSER
              </Text>
            </Animated.View>
          </Animated.View>
        ) : null}
      </View>
      <View style={[styles.actions, { paddingVertical: spacing.md, gap: spacing.xl }]}>
        <IconButton filled icon="close" label="Passer" onPress={() => fling("pass")} size={64} testID="discovery-pass" />
        <IconButton filled icon="information" label="Voir le profil" onPress={() => top && onOpen(top)} size={48} />
        <IconButton filled icon="heart" label="J’aime" onPress={() => fling("like")} size={64} testID="discovery-like" tone="primary" />
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  stack: { flex: 1 },
  actions: { flexDirection: "row", justifyContent: "center", alignItems: "center" },
  stamp: { position: "absolute", top: 48, paddingHorizontal: 14, paddingVertical: 4, borderWidth: 3, borderRadius: 10 },
  stampLike: { left: 24, transform: [{ rotate: "-14deg" }] },
  stampPass: { right: 24, transform: [{ rotate: "14deg" }] }
});
