import React, { useState } from "react";
import { Image, Pressable, View } from "react-native";
import { AppText, Chip, Row } from "./ui";
import { mediaUrl } from "../api/config";
import { formatDistance } from "../utils/format";
import { radius, spacing, useTheme } from "../theme/theme";
import type { Card } from "../api/types";

/** Photo area with tap zones (left/right) to move between photos, and progress bars. */
export function PhotoCarousel({ card, height }: { card: Card; height?: number }) {
  const { colors } = useTheme();
  const [index, setIndex] = useState(0);
  const photos = card.photos;
  const current = photos[Math.min(index, Math.max(photos.length - 1, 0))];
  const go = (delta: number) =>
    setIndex((i) => Math.min(Math.max(i + delta, 0), Math.max(photos.length - 1, 0)));
  const distance = formatDistance(card.distanceKm);

  return (
    <View
      testID="photo-carousel"
      style={[
        { borderRadius: radius.lg, overflow: "hidden", backgroundColor: colors.surfaceAlt },
        height ? { height } : { aspectRatio: 3 / 4 },
      ]}
    >
      {current ? (
        <Image
          accessibilityLabel={`Photo ${index + 1} sur ${photos.length} de ${card.firstName}`}
          source={{ uri: mediaUrl(current.url) }}
          style={{ position: "absolute", width: "100%", height: "100%" }}
          resizeMode="cover"
        />
      ) : null}
      <Row style={{ position: "absolute", top: spacing.sm, left: spacing.sm, right: spacing.sm, gap: 4 }}>
        {photos.length > 1
          ? photos.map((p, i) => (
              <View
                key={p.id}
                style={{
                  flex: 1,
                  height: 3,
                  borderRadius: 2,
                  backgroundColor: i === index ? "#fff" : "rgba(255,255,255,0.4)",
                }}
              />
            ))
          : null}
      </Row>
      <Row style={{ position: "absolute", top: 0, bottom: 0, left: 0, right: 0, gap: 0 }}>
        <Pressable
          style={{ flex: 1, height: "100%" }}
          onPress={() => go(-1)}
          accessibilityLabel="Photo précédente"
        />
        <Pressable
          style={{ flex: 1, height: "100%" }}
          onPress={() => go(1)}
          accessibilityLabel="Photo suivante"
        />
      </Row>
      <View
        pointerEvents="none"
        style={{
          position: "absolute",
          left: 0,
          right: 0,
          bottom: 0,
          padding: spacing.lg,
          backgroundColor: "rgba(0,0,0,0.42)",
          gap: 2,
        }}
      >
        <AppText variant="heading" style={{ color: "#fff", fontSize: 26 }} testID="card-name">
          {card.firstName}, {card.age}
        </AppText>
        {card.city || distance ? (
          <AppText variant="small" style={{ color: "#fff" }}>
            {[card.city, distance].filter(Boolean).join(" · ")}
          </AppText>
        ) : null}
      </View>
    </View>
  );
}

export function ProfileDetails({ card }: { card: Card }) {
  return (
    <View style={{ gap: spacing.lg }}>
      {card.bio ? (
        <View style={{ gap: spacing.xs }}>
          <AppText variant="heading">À propos</AppText>
          <AppText>{card.bio}</AppText>
        </View>
      ) : null}
      {card.interests.length > 0 ? (
        <View style={{ gap: spacing.sm }}>
          <AppText variant="heading">
            Centres d&apos;intérêt{card.sharedInterests > 0 ? ` · ${card.sharedInterests} en commun` : ""}
          </AppText>
          <Row style={{ flexWrap: "wrap", gap: spacing.sm }}>
            {card.interests.map((i) => (
              <Chip key={i.slug} label={i.label} />
            ))}
          </Row>
        </View>
      ) : null}
    </View>
  );
}
