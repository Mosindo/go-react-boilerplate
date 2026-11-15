import React, { forwardRef, useImperativeHandle, useMemo, useRef, useState } from "react";
import { Animated, PanResponder, Pressable, StyleSheet, View, useWindowDimensions } from "react-native";
import { LinearGradient } from "expo-linear-gradient";
import type { ProfileCard, SwipeAction } from "../../api/types";
import { radii, spacing, useTheme } from "../../theme";
import { Icon, PhotoImage, Text } from "../../ui";
import { Fact, facts, factStyles } from "./ProfileFacts";

export type SwipeCardHandle = { swipe: (action: SwipeAction) => void };

type Props = {
  card: ProfileCard;
  sharedCount: number;
  onSwiped: (action: SwipeAction) => void;
  onOpen: () => void;
  active: boolean;
};

const SWIPE_THRESHOLD = 110;

/**
 * Swipeable card: drag right to like, left to pass, or use the buttons.
 * Tapping the left/right half of the photo browses photos.
 */
export const SwipeCard = forwardRef<SwipeCardHandle, Props>(function SwipeCard({ card, sharedCount, onSwiped, onOpen, active }, ref) {
  const { colors } = useTheme();
  const { width } = useWindowDimensions();
  const position = useRef(new Animated.ValueXY()).current;
  const [photoIndex, setPhotoIndex] = useState(0);
  const leaving = useRef(false);

  const fling = (action: SwipeAction, velocity = 0) => {
    if (leaving.current) return;
    leaving.current = true;
    const toX = (action === "like" ? 1 : -1) * (width + 200);
    Animated.timing(position, {
      toValue: { x: toX, y: 40 },
      duration: Math.max(160, 280 - Math.abs(velocity) * 60),
      useNativeDriver: true
    }).start(() => onSwiped(action));
  };

  useImperativeHandle(ref, () => ({ swipe: (action: SwipeAction) => fling(action) }));

  const responder = useMemo(
    () =>
      PanResponder.create({
        onMoveShouldSetPanResponder: (_, g) => active && Math.abs(g.dx) > 8 && Math.abs(g.dx) > Math.abs(g.dy),
        onPanResponderMove: Animated.event([null, { dx: position.x, dy: position.y }], { useNativeDriver: false }),
        onPanResponderRelease: (_, g) => {
          if (g.dx > SWIPE_THRESHOLD || g.vx > 1.2) fling("like", g.vx);
          else if (g.dx < -SWIPE_THRESHOLD || g.vx < -1.2) fling("pass", g.vx);
          else Animated.spring(position, { toValue: { x: 0, y: 0 }, friction: 6, useNativeDriver: true }).start();
        },
        onPanResponderTerminate: () => Animated.spring(position, { toValue: { x: 0, y: 0 }, useNativeDriver: true }).start()
      }),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [active, width]
  );

  const rotate = position.x.interpolate({ inputRange: [-width, 0, width], outputRange: ["-12deg", "0deg", "12deg"] });
  const likeOpacity = position.x.interpolate({ inputRange: [20, SWIPE_THRESHOLD], outputRange: [0, 1], extrapolate: "clamp" });
  const passOpacity = position.x.interpolate({ inputRange: [-SWIPE_THRESHOLD, -20], outputRange: [1, 0], extrapolate: "clamp" });

  const photo = card.photos[photoIndex];
  const browse = (dir: 1 | -1) => setPhotoIndex((i) => Math.min(Math.max(i + dir, 0), card.photos.length - 1));

  return (
    <Animated.View
      {...responder.panHandlers}
      style={[
        styles.card,
        { backgroundColor: colors.surface, borderColor: colors.border },
        { transform: [{ translateX: position.x }, { translateY: position.y }, { rotate }] }
      ]}
      accessibilityLabel={`${card.firstName}, ${card.age} ans`}
      testID="swipe-card"
    >
      <PhotoImage photo={photo} style={StyleSheet.absoluteFill} priority="high" accessibilityLabel={`Photo de ${card.firstName}`} />
      <View style={styles.tapZones}>
        <Pressable style={styles.flex} onPress={() => browse(-1)} accessibilityLabel="Photo précédente" />
        <Pressable style={styles.flex} onPress={() => browse(1)} accessibilityLabel="Photo suivante" />
      </View>
      {card.photos.length > 1 ? (
        <View style={styles.dots} pointerEvents="none">
          {card.photos.map((p, i) => (
            <View key={p.id} style={[styles.dot, { backgroundColor: i === photoIndex ? "#fff" : "rgba(255,255,255,0.35)" }]} />
          ))}
        </View>
      ) : null}
      <Animated.View style={[styles.stamp, styles.stampLike, { opacity: likeOpacity, borderColor: colors.success }]} pointerEvents="none">
        <Text variant="heading" style={{ color: colors.success }}>
          J’AIME
        </Text>
      </Animated.View>
      <Animated.View style={[styles.stamp, styles.stampPass, { opacity: passOpacity, borderColor: colors.danger }]} pointerEvents="none">
        <Text variant="heading" style={{ color: colors.danger }}>
          PASSER
        </Text>
      </Animated.View>
      {card.isDemo ? (
        <View style={styles.demo} pointerEvents="none">
          <Text variant="micro" style={styles.white}>
            PROFIL DÉMO
          </Text>
        </View>
      ) : null}
      <LinearGradient colors={["transparent", "rgba(12,8,6,0.92)"]} style={styles.gradient} pointerEvents="none" />
      <Pressable
        style={styles.info}
        onPress={onOpen}
        accessibilityRole="button"
        accessibilityLabel={`Voir le profil de ${card.firstName}`}
        testID="swipe-card-open"
      >
        <View style={styles.nameRow}>
          <Text variant="title" style={styles.white} numberOfLines={1}>
            {card.firstName}
          </Text>
          <Text variant="title" style={[styles.white, styles.age]}>
            {card.age}
          </Text>
          <View style={styles.flex} />
          <Icon name="information-circle-outline" size={26} rawColor="#F7EFE9" />
        </View>
        <View style={factStyles.row}>
          {facts(card).map((f) => (
            <Fact key={f.label} icon={f.icon} label={f.label} light />
          ))}
        </View>
        {sharedCount > 0 ? (
          <View style={[styles.shared, { backgroundColor: colors.primary }]}>
            <Icon name="sparkles" size={13} color="onPrimary" />
            <Text variant="micro" color="onPrimary">
              {sharedCount} INTÉRÊT{sharedCount > 1 ? "S" : ""} EN COMMUN
            </Text>
          </View>
        ) : null}
        {card.bio ? (
          <Text variant="caption" style={styles.bio} numberOfLines={2}>
            {card.bio}
          </Text>
        ) : null}
      </Pressable>
    </Animated.View>
  );
});

const styles = StyleSheet.create({
  card: {
    ...StyleSheet.absoluteFillObject,
    borderRadius: radii.xl,
    overflow: "hidden",
    borderWidth: 1,
    shadowColor: "#000",
    shadowOpacity: 0.3,
    shadowRadius: 18,
    shadowOffset: { width: 0, height: 10 },
    elevation: 6
  },
  flex: { flex: 1 },
  tapZones: { ...StyleSheet.absoluteFillObject, flexDirection: "row", bottom: "35%" },
  dots: { position: "absolute", top: spacing.sm, left: spacing.sm, right: spacing.sm, flexDirection: "row", gap: 4 },
  dot: { flex: 1, height: 3.5, borderRadius: 2 },
  stamp: { position: "absolute", top: 48, borderWidth: 3, borderRadius: radii.sm, paddingHorizontal: spacing.sm, paddingVertical: 4 },
  stampLike: { left: 24, transform: [{ rotate: "-14deg" }] },
  stampPass: { right: 24, transform: [{ rotate: "14deg" }] },
  demo: {
    position: "absolute",
    top: spacing.lg,
    right: spacing.sm,
    backgroundColor: "rgba(0,0,0,0.55)",
    borderRadius: radii.pill,
    paddingHorizontal: 8,
    paddingVertical: 4
  },
  gradient: { position: "absolute", left: 0, right: 0, bottom: 0, height: "55%" },
  info: { position: "absolute", left: 0, right: 0, bottom: 0, padding: spacing.lg, gap: spacing.xs },
  nameRow: { flexDirection: "row", alignItems: "baseline", gap: spacing.xs },
  age: { fontWeight: "400" },
  white: { color: "#F7EFE9" },
  shared: {
    alignSelf: "flex-start",
    flexDirection: "row",
    alignItems: "center",
    gap: 4,
    borderRadius: radii.pill,
    paddingHorizontal: 8,
    paddingVertical: 4
  },
  bio: { color: "rgba(247,239,233,0.85)" }
});
