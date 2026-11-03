import React, { useMemo, useRef, useState } from "react";
import { Animated, PanResponder, Pressable, StyleSheet, Text, View } from "react-native";
import type { Card, SwipeAction } from "../../api/types";
import { distanceLabel } from "../../lib/format";
import { radius, spacing, useTheme } from "../../theme";
import { AuthImage } from "../../ui/AuthImage";
import { Chip, ChipRow } from "../../ui/components";

const THRESHOLD = 110;

type Props = {
  card: Card;
  /** Only the top card is draggable. */
  interactive: boolean;
  onSwipe: (action: SwipeAction) => void;
  onOpen: () => void;
  width: number;
  height: number;
};

export function SwipeCard({ card, interactive, onSwipe, onOpen, width, height }: Props) {
  const theme = useTheme();
  const position = useRef(new Animated.ValueXY()).current;
  const [photoIndex, setPhotoIndex] = useState(0);
  const callbacks = useRef({ onSwipe });
  callbacks.current = { onSwipe };

  const responder = useMemo(
    () =>
      PanResponder.create({
        onMoveShouldSetPanResponder: (_e, g) => Math.abs(g.dx) > 8,
        onPanResponderMove: Animated.event([null, { dx: position.x, dy: position.y }], { useNativeDriver: false }),
        onPanResponderRelease: (_e, g) => {
          if (Math.abs(g.dx) > THRESHOLD) {
            const action: SwipeAction = g.dx > 0 ? "like" : "pass";
            Animated.timing(position, {
              toValue: { x: Math.sign(g.dx) * width * 1.5, y: g.dy },
              duration: 180,
              useNativeDriver: false
            }).start(() => callbacks.current.onSwipe(action));
          } else {
            Animated.spring(position, { toValue: { x: 0, y: 0 }, useNativeDriver: false }).start();
          }
        }
      }),
    [position, width]
  );

  const rotate = position.x.interpolate({ inputRange: [-width, 0, width], outputRange: ["-12deg", "0deg", "12deg"] });
  const likeOpacity = position.x.interpolate({ inputRange: [0, THRESHOLD], outputRange: [0, 1], extrapolate: "clamp" });
  const passOpacity = position.x.interpolate({ inputRange: [-THRESHOLD, 0], outputRange: [1, 0], extrapolate: "clamp" });

  const photo = card.photos[Math.min(photoIndex, card.photos.length - 1)];
  const distance = distanceLabel(card.distanceKm);

  return (
    <Animated.View
      {...(interactive ? responder.panHandlers : {})}
      testID={interactive ? "swipe-card" : undefined}
      style={[
        styles.card,
        { width, height, backgroundColor: theme.surface },
        { transform: [{ translateX: position.x }, { translateY: position.y }, { rotate }] }
      ]}
    >
      <AuthImage path={photo?.url} label={`Photo de ${card.firstName}`} style={StyleSheet.absoluteFill} />
      <View style={styles.tapZones}>
        <Pressable accessibilityLabel="Photo précédente" style={styles.flex} onPress={() => setPhotoIndex((i) => Math.max(0, i - 1))} />
        <Pressable
          accessibilityLabel="Photo suivante"
          style={styles.flex}
          onPress={() => setPhotoIndex((i) => Math.min(card.photos.length - 1, i + 1))}
        />
      </View>
      {card.photos.length > 1 ? (
        <View style={styles.dots}>
          {card.photos.map((p, i) => (
            <View key={p.id} style={[styles.dot, { opacity: i === photoIndex ? 1 : 0.4 }]} />
          ))}
        </View>
      ) : null}
      <Animated.View pointerEvents="none" style={[styles.stamp, styles.likeStamp, { opacity: likeOpacity, borderColor: theme.success }]}>
        <Text style={[styles.stampText, { color: theme.success }]}>J&apos;AIME</Text>
      </Animated.View>
      <Animated.View pointerEvents="none" style={[styles.stamp, styles.passStamp, { opacity: passOpacity, borderColor: theme.danger }]}>
        <Text style={[styles.stampText, { color: theme.danger }]}>PASSER</Text>
      </Animated.View>
      <Pressable accessibilityRole="button" accessibilityLabel={`Voir le profil de ${card.firstName}`} onPress={onOpen} style={styles.info}>
        <Text style={styles.name} numberOfLines={1}>{`${card.firstName}, ${card.age}`}</Text>
        {card.city || distance ? <Text style={styles.meta}>{[card.city, distance].filter(Boolean).join(" · ")}</Text> : null}
        {card.bio ? (
          <Text style={styles.bio} numberOfLines={2}>
            {card.bio}
          </Text>
        ) : null}
        {card.interests.length > 0 ? (
          <ChipRow>
            {card.interests.slice(0, 3).map((interest) => (
              <Chip key={interest.slug} label={interest.label} />
            ))}
          </ChipRow>
        ) : null}
      </Pressable>
    </Animated.View>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  card: {
    position: "absolute",
    borderRadius: radius.lg,
    overflow: "hidden",
    elevation: 4,
    shadowColor: "#000",
    shadowOpacity: 0.18,
    shadowRadius: 12,
    shadowOffset: { width: 0, height: 6 }
  },
  tapZones: { ...StyleSheet.absoluteFillObject, flexDirection: "row", bottom: 140 },
  dots: {
    position: "absolute",
    top: spacing.md,
    left: spacing.lg,
    right: spacing.lg,
    flexDirection: "row",
    gap: 4,
    justifyContent: "center"
  },
  dot: { flex: 1, height: 4, borderRadius: 2, backgroundColor: "#FFFFFF" },
  info: {
    position: "absolute",
    left: 0,
    right: 0,
    bottom: 0,
    padding: spacing.lg,
    gap: spacing.xs,
    backgroundColor: "rgba(20,10,14,0.62)"
  },
  name: { color: "#FFFFFF", fontSize: 26, fontWeight: "800" },
  meta: { color: "#F3E6E0", fontSize: 14 },
  bio: { color: "#FFFFFF", fontSize: 15 },
  stamp: {
    position: "absolute",
    top: 48,
    borderWidth: 3,
    borderRadius: radius.sm,
    paddingHorizontal: spacing.md,
    paddingVertical: 2,
    backgroundColor: "rgba(255,255,255,0.85)"
  },
  likeStamp: { left: spacing.xl, transform: [{ rotate: "-12deg" }] },
  passStamp: { right: spacing.xl, transform: [{ rotate: "12deg" }] },
  stampText: { fontSize: 24, fontWeight: "900" }
});
