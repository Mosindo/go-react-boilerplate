import React, { forwardRef, useImperativeHandle, useMemo, useRef, useState } from "react";
import {
  Animated,
  PanResponder,
  StyleSheet,
  useWindowDimensions,
  View,
  type LayoutChangeEvent
} from "react-native";
import { Ionicons } from "@expo/vector-icons";
import type { Candidate, SwipeAction } from "../api/types";
import { radii, useTheme } from "../theme";
import { PhotoImage, Text } from "../ui";

export type DeckCardHandle = { swipe: (action: SwipeAction) => void };

type CardProps = {
  card: Candidate;
  onSwiped: (action: SwipeAction) => void;
  onOpen: () => void;
};

const THRESHOLD = 110;

/** Draggable top card. Photo browsing: tap left/right edge; tap centre opens the profile. */
export const TopCard = forwardRef<DeckCardHandle, CardProps>(function TopCard(
  { card, onSwiped, onOpen },
  ref
) {
  const { width } = useWindowDimensions();
  const position = useRef(new Animated.ValueXY()).current;
  const [photoIndex, setPhotoIndex] = useState(0);
  const [cardWidth, setCardWidth] = useState(width);
  const busy = useRef(false);
  const propsRef = useRef({ onSwiped, onOpen, photos: card.photos.length, cardWidth });
  propsRef.current = { onSwiped, onOpen, photos: card.photos.length, cardWidth };

  const fling = (action: SwipeAction) => {
    if (busy.current) return;
    busy.current = true;
    Animated.timing(position, {
      toValue: { x: action === "like" ? width * 1.4 : -width * 1.4, y: 40 },
      duration: 220,
      useNativeDriver: false
    }).start(() => propsRef.current.onSwiped(action));
  };
  useImperativeHandle(ref, () => ({ swipe: fling }));

  const pan = useMemo(
    () =>
      PanResponder.create({
        onMoveShouldSetPanResponder: (_, g) => Math.abs(g.dx) > 6 || Math.abs(g.dy) > 6,
        onPanResponderMove: Animated.event([null, { dx: position.x, dy: position.y }], {
          useNativeDriver: false
        }),
        onPanResponderRelease: (e, g) => {
          const { onOpen: open, photos, cardWidth: w } = propsRef.current;
          if (Math.abs(g.dx) < 6 && Math.abs(g.dy) < 6) {
            // A tap.
            const x = e.nativeEvent.locationX;
            if (x < w * 0.3) setPhotoIndex((i) => Math.max(0, i - 1));
            else if (x > w * 0.7) setPhotoIndex((i) => Math.min(Math.max(photos - 1, 0), i + 1));
            else open();
            return;
          }
          if (g.dx > THRESHOLD) fling("like");
          else if (g.dx < -THRESHOLD) fling("pass");
          else
            Animated.spring(position, { toValue: { x: 0, y: 0 }, useNativeDriver: false }).start();
        }
      }),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    []
  );

  const rotate = position.x.interpolate({
    inputRange: [-width, 0, width],
    outputRange: ["-12deg", "0deg", "12deg"]
  });
  const likeOpacity = position.x.interpolate({
    inputRange: [20, THRESHOLD],
    outputRange: [0, 1],
    extrapolate: "clamp"
  });
  const passOpacity = position.x.interpolate({
    inputRange: [-THRESHOLD, -20],
    outputRange: [1, 0],
    extrapolate: "clamp"
  });
  const photo = card.photos[photoIndex] ?? card.photos[0];

  return (
    <Animated.View
      {...pan.panHandlers}
      onLayout={(e: LayoutChangeEvent) => setCardWidth(e.nativeEvent.layout.width)}
      style={[
        styles.card,
        { transform: [{ translateX: position.x }, { translateY: position.y }, { rotate }] }
      ]}
    >
      <CardFace card={card} photoId={photo?.id} photoIndex={photoIndex} />
      <Animated.View
        pointerEvents="none"
        style={[styles.stamp, styles.stampLike, { opacity: likeOpacity }]}
      >
        <Text variant="title" style={styles.stampLikeText}>
          J'AIME
        </Text>
      </Animated.View>
      <Animated.View
        pointerEvents="none"
        style={[styles.stamp, styles.stampPass, { opacity: passOpacity }]}
      >
        <Text variant="title" style={styles.stampPassText}>
          PASSER
        </Text>
      </Animated.View>
    </Animated.View>
  );
});

export function CardFace({
  card,
  photoId,
  photoIndex
}: {
  card: Candidate;
  photoId: string | undefined;
  photoIndex: number;
}) {
  const t = useTheme();
  return (
    <View style={[StyleSheet.absoluteFill, { backgroundColor: t.surfaceAlt }]}>
      <PhotoImage
        photoId={photoId}
        style={StyleSheet.absoluteFill}
        accessibilityLabel={`Photo de ${card.firstName}`}
      />
      {card.photos.length > 1 ? (
        <View style={styles.bars} pointerEvents="none">
          {card.photos.map((p, i) => (
            <View key={p.id} style={[styles.bar, { opacity: i === photoIndex ? 1 : 0.4 }]} />
          ))}
        </View>
      ) : null}
      <View style={styles.caption} pointerEvents="none">
        <Text variant="title" style={styles.white}>
          {card.firstName}, {card.age}
        </Text>
        <View style={styles.meta}>
          {card.city ? (
            <View style={styles.metaItem}>
              <Ionicons name="location-outline" size={15} color="#fff" />
              <Text variant="caption" style={styles.white}>
                {card.city}
                {card.distanceKm !== null ? ` · à ~${card.distanceKm} km` : ""}
              </Text>
            </View>
          ) : card.distanceKm !== null ? (
            <Text variant="caption" style={styles.white}>
              À ~{card.distanceKm} km
            </Text>
          ) : null}
        </View>
        {card.bio ? (
          <Text variant="caption" style={styles.white} numberOfLines={2}>
            {card.bio}
          </Text>
        ) : null}
      </View>
    </View>
  );
}

export function BackCard({ card }: { card: Candidate }) {
  return (
    <View style={[styles.card, styles.back]} pointerEvents="none">
      <CardFace card={card} photoId={card.photos[0]?.id} photoIndex={0} />
    </View>
  );
}

const styles = StyleSheet.create({
  card: { ...StyleSheet.absoluteFillObject, borderRadius: radii.xl, overflow: "hidden" },
  back: { transform: [{ scale: 0.95 }], opacity: 0.9 },
  bars: { position: "absolute", top: 10, left: 10, right: 10, flexDirection: "row", gap: 4 },
  bar: { flex: 1, height: 3, borderRadius: 2, backgroundColor: "#fff" },
  caption: {
    position: "absolute",
    left: 0,
    right: 0,
    bottom: 0,
    padding: 18,
    paddingTop: 48,
    gap: 4,
    backgroundColor: "rgba(0,0,0,0.38)"
  },
  white: { color: "#fff" },
  meta: { flexDirection: "row", gap: 12 },
  metaItem: { flexDirection: "row", alignItems: "center", gap: 4 },
  stamp: {
    position: "absolute",
    top: 36,
    borderWidth: 3,
    borderRadius: radii.md,
    paddingHorizontal: 12,
    paddingVertical: 4
  },
  stampLike: { left: 24, borderColor: "#5FD39B", transform: [{ rotate: "-12deg" }] },
  stampPass: { right: 24, borderColor: "#F2A19A", transform: [{ rotate: "12deg" }] },
  stampLikeText: { color: "#5FD39B" },
  stampPassText: { color: "#F2A19A" }
});
