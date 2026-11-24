import React, { forwardRef, useImperativeHandle, useMemo, useRef, useState } from "react";
import { Animated, PanResponder, Pressable, StyleSheet, useWindowDimensions, View, type GestureResponderEvent } from "react-native";
import { Image } from "expo-image";
import Ionicons from "@expo/vector-icons/Ionicons";
import type { ProfileCard, SwipeDecision } from "../../api/types";
import { absoluteUrl } from "../../config";
import { colors, radii, shadow, spacing } from "../../theme/tokens";
import { distanceLabel } from "../../utils/format";
import { AppText } from "../AppText";
import { Chip } from "../Chip";

export type SwipeCardHandle = { swipe: (decision: SwipeDecision) => void };

type Props = {
  card: ProfileCard;
  active: boolean;
  onSwiped: (decision: SwipeDecision) => void;
  onOpen: () => void;
};

export const SwipeCard = forwardRef<SwipeCardHandle, Props>(function SwipeCard({ card, active, onSwiped, onOpen }, ref) {
  const { width } = useWindowDimensions();
  const threshold = Math.min(width * 0.28, 140);
  const position = useRef(new Animated.ValueXY()).current;
  const [photoIndex, setPhotoIndex] = useState(0);
  const leaving = useRef(false);

  const fling = (decision: SwipeDecision) => {
    if (leaving.current) return;
    leaving.current = true;
    Animated.timing(position, {
      toValue: { x: (decision === "like" ? 1 : -1) * width * 1.4, y: 30 },
      duration: 230,
      useNativeDriver: false
    }).start(() => onSwiped(decision));
  };

  useImperativeHandle(ref, () => ({ swipe: fling }));

  const responder = useMemo(
    () =>
      PanResponder.create({
        onMoveShouldSetPanResponder: (_, g) => active && Math.abs(g.dx) > 8 && Math.abs(g.dx) > Math.abs(g.dy),
        onPanResponderMove: Animated.event([null, { dx: position.x, dy: position.y }], { useNativeDriver: false }),
        onPanResponderRelease: (_, g) => {
          if (g.dx > threshold || g.vx > 1.2) fling("like");
          else if (g.dx < -threshold || g.vx < -1.2) fling("pass");
          else Animated.spring(position, { toValue: { x: 0, y: 0 }, friction: 6, useNativeDriver: false }).start();
        },
        onPanResponderTerminate: () => Animated.spring(position, { toValue: { x: 0, y: 0 }, useNativeDriver: false }).start()
      }),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [active, threshold]
  );

  const rotate = position.x.interpolate({ inputRange: [-width, 0, width], outputRange: ["-10deg", "0deg", "10deg"] });
  const likeOpacity = position.x.interpolate({ inputRange: [0, threshold], outputRange: [0, 1], extrapolate: "clamp" });
  const passOpacity = position.x.interpolate({ inputRange: [-threshold, 0], outputRange: [1, 0], extrapolate: "clamp" });

  const photos = card.photos;
  const photo = photos[Math.min(photoIndex, photos.length - 1)];
  const onPhotoTap = (event: GestureResponderEvent) => {
    const x = event.nativeEvent.locationX;
    if (x < width * 0.35) setPhotoIndex((i) => Math.max(0, i - 1));
    else setPhotoIndex((i) => Math.min(photos.length - 1, i + 1));
  };

  const distance = distanceLabel(card.distanceKm);
  const subtitle = [card.jobTitle, distance].filter(Boolean).join(" · ");
  const highlights = [...card.interests].sort((a, b) => Number(b.shared) - Number(a.shared)).slice(0, 3);

  return (
    <Animated.View
      {...(active ? responder.panHandlers : {})}
      style={[styles.card, active ? { transform: [...position.getTranslateTransform(), { rotate }] } : styles.behind]}
      testID={active ? "discover-card" : undefined}
      accessibilityLabel={`${card.firstName}, ${card.age} ans${distance ? `, ${distance}` : ""}`}
    >
      {photo ? <Image source={{ uri: absoluteUrl(photo.url) }} style={StyleSheet.absoluteFill} contentFit="cover" transition={200} cachePolicy="memory-disk" /> : null}
      <Pressable style={StyleSheet.absoluteFill} onPress={onPhotoTap} accessibilityLabel="Photo suivante" disabled={!active || photos.length < 2} />

      {photos.length > 1 ? (
        <View style={styles.bars} pointerEvents="none">
          {photos.map((p, i) => (
            <View key={p.id} style={[styles.bar, i === photoIndex && styles.barActive]} />
          ))}
        </View>
      ) : null}

      <Animated.View style={[styles.stamp, styles.likeStamp, { opacity: likeOpacity }]} pointerEvents="none">
        <AppText variant="heading" color={colors.success}>
          J’AIME
        </AppText>
      </Animated.View>
      <Animated.View style={[styles.stamp, styles.passStamp, { opacity: passOpacity }]} pointerEvents="none">
        <AppText variant="heading" color={colors.textMuted}>
          PASSER
        </AppText>
      </Animated.View>

      <View style={styles.scrim} pointerEvents="box-none">
        <View style={[styles.fade, { opacity: 0.25 }]} pointerEvents="none" />
        <View style={[styles.fade, { opacity: 0.5 }]} pointerEvents="none" />
        <View style={styles.info} pointerEvents="box-none">
          <View style={styles.titleRow} pointerEvents="box-none">
            <View style={styles.flex} pointerEvents="none">
              <AppText variant="title" color={colors.white}>
                {card.firstName}, {card.age}
              </AppText>
              {subtitle ? (
                <AppText color={colors.text} numberOfLines={1}>
                  {subtitle}
                </AppText>
              ) : null}
            </View>
            <Pressable accessibilityRole="button" accessibilityLabel={`Voir le profil de ${card.firstName}`} onPress={onOpen} style={styles.infoButton} testID="discover-open-profile">
              <Ionicons name="chevron-up" size={22} color={colors.text} />
            </Pressable>
          </View>
          {highlights.length > 0 ? (
            <View style={styles.chips} pointerEvents="none">
              {highlights.map((i) => (
                <Chip key={i.id} label={i.label} highlighted={i.shared} />
              ))}
            </View>
          ) : null}
        </View>
      </View>
    </Animated.View>
  );
});

const styles = StyleSheet.create({
  card: {
    ...StyleSheet.absoluteFillObject,
    borderRadius: radii.xl,
    overflow: "hidden",
    backgroundColor: colors.surfaceRaised,
    borderWidth: 1,
    borderColor: colors.border,
    ...shadow.card
  },
  behind: { transform: [{ scale: 0.95 }, { translateY: 14 }], opacity: 0.75 },
  bars: { position: "absolute", top: spacing.sm, left: spacing.sm, right: spacing.sm, flexDirection: "row", gap: 4 },
  bar: { flex: 1, height: 3, borderRadius: 2, backgroundColor: "rgba(255,255,255,0.35)" },
  barActive: { backgroundColor: colors.white },
  stamp: { position: "absolute", top: spacing.xl, paddingHorizontal: spacing.md, paddingVertical: spacing.xs, borderRadius: radii.sm, borderWidth: 3 },
  likeStamp: { left: spacing.lg, borderColor: colors.success, transform: [{ rotate: "-12deg" }] },
  passStamp: { right: spacing.lg, borderColor: colors.textMuted, transform: [{ rotate: "12deg" }] },
  scrim: { position: "absolute", left: 0, right: 0, bottom: 0 },
  fade: { ...StyleSheet.absoluteFillObject, backgroundColor: "#000" },
  info: { padding: spacing.lg, gap: spacing.sm },
  titleRow: { flexDirection: "row", alignItems: "flex-end", gap: spacing.sm },
  flex: { flex: 1 },
  infoButton: {
    width: 40,
    height: 40,
    borderRadius: 20,
    alignItems: "center",
    justifyContent: "center",
    backgroundColor: "rgba(255,255,255,0.18)"
  },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.xs }
});
