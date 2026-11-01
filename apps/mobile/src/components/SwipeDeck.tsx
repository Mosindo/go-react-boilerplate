import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Animated, PanResponder, Pressable, StyleSheet, View, useWindowDimensions } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { LinearGradient } from "expo-linear-gradient";
import type { Card } from "../api/types";
import { formatDistance } from "../lib/format";
import { radius, spacing, useColors } from "../theme";
import { PhotoImage, Text } from "./ui";

export type SwipeAction = "like" | "pass";

const SWIPE_THRESHOLD = 110;

function CardFace({ card, photoIndex }: { card: Card; photoIndex: number }) {
  const colors = useColors();
  const distance = formatDistance(card.distanceKm);
  const photoId = card.photoIds[Math.min(photoIndex, card.photoIds.length - 1)];
  return (
    <View style={[styles.card, { backgroundColor: colors.surface }]}>
      {photoId ? <PhotoImage photoId={photoId} style={StyleSheet.absoluteFill} /> : null}
      {card.photoIds.length > 1 ? (
        <View style={styles.dots}>
          {card.photoIds.map((id, index) => (
            <View
              key={id}
              style={[styles.dot, { backgroundColor: index === photoIndex ? "#FFFFFF" : "rgba(255,255,255,0.4)" }]}
            />
          ))}
        </View>
      ) : null}
      <LinearGradient colors={["transparent", "rgba(0,0,0,0.75)"]} style={styles.gradient}>
        <Text style={styles.name} variant="title">
          {card.firstName}, {card.age}
        </Text>
        {distance || card.city ? (
          <Text style={styles.meta}>{[card.city, distance].filter(Boolean).join(" · ")}</Text>
        ) : null}
        {card.interests.length > 0 ? (
          <View style={styles.tags}>
            {card.interests.slice(0, 4).map((interest) => (
              <View key={interest.slug} style={styles.tag}>
                <Text style={styles.tagText} variant="caption">
                  {interest.label}
                </Text>
              </View>
            ))}
          </View>
        ) : null}
      </LinearGradient>
    </View>
  );
}

/**
 * Pile de profils avec swipe (glisser) ET boutons: les deux declenchent la meme
 * action, ce qui garde l'ecran accessible sans geste.
 */
export function SwipeDeck({
  cards,
  onSwipe,
  onOpen,
  disabled,
}: {
  cards: Card[];
  onSwipe: (card: Card, action: SwipeAction) => void;
  onOpen: (card: Card) => void;
  disabled?: boolean;
}) {
  const colors = useColors();
  const { width } = useWindowDimensions();
  const pan = useRef(new Animated.ValueXY()).current;
  const [photoIndex, setPhotoIndex] = useState(0);
  const top = cards[0];
  const next = cards[1];
  const flying = useRef(false);
  const latest = useRef({ top, onSwipe, disabled });
  latest.current = { top, onSwipe, disabled };

  useEffect(() => {
    setPhotoIndex(0);
    flying.current = false;
    pan.setValue({ x: 0, y: 0 });
  }, [top?.userId, pan]);

  const flyOut = useCallback(
    (action: SwipeAction) => {
      const { top: current, onSwipe: handler, disabled: off } = latest.current;
      if (!current || flying.current || off) {
        return;
      }
      flying.current = true;
      Animated.timing(pan, {
        toValue: { x: action === "like" ? width * 1.3 : -width * 1.3, y: 40 },
        duration: 220,
        useNativeDriver: false,
      }).start(() => handler(current, action));
    },
    [pan, width],
  );

  const responder = useMemo(
    () =>
      PanResponder.create({
        onMoveShouldSetPanResponder: (_, g) => Math.abs(g.dx) > 8 && Math.abs(g.dx) > Math.abs(g.dy),
        onPanResponderMove: (_, g) => pan.setValue({ x: g.dx, y: g.dy * 0.3 }),
        onPanResponderRelease: (_, g) => {
          if (g.dx > SWIPE_THRESHOLD) {
            flyOut("like");
          } else if (g.dx < -SWIPE_THRESHOLD) {
            flyOut("pass");
          } else {
            Animated.spring(pan, { toValue: { x: 0, y: 0 }, useNativeDriver: false }).start();
          }
        },
        onPanResponderTerminate: () =>
          Animated.spring(pan, { toValue: { x: 0, y: 0 }, useNativeDriver: false }).start(),
      }),
    [pan, flyOut],
  );

  const rotate = pan.x.interpolate({ inputRange: [-width, 0, width], outputRange: ["-12deg", "0deg", "12deg"] });
  const likeOpacity = pan.x.interpolate({
    inputRange: [0, SWIPE_THRESHOLD],
    outputRange: [0, 1],
    extrapolate: "clamp",
  });
  const passOpacity = pan.x.interpolate({
    inputRange: [-SWIPE_THRESHOLD, 0],
    outputRange: [1, 0],
    extrapolate: "clamp",
  });
  const nextScale = pan.x.interpolate({
    inputRange: [-width, 0, width],
    outputRange: [1, 0.94, 1],
    extrapolate: "clamp",
  });

  if (!top) {
    return null;
  }

  const onCardPress = (locationX: number, cardWidth: number) => {
    if (top.photoIds.length > 1 && (locationX < cardWidth * 0.3 || locationX > cardWidth * 0.7)) {
      setPhotoIndex((i) => {
        const delta = locationX < cardWidth * 0.3 ? -1 : 1;
        return Math.max(0, Math.min(top.photoIds.length - 1, i + delta));
      });
    } else {
      onOpen(top);
    }
  };

  return (
    <View style={styles.container}>
      <View style={styles.stack}>
        {next ? (
          <Animated.View style={[StyleSheet.absoluteFill, { transform: [{ scale: nextScale }] }]}>
            <CardFace card={next} photoIndex={0} />
          </Animated.View>
        ) : null}
        <Animated.View
          {...responder.panHandlers}
          style={[StyleSheet.absoluteFill, { transform: [{ translateX: pan.x }, { translateY: pan.y }, { rotate }] }]}
        >
          <Pressable
            accessibilityHint="Touchez les bords pour changer de photo, le centre pour ouvrir le profil"
            accessibilityLabel={`${top.firstName}, ${top.age} ans`}
            onPress={(e) => onCardPress(e.nativeEvent.locationX, width - spacing.lg * 2)}
            style={StyleSheet.absoluteFill}
          >
            <CardFace card={top} photoIndex={photoIndex} />
          </Pressable>
          <Animated.View
            pointerEvents="none"
            style={[styles.stamp, styles.stampLike, { opacity: likeOpacity, borderColor: colors.like }]}
          >
            <Text style={{ color: colors.like }} variant="heading">
              J&apos;AIME
            </Text>
          </Animated.View>
          <Animated.View
            pointerEvents="none"
            style={[styles.stamp, styles.stampPass, { opacity: passOpacity, borderColor: colors.pass }]}
          >
            <Text style={{ color: colors.pass }} variant="heading">
              PASSER
            </Text>
          </Animated.View>
        </Animated.View>
      </View>

      <View style={styles.actions}>
        <Pressable
          accessibilityLabel="Passer"
          accessibilityRole="button"
          disabled={disabled}
          onPress={() => flyOut("pass")}
          style={[styles.action, { backgroundColor: colors.surface, borderColor: colors.border }]}
          testID="swipe-pass"
        >
          <Ionicons color={colors.pass} name="close" size={32} />
        </Pressable>
        <Pressable
          accessibilityLabel="Voir le profil complet"
          accessibilityRole="button"
          onPress={() => onOpen(top)}
          style={[styles.actionSmall, { backgroundColor: colors.surface, borderColor: colors.border }]}
        >
          <Ionicons color={colors.textMuted} name="information" size={22} />
        </Pressable>
        <Pressable
          accessibilityLabel="J'aime"
          accessibilityRole="button"
          disabled={disabled}
          onPress={() => flyOut("like")}
          style={[styles.action, { backgroundColor: colors.primary, borderColor: colors.primary }]}
          testID="swipe-like"
        >
          <Ionicons color={colors.onPrimary} name="heart" size={30} />
        </Pressable>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  stack: { flex: 1, margin: spacing.lg, marginBottom: spacing.md },
  card: { flex: 1, borderRadius: radius.lg, overflow: "hidden" },
  gradient: { position: "absolute", left: 0, right: 0, bottom: 0, padding: spacing.lg, paddingTop: 80 },
  name: { color: "#FFFFFF" },
  meta: { color: "rgba(255,255,255,0.9)", marginTop: 2 },
  tags: { flexDirection: "row", flexWrap: "wrap", marginTop: spacing.sm },
  tag: {
    backgroundColor: "rgba(255,255,255,0.22)",
    borderRadius: radius.pill,
    paddingHorizontal: 10,
    paddingVertical: 4,
    marginRight: 6,
    marginBottom: 6,
  },
  tagText: { color: "#FFFFFF" },
  dots: { position: "absolute", top: 10, left: 12, right: 12, flexDirection: "row", gap: 4 },
  dot: { flex: 1, height: 3, borderRadius: 2 },
  stamp: {
    position: "absolute",
    top: 40,
    borderWidth: 3,
    borderRadius: radius.sm,
    paddingHorizontal: spacing.md,
    paddingVertical: 2,
    backgroundColor: "rgba(255,255,255,0.85)",
  },
  stampLike: { left: 24, transform: [{ rotate: "-12deg" }] },
  stampPass: { right: 24, transform: [{ rotate: "12deg" }] },
  actions: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "center",
    gap: spacing.xl,
    paddingBottom: spacing.lg,
  },
  action: { width: 68, height: 68, borderRadius: 34, borderWidth: 1, alignItems: "center", justifyContent: "center" },
  actionSmall: {
    width: 44,
    height: 44,
    borderRadius: 22,
    borderWidth: 1,
    alignItems: "center",
    justifyContent: "center",
  },
});
