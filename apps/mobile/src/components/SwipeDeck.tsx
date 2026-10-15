import React, { forwardRef, useImperativeHandle, useMemo, useRef } from "react";
import { Animated, PanResponder, Pressable, View, useWindowDimensions } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import type { Card, SwipeAction } from "../api/types";
import { distanceLabel } from "./ProfileBody";
import { PhotoCarousel } from "./PhotoCarousel";
import { Chip, Text, radii, spacing } from "../shared/ui";
import { useTheme } from "../theme";

const SWIPE_DISTANCE = 0.28; // fraction of the screen width
const SWIPE_VELOCITY = 0.9;

export type DeckHandle = {
  swipe: (action: SwipeAction) => void;
};

type TopCardProps = {
  card: Card;
  onDecide: (card: Card, action: SwipeAction) => void;
  onOpenProfile: (card: Card) => void;
};

const TopCard = forwardRef<DeckHandle, TopCardProps>(function TopCard({ card, onDecide, onOpenProfile }, ref) {
  const palette = useTheme();
  const { width } = useWindowDimensions();
  const position = useRef(new Animated.ValueXY()).current;
  const busy = useRef(false);

  // Gesture callbacks are created once; keep the latest props reachable.
  const latest = useRef({ card, onDecide });
  latest.current = { card, onDecide };

  const flyOut = (action: SwipeAction) => {
    if (busy.current) {
      return;
    }
    busy.current = true;
    Animated.timing(position, {
      toValue: { x: (action === "like" ? 1 : -1) * width * 1.4, y: action === "like" ? -30 : 30 },
      duration: 230,
      useNativeDriver: false
    }).start(() => latest.current.onDecide(latest.current.card, action));
  };
  const flyOutRef = useRef(flyOut);
  flyOutRef.current = flyOut;

  useImperativeHandle(ref, () => ({ swipe: (action) => flyOutRef.current(action) }), []);

  const panResponder = useMemo(
    () =>
      PanResponder.create({
        onMoveShouldSetPanResponder: (_, g) => !busy.current && Math.abs(g.dx) > 10 && Math.abs(g.dx) > Math.abs(g.dy) * 1.2,
        onPanResponderMove: Animated.event([null, { dx: position.x, dy: position.y }], { useNativeDriver: false }),
        onPanResponderRelease: (_, g) => {
          if (g.dx > width * SWIPE_DISTANCE || g.vx > SWIPE_VELOCITY) {
            flyOutRef.current("like");
          } else if (g.dx < -width * SWIPE_DISTANCE || g.vx < -SWIPE_VELOCITY) {
            flyOutRef.current("pass");
          } else {
            Animated.spring(position, { toValue: { x: 0, y: 0 }, friction: 6, useNativeDriver: false }).start();
          }
        },
        onPanResponderTerminate: () => {
          Animated.spring(position, { toValue: { x: 0, y: 0 }, useNativeDriver: false }).start();
        }
      }),
    [position, width]
  );

  const rotate = position.x.interpolate({ inputRange: [-width, 0, width], outputRange: ["-14deg", "0deg", "14deg"] });
  const likeOpacity = position.x.interpolate({ inputRange: [0, width * 0.25], outputRange: [0, 1], extrapolate: "clamp" });
  const passOpacity = position.x.interpolate({ inputRange: [-width * 0.25, 0], outputRange: [1, 0], extrapolate: "clamp" });
  const distance = distanceLabel(card.distanceKm);

  return (
    <Animated.View
      {...panResponder.panHandlers}
      style={{
        ...absolute,
        borderRadius: radii.xl,
        overflow: "hidden",
        backgroundColor: palette.surface,
        transform: [{ translateX: position.x }, { translateY: position.y }, { rotate }]
      }}
      testID="discover-card"
    >
      <PhotoCarousel name={card.firstName} photos={card.photos} />

      <Animated.View
        pointerEvents="none"
        style={[stamp, { left: spacing.xl, borderColor: palette.success, opacity: likeOpacity, transform: [{ rotate: "-14deg" }] }]}
      >
        <Text style={{ color: palette.success }} variant="subheading" weight="bold">
          J&apos;AIME
        </Text>
      </Animated.View>
      <Animated.View
        pointerEvents="none"
        style={[stamp, { right: spacing.xl, borderColor: palette.pass, opacity: passOpacity, transform: [{ rotate: "14deg" }] }]}
      >
        <Text style={{ color: "#FFFFFF" }} variant="subheading" weight="bold">
          PASSER
        </Text>
      </Animated.View>

      <View
        style={{
          position: "absolute",
          left: 0,
          right: 0,
          bottom: 0,
          backgroundColor: "rgba(18, 10, 22, 0.58)",
          padding: spacing.lg,
          gap: spacing.sm
        }}
      >
        <Pressable
          accessibilityHint="Ouvre le profil complet"
          accessibilityLabel={`${card.firstName}, ${card.age} ans`}
          accessibilityRole="button"
          onPress={() => onOpenProfile(card)}
          style={{ flexDirection: "row", alignItems: "flex-end", gap: spacing.md }}
        >
          <View style={{ flex: 1, gap: 2 }}>
            <Text numberOfLines={1} style={{ color: "#FFFFFF" }} variant="heading" weight="bold">
              {card.firstName}, {card.age}
            </Text>
            {distance || card.city ? (
              <Text numberOfLines={1} style={{ color: "rgba(255,255,255,0.88)" }} variant="label">
                {[card.city, distance].filter(Boolean).join(" · ")}
              </Text>
            ) : null}
          </View>
          <Ionicons color="#FFFFFF" name="information-circle-outline" size={28} />
        </Pressable>
        {card.likedMe ? (
          <View
            style={{
              alignSelf: "flex-start",
              backgroundColor: palette.primary,
              borderRadius: radii.pill,
              paddingHorizontal: spacing.md,
              paddingVertical: 3
            }}
          >
            <Text style={{ color: palette.onPrimary }} variant="caption" weight="bold">
              Intéressé(e) par vous
            </Text>
          </View>
        ) : null}
        {card.interests.length > 0 ? (
          <View style={{ flexDirection: "row", flexWrap: "wrap", gap: spacing.xs }}>
            {card.interests.slice(0, 3).map((interest) => (
              <Chip
                key={interest.slug}
                label={interest.label}
                selected={false}
                style={{ backgroundColor: "rgba(255,255,255,0.18)", borderColor: "transparent" }}
              />
            ))}
          </View>
        ) : null}
      </View>
    </Animated.View>
  );
});

type SwipeDeckProps = {
  cards: Card[];
  onDecide: (card: Card, action: SwipeAction) => void;
  onOpenProfile: (card: Card) => void;
  deckRef: React.RefObject<DeckHandle | null>;
};

/** Shows the top card (draggable) over a preview of the next one. */
export function SwipeDeck({ cards, deckRef, onDecide, onOpenProfile }: SwipeDeckProps) {
  const palette = useTheme();
  const [top, next] = cards;
  if (!top) {
    return null;
  }

  return (
    <View style={{ flex: 1 }}>
      {next ? (
        <View
          pointerEvents="none"
          style={{
            ...absolute,
            borderRadius: radii.xl,
            overflow: "hidden",
            backgroundColor: palette.surfaceMuted,
            transform: [{ scale: 0.95 }],
            top: spacing.md
          }}
        >
          <PhotoCarousel interactive={false} name={next.firstName} photos={next.photos.slice(0, 1)} />
        </View>
      ) : null}
      <TopCard card={top} key={top.userId} onDecide={onDecide} onOpenProfile={onOpenProfile} ref={deckRef} />
    </View>
  );
}

const absolute = { position: "absolute", top: 0, left: 0, right: 0, bottom: 0 } as const;

const stamp = {
  position: "absolute",
  top: spacing.xxl,
  borderWidth: 3,
  borderRadius: radii.sm,
  paddingHorizontal: spacing.md,
  paddingVertical: spacing.xs
} as const;
