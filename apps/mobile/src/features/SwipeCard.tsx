import React, { forwardRef, useImperativeHandle, useMemo, useRef, useState } from "react";
import { Animated, PanResponder, Pressable, StyleSheet, View, useWindowDimensions } from "react-native";
import type { PublicProfile, SwipeAction } from "../api/types";
import { radii, useTheme } from "../theme/theme";
import { Chip, PhotoImage, Text } from "../ui";

export type SwipeCardHandle = { swipe: (action: SwipeAction) => void };

type Props = {
  profile: PublicProfile;
  onSwiped: (action: SwipeAction) => void;
  onOpenProfile: () => void;
  interactive: boolean;
};

const THRESHOLD = 110;

export const SwipeCard = forwardRef<SwipeCardHandle, Props>(function SwipeCard({ profile, onSwiped, onOpenProfile, interactive }, ref) {
  const { colors } = useTheme();
  const { width } = useWindowDimensions();
  const position = useRef(new Animated.ValueXY()).current;
  const [photoIndex, setPhotoIndex] = useState(0);
  const done = useRef(false);
  const callbacks = useRef({ onSwiped });
  callbacks.current = { onSwiped };

  const fling = (action: SwipeAction) => {
    if (done.current) return;
    done.current = true;
    Animated.timing(position, {
      toValue: { x: action === "like" ? width * 1.4 : -width * 1.4, y: 40 },
      duration: 220,
      useNativeDriver: false
    }).start(() => callbacks.current.onSwiped(action));
  };

  useImperativeHandle(ref, () => ({ swipe: fling }));

  const pan = useMemo(
    () =>
      PanResponder.create({
        onMoveShouldSetPanResponder: (_, g) => interactive && Math.abs(g.dx) > 8 && Math.abs(g.dx) > Math.abs(g.dy),
        onPanResponderMove: Animated.event([null, { dx: position.x, dy: position.y }], { useNativeDriver: false }),
        onPanResponderRelease: (_, g) => {
          if (g.dx > THRESHOLD) fling("like");
          else if (g.dx < -THRESHOLD) fling("pass");
          else Animated.spring(position, { toValue: { x: 0, y: 0 }, useNativeDriver: false, friction: 6 }).start();
        }
      }),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [interactive, width]
  );

  const rotate = position.x.interpolate({ inputRange: [-width, 0, width], outputRange: ["-12deg", "0deg", "12deg"] });
  const likeOpacity = position.x.interpolate({ inputRange: [0, THRESHOLD], outputRange: [0, 1], extrapolate: "clamp" });
  const passOpacity = position.x.interpolate({ inputRange: [-THRESHOLD, 0], outputRange: [1, 0], extrapolate: "clamp" });

  const photos = profile.photos;
  const current = photos[Math.min(photoIndex, Math.max(photos.length - 1, 0))];
  const shown = profile.interests.slice(0, 3);
  const summary = `${profile.firstName}, ${profile.age} ans${profile.city ? `, ${profile.city}` : ""}`;

  return (
    <Animated.View
      {...pan.panHandlers}
      accessible
      accessibilityLabel={summary}
      style={[
        styles.card,
        { backgroundColor: colors.surface, transform: [{ translateX: position.x }, { translateY: position.y }, { rotate }] }
      ]}
    >
      <PhotoImage url={current?.url} name={profile.firstName} style={StyleSheet.absoluteFill} />
      <View style={styles.indicators} pointerEvents="none">
        {photos.map((p, i) => (
          <View key={p.id} style={[styles.dot, { backgroundColor: i === photoIndex ? "#FFFFFF" : "rgba(255,255,255,0.4)" }]} />
        ))}
      </View>
      <View style={styles.tapZones}>
        <Pressable accessibilityLabel="Photo précédente" style={styles.tapZone} onPress={() => setPhotoIndex((i) => Math.max(0, i - 1))} />
        <Pressable
          accessibilityLabel="Photo suivante"
          style={styles.tapZone}
          onPress={() => setPhotoIndex((i) => Math.min(photos.length - 1, i + 1))}
        />
      </View>
      <Animated.View style={[styles.stamp, styles.likeStamp, { opacity: likeOpacity, borderColor: colors.like }]} pointerEvents="none">
        <Text variant="heading" color={colors.like}>
          J'AIME
        </Text>
      </Animated.View>
      <Animated.View style={[styles.stamp, styles.passStamp, { opacity: passOpacity, borderColor: colors.pass }]} pointerEvents="none">
        <Text variant="heading" color="#FFFFFF">
          PASSER
        </Text>
      </Animated.View>
      <View style={styles.footer}>
        <View style={styles.footerRow}>
          <View style={styles.flex}>
            <Text variant="title" color="#FFFFFF" numberOfLines={1}>
              {profile.firstName}, {profile.age}
            </Text>
            <Text color="#FFFFFF" numberOfLines={1}>
              {[profile.city, profile.distanceKm ? `à ~${profile.distanceKm} km` : null].filter(Boolean).join(" · ")}
            </Text>
          </View>
          <Pressable
            accessibilityRole="button"
            accessibilityLabel={`Voir le profil de ${profile.firstName}`}
            onPress={onOpenProfile}
            style={styles.info}
          >
            <Text variant="heading" color="#FFFFFF">
              i
            </Text>
          </Pressable>
        </View>
        <View style={styles.chips}>
          {shown.map((i) => (
            <Chip key={i.id} label={i.label} />
          ))}
        </View>
      </View>
    </Animated.View>
  );
});

const styles = StyleSheet.create({
  card: {
    ...StyleSheet.absoluteFillObject,
    borderRadius: radii.lg,
    overflow: "hidden",
    elevation: 4,
    shadowColor: "#000",
    shadowOpacity: 0.18,
    shadowRadius: 12,
    shadowOffset: { width: 0, height: 6 }
  },
  flex: { flex: 1 },
  indicators: { position: "absolute", top: 10, left: 12, right: 12, flexDirection: "row", gap: 4 },
  dot: { flex: 1, height: 3, borderRadius: 2 },
  tapZones: { ...StyleSheet.absoluteFillObject, flexDirection: "row", bottom: 140 },
  tapZone: { flex: 1 },
  stamp: { position: "absolute", top: 48, borderWidth: 3, borderRadius: radii.md, paddingHorizontal: 12, paddingVertical: 4 },
  likeStamp: { left: 24, transform: [{ rotate: "-10deg" }] },
  passStamp: { right: 24, transform: [{ rotate: "10deg" }], backgroundColor: "rgba(0,0,0,0.35)" },
  footer: { position: "absolute", left: 0, right: 0, bottom: 0, padding: 18, gap: 10, backgroundColor: "rgba(20,12,10,0.55)" },
  footerRow: { flexDirection: "row", alignItems: "center", gap: 12 },
  info: { width: 38, height: 38, borderRadius: 19, borderWidth: 2, borderColor: "#FFFFFF", alignItems: "center", justifyContent: "center" },
  chips: { flexDirection: "row", gap: 6, flexWrap: "wrap" }
});
