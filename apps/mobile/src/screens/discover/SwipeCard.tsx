import React, { useImperativeHandle, useRef, useState, forwardRef } from "react";
import { Animated, PanResponder, Pressable, StyleSheet, View, useWindowDimensions } from "react-native";
import { LinearGradient } from "expo-linear-gradient";
import type { PublicProfile, SwipeAction } from "../../api/types";
import { DemoBadge, ProfileFacts } from "../../components/ProfileBadges";
import { radii, spacing, useTheme } from "../../theme";
import { Chip, Icon, RemoteImage, Text } from "../../ui";

export type SwipeCardHandle = { swipe: (action: SwipeAction) => void };

type Props = {
  profile: PublicProfile;
  onSwiped: (action: SwipeAction) => void;
  onOpen: () => void;
  active: boolean;
};

const ROTATION = 12;

export const SwipeCard = forwardRef<SwipeCardHandle, Props>(function SwipeCard({ profile, onSwiped, onOpen, active }, ref) {
  const { colors } = useTheme();
  const { width } = useWindowDimensions();
  const threshold = Math.min(width, 520) * 0.28;
  const position = useRef(new Animated.ValueXY()).current;
  const [photoIndex, setPhotoIndex] = useState(0);
  const leaving = useRef(false);

  const flyOut = (action: SwipeAction) => {
    if (leaving.current) return;
    leaving.current = true;
    const x = (action === "like" ? 1 : -1) * width * 1.3;
    Animated.timing(position, { toValue: { x, y: 40 }, duration: 260, useNativeDriver: false }).start(() => onSwiped(action));
  };

  useImperativeHandle(ref, () => ({ swipe: flyOut }));

  const responder = useRef(
    PanResponder.create({
      onMoveShouldSetPanResponder: (_, g) => Math.abs(g.dx) > 8 && Math.abs(g.dx) > Math.abs(g.dy),
      onPanResponderMove: (_, g) => position.setValue({ x: g.dx, y: g.dy * 0.25 }),
      onPanResponderRelease: (_, g) => {
        if (g.dx > threshold || g.vx > 1.2) flyOut("like");
        else if (g.dx < -threshold || g.vx < -1.2) flyOut("pass");
        else Animated.spring(position, { toValue: { x: 0, y: 0 }, friction: 6, useNativeDriver: false }).start();
      },
      onPanResponderTerminate: () => Animated.spring(position, { toValue: { x: 0, y: 0 }, useNativeDriver: false }).start()
    })
  ).current;

  const rotate = position.x.interpolate({ inputRange: [-width, 0, width], outputRange: [`-${ROTATION}deg`, "0deg", `${ROTATION}deg`] });
  const likeOpacity = position.x.interpolate({ inputRange: [20, threshold], outputRange: [0, 1], extrapolate: "clamp" });
  const passOpacity = position.x.interpolate({ inputRange: [-threshold, -20], outputRange: [1, 0], extrapolate: "clamp" });

  const photos = profile.photos;
  const photo = photos[Math.min(photoIndex, photos.length - 1)];

  return (
    <Animated.View
      {...(active ? responder.panHandlers : {})}
      accessibilityHint="Glissez à droite pour liker, à gauche pour passer"
      accessibilityLabel={`${profile.firstName}, ${profile.age} ans`}
      style={[
        styles.card,
        { backgroundColor: colors.surfaceMuted, shadowColor: colors.shadow },
        { transform: [{ translateX: position.x }, { translateY: position.y }, { rotate }] }
      ]}
      testID={active ? "swipe-card" : undefined}
    >
      {photo ? <RemoteImage style={StyleSheet.absoluteFill} url={photo.url} /> : null}

      {photos.length > 1 ? (
        <View style={styles.bars}>
          {photos.map((p, i) => (
            <View key={p.id} style={[styles.bar, { backgroundColor: i === photoIndex ? "#FFFFFF" : "rgba(255,255,255,0.4)" }]} />
          ))}
        </View>
      ) : null}

      <View style={styles.tapZones}>
        <Pressable accessibilityLabel="Photo précédente" onPress={() => setPhotoIndex((i) => Math.max(0, i - 1))} style={styles.tapZone} />
        <Pressable
          accessibilityLabel="Photo suivante"
          onPress={() => setPhotoIndex((i) => Math.min(photos.length - 1, i + 1))}
          style={styles.tapZone}
        />
      </View>

      <Animated.View style={[styles.stamp, styles.likeStamp, { opacity: likeOpacity, borderColor: colors.success }]}>
        <Text style={{ color: colors.success }} variant="title">
          J’AIME
        </Text>
      </Animated.View>
      <Animated.View style={[styles.stamp, styles.passStamp, { opacity: passOpacity, borderColor: "#FFFFFF" }]}>
        <Text tone="inverse" variant="title">
          PASSER
        </Text>
      </Animated.View>

      <LinearGradient colors={["transparent", "rgba(15,8,12,0.55)", "rgba(15,8,12,0.9)"]} pointerEvents="box-none" style={styles.info}>
        <Pressable
          accessibilityHint="Voir le profil complet"
          accessibilityRole="button"
          onPress={onOpen}
          style={styles.infoPress}
          testID={active ? "swipe-card-open" : undefined}
        >
          <View style={styles.nameRow}>
            <Text numberOfLines={1} style={styles.name} tone="inverse" variant="title">
              {profile.firstName}, {profile.age}
            </Text>
            {profile.isDemo ? <DemoBadge /> : null}
            <Icon color="#FFFFFF" name="information-circle-outline" size={24} />
          </View>
          <ProfileFacts light profile={profile} />
          {profile.bio ? (
            <Text numberOfLines={2} tone="inverse">
              {profile.bio}
            </Text>
          ) : null}
          {profile.interests.length > 0 ? (
            <View style={styles.chips}>
              {profile.interests.slice(0, 3).map((i) => (
                <Chip key={i.slug} label={i.label} />
              ))}
            </View>
          ) : null}
        </Pressable>
      </LinearGradient>
    </Animated.View>
  );
});

const styles = StyleSheet.create({
  card: {
    ...StyleSheet.absoluteFillObject,
    borderRadius: radii.xl,
    overflow: "hidden",
    shadowOpacity: 0.18,
    shadowRadius: 24,
    shadowOffset: { width: 0, height: 12 },
    elevation: 6
  },
  bars: { position: "absolute", top: spacing.sm, left: spacing.sm, right: spacing.sm, flexDirection: "row", gap: 4 },
  bar: { flex: 1, height: 4, borderRadius: 2 },
  tapZones: { ...StyleSheet.absoluteFillObject, flexDirection: "row", bottom: "40%" },
  tapZone: { flex: 1 },
  stamp: { position: "absolute", top: 48, borderWidth: 4, borderRadius: radii.md, paddingHorizontal: spacing.sm, paddingVertical: 2 },
  likeStamp: { left: 24, transform: [{ rotate: "-14deg" }] },
  passStamp: { right: 24, transform: [{ rotate: "14deg" }] },
  info: { position: "absolute", left: 0, right: 0, bottom: 0, paddingTop: 80 },
  infoPress: { padding: spacing.lg, gap: spacing.xs },
  nameRow: { flexDirection: "row", alignItems: "center", gap: spacing.xs },
  name: { flexShrink: 1 },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: 6, marginTop: 4 }
});
