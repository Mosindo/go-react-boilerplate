import React, { useState } from "react";
import { Pressable, ScrollView, StyleSheet, Text, View } from "react-native";
import type { Card } from "../api/types";
import { formatDistance, pluralize } from "../lib/format";
import { fontSizes, radii, spacing, useTheme } from "../theme/theme";
import { AppText, Chip, PhotoImage } from "./ui";

/** Photo carousel (tap left/right) with name, age and distance overlaid. */
export function CardPhoto({ card, onInfo, children }: { card: Card; onInfo?: () => void; children?: React.ReactNode }) {
  const [index, setIndex] = useState(0);
  const photo = card.photos[Math.min(index, card.photos.length - 1)];
  const distance = formatDistance(card.distanceKm);
  const go = (delta: number) => setIndex((i) => Math.max(0, Math.min(card.photos.length - 1, i + delta)));
  return (
    <View style={styles.photoWrap}>
      <PhotoImage url={photo?.url} style={StyleSheet.absoluteFill} label={`Photo de ${card.firstName}`} />
      <View style={styles.tapZones} pointerEvents="box-none">
        <Pressable style={styles.grow} accessibilityLabel="Photo précédente" onPress={() => go(-1)} />
        <Pressable style={styles.grow} accessibilityLabel="Photo suivante" onPress={() => go(1)} />
      </View>
      {card.photos.length > 1 ? (
        <View style={styles.bars} pointerEvents="none">
          {card.photos.map((p, i) => (
            <View key={p.id} style={[styles.bar, { backgroundColor: i === index ? "#FFFFFF" : "rgba(255,255,255,0.4)" }]} />
          ))}
        </View>
      ) : null}
      <View style={styles.scrim} pointerEvents="box-none">
        <View style={styles.scrimInner}>
          <View style={styles.grow} pointerEvents="none">
            <Text style={styles.name}>
              {card.firstName}, {card.age}
            </Text>
            {distance || card.city ? <Text style={styles.meta}>{[card.city, distance].filter(Boolean).join(" · ")}</Text> : null}
          </View>
          {onInfo ? (
            <Pressable
              accessibilityRole="button"
              accessibilityLabel={`Voir le profil de ${card.firstName}`}
              onPress={onInfo}
              style={styles.info}
              hitSlop={8}
            >
              <Text style={styles.infoText}>i</Text>
            </Pressable>
          ) : null}
        </View>
      </View>
      {children}
    </View>
  );
}

export function CardDetails({ card }: { card: Card }) {
  const { colors } = useTheme();
  return (
    <ScrollView contentContainerStyle={{ padding: spacing.lg, gap: spacing.lg }}>
      {card.bio ? (
        <View style={{ gap: spacing.xs }}>
          <AppText variant="label" muted>
            À propos
          </AppText>
          <AppText>{card.bio}</AppText>
        </View>
      ) : null}
      {card.interests.length > 0 ? (
        <View style={{ gap: spacing.sm }}>
          <AppText variant="label" muted>
            Centres d'intérêt{card.sharedInterests > 0 ? ` · ${card.sharedInterests} en commun` : ""}
          </AppText>
          <View style={{ flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }}>
            {card.interests.map((i) => (
              <Chip key={i.id} label={i.label} small />
            ))}
          </View>
        </View>
      ) : null}
      {card.photos.slice(1).map((p) => (
        <PhotoImage
          key={p.id}
          url={p.url}
          style={{ width: "100%", aspectRatio: 0.8, borderRadius: radii.lg, backgroundColor: colors.surfaceAlt }}
        />
      ))}
    </ScrollView>
  );
}

export function SharedBadge({ count }: { count: number }) {
  const { colors } = useTheme();
  if (count < 1) return null;
  return (
    <View style={[styles.shared, { backgroundColor: colors.accent }]}>
      <Text style={{ fontSize: fontSizes.xs, fontWeight: "800", color: "#2A1F2D" }}>
        {count} {pluralize(count, "intérêt commun", "intérêts communs")}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  grow: { flex: 1 },
  photoWrap: { flex: 1, overflow: "hidden", borderRadius: radii.lg, backgroundColor: "#222" },
  tapZones: { ...StyleSheet.absoluteFillObject, flexDirection: "row" },
  bars: { position: "absolute", top: 10, left: 10, right: 10, flexDirection: "row", gap: 4 },
  bar: { flex: 1, height: 3, borderRadius: 2 },
  scrim: { position: "absolute", left: 0, right: 0, bottom: 0, backgroundColor: "rgba(0,0,0,0.38)" },
  scrimInner: { flexDirection: "row", alignItems: "flex-end", padding: spacing.lg, gap: spacing.md },
  name: { color: "#FFFFFF", fontSize: fontSizes.xl, fontWeight: "800" },
  meta: { color: "rgba(255,255,255,0.9)", fontSize: fontSizes.sm, marginTop: 2 },
  info: { width: 36, height: 36, borderRadius: 18, borderWidth: 2, borderColor: "#FFFFFF", alignItems: "center", justifyContent: "center" },
  infoText: { color: "#FFFFFF", fontWeight: "800", fontStyle: "italic" },
  shared: { position: "absolute", top: 24, left: 12, borderRadius: radii.pill, paddingHorizontal: 10, paddingVertical: 4 }
});
