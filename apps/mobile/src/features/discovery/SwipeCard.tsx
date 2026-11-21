import React, { forwardRef, useImperativeHandle, useMemo, useRef, useState } from "react";
import { Animated, PanResponder, Pressable, View, useWindowDimensions } from "react-native";
import { LinearGradient } from "expo-linear-gradient";
import type { FeedItem, SwipeAction } from "../../api/types";
import { makeStyles } from "../../theme";
import { Icon, RemoteImage, Text } from "../../ui";
import { formatDistance } from "../../utils/format";

export type SwipeCardHandle = { swipe: (action: SwipeAction) => void };

type Props = {
  card: FeedItem;
  active: boolean;
  onDecision: (card: FeedItem, action: SwipeAction) => void;
  onOpen: (card: FeedItem) => void;
};

export const SwipeCard = forwardRef<SwipeCardHandle, Props>(function SwipeCard({ card, active, onDecision, onOpen }, ref) {
  const styles = useStyles();
  const { width } = useWindowDimensions();
  const position = useRef(new Animated.ValueXY()).current;
  const [photoIndex, setPhotoIndex] = useState(0);
  const decided = useRef(false);
  const threshold = Math.min(width, 640) * 0.28;

  const flyOut = (action: SwipeAction) => {
    if (decided.current) return;
    decided.current = true;
    Animated.timing(position, {
      toValue: { x: (action === "like" ? 1 : -1) * (Math.min(width, 640) * 1.4), y: 40 },
      duration: 240,
      useNativeDriver: false
    }).start(() => onDecision(card, action));
  };

  useImperativeHandle(ref, () => ({ swipe: flyOut }));

  const responder = useMemo(
    () =>
      PanResponder.create({
        onMoveShouldSetPanResponder: (_, g) => active && Math.abs(g.dx) > 8 && Math.abs(g.dx) > Math.abs(g.dy),
        onPanResponderMove: Animated.event([null, { dx: position.x, dy: position.y }], { useNativeDriver: false }),
        onPanResponderRelease: (_, g) => {
          if (g.dx > threshold || g.vx > 1.2) {
            flyOut("like");
          } else if (g.dx < -threshold || g.vx < -1.2) {
            flyOut("pass");
          } else {
            Animated.spring(position, { toValue: { x: 0, y: 0 }, friction: 6, useNativeDriver: false }).start();
          }
        },
        onPanResponderTerminate: () => {
          Animated.spring(position, { toValue: { x: 0, y: 0 }, friction: 6, useNativeDriver: false }).start();
        }
      }),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [active, threshold]
  );

  const rotate = position.x.interpolate({ inputRange: [-width, 0, width], outputRange: ["-12deg", "0deg", "12deg"] });
  const likeOpacity = position.x.interpolate({ inputRange: [0, threshold], outputRange: [0, 1], extrapolate: "clamp" });
  const passOpacity = position.x.interpolate({ inputRange: [-threshold, 0], outputRange: [1, 0], extrapolate: "clamp" });

  const photo = card.photos[photoIndex] ?? card.photos[0];
  const distance = formatDistance(card.distanceKm);
  const details = [card.jobTitle, card.city, distance].filter(Boolean).join(" · ");

  return (
    <Animated.View
      testID={active ? "discovery-card" : undefined}
      accessible
      accessibilityLabel={`${card.firstName}, ${card.age} ans${details ? `, ${details}` : ""}${card.likedYou ? ", vous a liké" : ""}`}
      accessibilityHint="Utilisez les boutons J'aime ou Passer, ou ouvrez le profil complet."
      accessibilityActions={[
        { name: "like", label: "J'aime" },
        { name: "pass", label: "Passer" },
        { name: "activate", label: "Voir le profil" }
      ]}
      onAccessibilityAction={(e) => {
        if (e.nativeEvent.actionName === "like") flyOut("like");
        else if (e.nativeEvent.actionName === "pass") flyOut("pass");
        else onOpen(card);
      }}
      style={[styles.card, active ? { transform: [{ translateX: position.x }, { translateY: position.y }, { rotate }] } : styles.behind]}
      {...(active ? responder.panHandlers : {})}
    >
      <RemoteImage uri={photo?.url} style={styles.image} recyclingKey={photo?.id} />

      {card.photos.length > 1 ? (
        <View style={styles.progress} pointerEvents="none">
          {card.photos.map((p, i) => (
            <View key={p.id} style={[styles.segment, i === photoIndex ? styles.segmentActive : null]} />
          ))}
        </View>
      ) : null}

      <View style={styles.tapZones}>
        <Pressable accessibilityLabel="Photo précédente" style={styles.tapZone} onPress={() => setPhotoIndex((i) => Math.max(0, i - 1))} />
        <Pressable
          accessibilityLabel="Photo suivante"
          style={styles.tapZone}
          onPress={() => setPhotoIndex((i) => Math.min(card.photos.length - 1, i + 1))}
        />
      </View>

      <Animated.View style={[styles.stamp, styles.stampLike, { opacity: likeOpacity }]} pointerEvents="none">
        <Text variant="heading" tone="onImage">
          J&apos;AIME
        </Text>
      </Animated.View>
      <Animated.View style={[styles.stamp, styles.stampPass, { opacity: passOpacity }]} pointerEvents="none">
        <Text variant="heading" tone="onImage">
          PASSER
        </Text>
      </Animated.View>

      <LinearGradient colors={["transparent", "rgba(10,6,4,0.82)"]} style={styles.gradient} pointerEvents="box-none">
        <Pressable
          accessibilityRole="button"
          accessibilityLabel={`Voir le profil de ${card.firstName}`}
          onPress={() => onOpen(card)}
          style={styles.info}
          testID={active ? "discovery-open-profile" : undefined}
        >
          <View style={styles.badges}>
            {card.likedYou ? (
              <View style={styles.badge}>
                <Icon name="heart" size={13} color="#FFFFFF" />
                <Text variant="micro" tone="onImage">
                  VOUS A LIKÉ
                </Text>
              </View>
            ) : null}
            {card.isDemo ? (
              <View style={[styles.badge, styles.demoBadge]}>
                <Text variant="micro" tone="onImage">
                  DÉMO
                </Text>
              </View>
            ) : null}
          </View>
          <View style={styles.nameRow}>
            <Text variant="title" tone="onImage" numberOfLines={1} style={styles.name}>
              {card.firstName}, {card.age}
            </Text>
            <Icon name="information-circle-outline" size={26} color="#FFFFFF" />
          </View>
          {details ? (
            <Text variant="label" tone="onImage" numberOfLines={1} style={styles.details}>
              {details}
            </Text>
          ) : null}
          {card.interests.length > 0 ? (
            <View style={styles.interests}>
              {card.interests.slice(0, 3).map((interest) => (
                <View key={interest.id} style={styles.interest}>
                  <Text variant="caption" tone="onImage">
                    {interest.label}
                  </Text>
                </View>
              ))}
            </View>
          ) : null}
        </Pressable>
      </LinearGradient>
    </Animated.View>
  );
});

const fill = { position: "absolute" as const, top: 0, left: 0, right: 0, bottom: 0 };

const useStyles = makeStyles((t) => ({
  card: {
    ...fill,
    borderRadius: t.radii.xl,
    overflow: "hidden",
    backgroundColor: t.colors.skeleton,
    shadowColor: t.colors.shadow,
    shadowOpacity: 0.18,
    shadowRadius: 20,
    shadowOffset: { width: 0, height: 10 },
    elevation: 5
  },
  behind: { transform: [{ scale: 0.95 }, { translateY: 14 }], opacity: 0.9 },
  image: fill,
  progress: { position: "absolute", top: t.spacing.md, left: t.spacing.md, right: t.spacing.md, flexDirection: "row", gap: t.spacing.xs },
  segment: { flex: 1, height: 3, borderRadius: 2, backgroundColor: "rgba(255,255,255,0.35)" },
  segmentActive: { backgroundColor: "#FFFFFF" },
  tapZones: { ...fill, bottom: "35%", flexDirection: "row" },
  tapZone: { flex: 1 },
  stamp: {
    position: "absolute",
    top: 56,
    paddingHorizontal: t.spacing.md,
    paddingVertical: t.spacing.xs,
    borderRadius: t.radii.sm,
    borderWidth: 2,
    borderColor: "#FFFFFF"
  },
  stampLike: { left: t.spacing.xl, backgroundColor: "rgba(180,70,47,0.85)", transform: [{ rotate: "-10deg" }] },
  stampPass: { right: t.spacing.xl, backgroundColor: "rgba(40,34,30,0.85)", transform: [{ rotate: "10deg" }] },
  gradient: { position: "absolute", left: 0, right: 0, bottom: 0, paddingTop: 90 },
  info: { padding: t.spacing.xl, gap: t.spacing.xs },
  badges: { flexDirection: "row", gap: t.spacing.xs },
  badge: {
    flexDirection: "row",
    alignItems: "center",
    gap: 4,
    backgroundColor: "rgba(180,70,47,0.92)",
    borderRadius: t.radii.pill,
    paddingHorizontal: t.spacing.sm,
    paddingVertical: 3
  },
  demoBadge: { backgroundColor: "rgba(255,255,255,0.25)" },
  nameRow: { flexDirection: "row", alignItems: "center", gap: t.spacing.sm },
  name: { flexShrink: 1 },
  details: { opacity: 0.92 },
  interests: { flexDirection: "row", flexWrap: "wrap", gap: t.spacing.xs, marginTop: t.spacing.xs },
  interest: {
    borderRadius: t.radii.pill,
    borderWidth: 1,
    borderColor: "rgba(255,255,255,0.45)",
    paddingHorizontal: t.spacing.sm,
    paddingVertical: 2
  }
}));
