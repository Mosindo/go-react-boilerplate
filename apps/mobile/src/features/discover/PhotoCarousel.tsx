import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import type { Photo } from "../../api/types";
import { RemoteImage, Text, radius, useColors } from "../../shared/ui";

type Props = { photos: Photo[]; name: string; children?: React.ReactNode; rounded?: boolean };

/** Full-bleed photo with tap-left / tap-right navigation and segment indicators. */
export function PhotoCarousel({ photos, name, children, rounded = true }: Props) {
  const c = useColors();
  const [index, setIndex] = useState(0);
  const safeIndex = Math.min(index, Math.max(0, photos.length - 1));
  const current = photos[safeIndex];
  const go = (delta: number) => setIndex((i) => Math.min(photos.length - 1, Math.max(0, i + delta)));

  return (
    <View style={[styles.fill, rounded ? { borderRadius: radius.lg, overflow: "hidden" } : null, { backgroundColor: c.surfaceMuted }]}>
      {current ? (
        <RemoteImage path={current.url} priority="high" accessibilityLabel={`Photo ${safeIndex + 1} sur ${photos.length} de ${name}`} style={styles.fill} />
      ) : (
        <View style={[styles.fill, styles.center]}><Text tone="muted">Pas de photo</Text></View>
      )}
      {photos[safeIndex + 1] ? <RemoteImage path={photos[safeIndex + 1]?.url} priority="low" style={styles.preload} /> : null}
      <View style={styles.tapZones} pointerEvents="box-none">
        <Pressable style={styles.zone} onPress={() => go(-1)} accessibilityRole="button" accessibilityLabel="Photo précédente" />
        <Pressable style={styles.zone} onPress={() => go(1)} accessibilityRole="button" accessibilityLabel="Photo suivante" />
      </View>
      {photos.length > 1 ? (
        <View style={styles.bars} pointerEvents="none">
          {photos.map((p, i) => (
            <View key={p.id} style={[styles.bar, { backgroundColor: i === safeIndex ? "#FFFFFF" : "rgba(255,255,255,0.4)" }]} />
          ))}
        </View>
      ) : null}
      {children}
    </View>
  );
}

const styles = StyleSheet.create({
  fill: { flex: 1, width: "100%", height: "100%" },
  center: { alignItems: "center", justifyContent: "center" },
  preload: { position: "absolute", width: 1, height: 1, opacity: 0 },
  tapZones: { ...StyleSheet.absoluteFillObject, flexDirection: "row" },
  zone: { flex: 1 },
  bars: { position: "absolute", top: 10, left: 12, right: 12, flexDirection: "row", gap: 4 },
  bar: { flex: 1, height: 4, borderRadius: 2 },
});
