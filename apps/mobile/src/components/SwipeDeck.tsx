import { Ionicons } from "@expo/vector-icons";
import { forwardRef, useCallback, useEffect, useImperativeHandle, useMemo, useRef, useState } from "react";
import { AccessibilityInfo, Animated, PanResponder, Pressable, StyleSheet, View, useWindowDimensions } from "react-native";
import type { ProfileCard, SwipeAction } from "../api/types";
import { Text } from "../shared/ui";
import { radii, spacing, useTheme } from "../theme";
import { ProfileCardView } from "./ProfileCardView";

export type SwipeDeckHandle = { swipe: (action: SwipeAction) => void };

type Props = {
  cards: ProfileCard[];
  onSwipe: (card: ProfileCard, action: SwipeAction) => void;
  onOpenDetails: (card: ProfileCard) => void;
};

const SWIPE_THRESHOLD = 0.28; // fraction of the screen width

/**
 * Stack of profile cards. Drag right to like, left to pass — or use the buttons, which are the accessible
 * alternative to the gesture. The card behind is rendered underneath so the next profile appears instantly.
 */
export const SwipeDeck = forwardRef<SwipeDeckHandle, Props>(function SwipeDeck({ cards, onSwipe, onOpenDetails }, ref) {
  const { width } = useWindowDimensions();
  const { colors } = useTheme();
  const position = useRef(new Animated.ValueXY()).current;
  const [reduceMotion, setReduceMotion] = useState(false);
  const busy = useRef(false);
  const top = cards[0];
  const next = cards[1];

  useEffect(() => {
    void AccessibilityInfo.isReduceMotionEnabled().then(setReduceMotion);
    const sub = AccessibilityInfo.addEventListener("reduceMotionChanged", setReduceMotion);
    return () => sub.remove();
  }, []);

  const latest = useRef({ top, onSwipe, width, reduceMotion });
  useEffect(() => {
    latest.current = { top, onSwipe, width, reduceMotion };
  });

  const fling = useCallback(
    (action: SwipeAction) => {
      const { top: card, onSwipe: notify, width: w, reduceMotion: reduce } = latest.current;
      if (!card || busy.current) return;
      busy.current = true;
      const finish = () => {
        position.setValue({ x: 0, y: 0 });
        busy.current = false;
        notify(card, action);
      };
      if (reduce) {
        finish();
        return;
      }
      Animated.timing(position, { toValue: { x: action === "like" ? w * 1.4 : -w * 1.4, y: 0 }, duration: 220, useNativeDriver: false }).start(finish);
    },
    [position]
  );

  useImperativeHandle(ref, () => ({ swipe: fling }), [fling]);

  const responder = useMemo(
    () =>
      PanResponder.create({
        // Let taps (photo navigation, info button) through; only claim real horizontal drags.
        onMoveShouldSetPanResponder: (_, g) => Math.abs(g.dx) > 8 && Math.abs(g.dx) > Math.abs(g.dy),
        onPanResponderMove: (_, g) => position.setValue({ x: g.dx, y: g.dy * 0.2 }),
        onPanResponderRelease: (_, g) => {
          const limit = latest.current.width * SWIPE_THRESHOLD;
          if (g.dx > limit || g.vx > 0.9) fling("like");
          else if (g.dx < -limit || g.vx < -0.9) fling("pass");
          else Animated.spring(position, { toValue: { x: 0, y: 0 }, useNativeDriver: false, friction: 6 }).start();
        },
        onPanResponderTerminate: () => Animated.spring(position, { toValue: { x: 0, y: 0 }, useNativeDriver: false }).start()
      }),
    [fling, position]
  );

  const rotate = position.x.interpolate({ inputRange: [-width, 0, width], outputRange: ["-12deg", "0deg", "12deg"] });
  const likeOpacity = position.x.interpolate({ inputRange: [0, width * 0.25], outputRange: [0, 1], extrapolate: "clamp" });
  const passOpacity = position.x.interpolate({ inputRange: [-width * 0.25, 0], outputRange: [1, 0], extrapolate: "clamp" });
  const nextScale = position.x.interpolate({ inputRange: [-width, 0, width], outputRange: [1, 0.94, 1], extrapolate: "clamp" });

  if (!top) return null;

  return (
    <View style={styles.wrapper}>
      <View style={styles.stack}>
        {next ? (
          <Animated.View style={[styles.layer, { transform: [{ scale: nextScale }] }]} pointerEvents="none">
            <ProfileCardView profile={next} interactive={false} />
          </Animated.View>
        ) : null}
        <Animated.View
          style={[styles.layer, { transform: [{ translateX: position.x }, { translateY: position.y }, { rotate }] }]}
          {...responder.panHandlers}
        >
          <ProfileCardView profile={top} onOpenDetails={() => onOpenDetails(top)} />
          <Animated.View style={[styles.stamp, styles.likeStamp, { opacity: likeOpacity, borderColor: colors.success }]} pointerEvents="none">
            <Text variant="title" style={{ color: colors.success }}>
              LIKE
            </Text>
          </Animated.View>
          <Animated.View style={[styles.stamp, styles.passStamp, { opacity: passOpacity, borderColor: "#FFFFFF" }]} pointerEvents="none">
            <Text variant="title" tone="onImage">
              PASSER
            </Text>
          </Animated.View>
        </Animated.View>
      </View>

      <View style={styles.actions}>
        <RoundButton testID="swipe-pass" icon="close" label={`Passer ${top.firstName}`} color={colors.textMuted} bg={colors.surface} border={colors.border} onPress={() => fling("pass")} />
        <RoundButton testID="swipe-like" icon="heart" label={`Liker ${top.firstName}`} color={colors.onPrimary} bg={colors.primary} border={colors.primary} large onPress={() => fling("like")} />
      </View>
    </View>
  );
});

function RoundButton({ icon, label, color, bg, border, onPress, large, testID }: { icon: keyof typeof Ionicons.glyphMap; label: string; color: string; bg: string; border: string; onPress: () => void; large?: boolean; testID?: string }) {
  const size = large ? 72 : 60;
  return (
    <Pressable
      testID={testID}
      accessibilityRole="button"
      accessibilityLabel={label}
      onPress={onPress}
      style={({ pressed }) => [styles.round, { width: size, height: size, borderRadius: size / 2, backgroundColor: bg, borderColor: border, transform: [{ scale: pressed ? 0.92 : 1 }] }]}
    >
      <Ionicons name={icon} size={large ? 34 : 30} color={color} />
    </Pressable>
  );
}

const styles = StyleSheet.create({
  wrapper: { flex: 1, gap: spacing.lg },
  stack: { flex: 1 },
  layer: { ...StyleSheet.absoluteFillObject },
  actions: { flexDirection: "row", justifyContent: "center", alignItems: "center", gap: spacing.xxl, paddingBottom: spacing.sm },
  round: { alignItems: "center", justifyContent: "center", borderWidth: 1.5, shadowColor: "#000", shadowOpacity: 0.15, shadowRadius: 8, shadowOffset: { width: 0, height: 3 }, elevation: 4 },
  stamp: { position: "absolute", top: 56, borderWidth: 4, borderRadius: radii.md, paddingHorizontal: spacing.md, paddingVertical: 2 },
  likeStamp: { left: 24, transform: [{ rotate: "-14deg" }] },
  passStamp: { right: 24, transform: [{ rotate: "14deg" }] }
});
