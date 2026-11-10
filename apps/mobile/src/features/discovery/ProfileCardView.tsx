import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Image } from "expo-image";
import { LinearGradient } from "expo-linear-gradient";
import { Ionicons } from "@expo/vector-icons";
import { mediaSource } from "../../api/client";
import type { ProfileCard } from "../../api/types";
import { useTheme } from "../../theme/ThemeProvider";
import { Text } from "../../ui";
import { formatDistance } from "../../utils/format";

type Props = {
  card: ProfileCard;
  onOpen?: () => void;
  interactive?: boolean;
};

/** Full-bleed photo card with tap zones to browse photos. */
export function ProfileCardView({ card, onOpen, interactive = true }: Props) {
  const theme = useTheme();
  const [index, setIndex] = useState(0);
  const photo = card.photos[index] ?? card.photos[0];
  const distance = formatDistance(card.distanceKm);
  const meta = [card.jobTitle, distance].filter(Boolean).join(" · ");

  return (
    <View style={[styles.card, { borderRadius: theme.radii.xl, backgroundColor: theme.colors.surfaceAlt }]}>
      {photo ? (
        <Image
          accessibilityLabel={`Photo ${index + 1} sur ${card.photos.length} de ${card.firstName}`}
          cachePolicy="memory-disk"
          contentFit="cover"
          priority="high"
          recyclingKey={photo.id}
          source={mediaSource(photo)}
          style={StyleSheet.absoluteFill}
          transition={120}
        />
      ) : null}

      {card.photos.length > 1 ? (
        <View style={styles.bars}>
          {card.photos.map((p, i) => (
            <View key={p.id} style={[styles.bar, { backgroundColor: i === index ? "#FFFFFF" : "rgba(255,255,255,0.4)" }]} />
          ))}
        </View>
      ) : null}

      {interactive && card.photos.length > 1 ? (
        <View style={styles.tapZones}>
          <Pressable accessibilityLabel="Photo précédente" onPress={() => setIndex((i) => Math.max(0, i - 1))} style={styles.tapZone} />
          <Pressable accessibilityLabel="Photo suivante" onPress={() => setIndex((i) => Math.min(card.photos.length - 1, i + 1))} style={styles.tapZone} />
        </View>
      ) : null}

      <LinearGradient colors={["transparent", "rgba(15,10,12,0.15)", "rgba(15,10,12,0.85)"]} pointerEvents="box-none" style={styles.gradient}>
        <Pressable accessibilityHint="Ouvre le profil complet" accessibilityRole="button" disabled={!onOpen} onPress={onOpen} style={styles.info}>
          <View style={styles.nameRow}>
            <Text color="#FFFFFF" numberOfLines={1} style={styles.name} variant="display">
              {card.firstName}
              <Text color="rgba(255,255,255,0.85)" variant="title">
                {"  "}
                {card.age}
              </Text>
            </Text>
            {onOpen ? <Ionicons color="#FFFFFF" name="information-circle-outline" size={28} /> : null}
          </View>
          {meta ? (
            <Text color="rgba(255,255,255,0.9)" numberOfLines={1} variant="small">
              {meta}
            </Text>
          ) : null}
          {card.interests.length ? (
            <View style={styles.chips}>
              {card.interests.slice(0, 4).map((i) => (
                <View key={i.id} style={styles.chip}>
                  <Text color="#FFFFFF" variant="caption">
                    {i.label}
                  </Text>
                </View>
              ))}
            </View>
          ) : null}
        </Pressable>
      </LinearGradient>
    </View>
  );
}

const styles = StyleSheet.create({
  card: { flex: 1, overflow: "hidden" },
  bars: { position: "absolute", top: 10, left: 12, right: 12, flexDirection: "row", gap: 4, zIndex: 2 },
  bar: { flex: 1, height: 3, borderRadius: 2 },
  tapZones: { ...StyleSheet.absoluteFillObject, flexDirection: "row", bottom: "35%" },
  tapZone: { flex: 1 },
  gradient: { position: "absolute", left: 0, right: 0, bottom: 0, paddingTop: 120, paddingHorizontal: 18, paddingBottom: 18 },
  info: { gap: 6 },
  nameRow: { flexDirection: "row", alignItems: "flex-end", gap: 8 },
  name: { flex: 1 },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: 6, marginTop: 4 },
  chip: { backgroundColor: "rgba(255,255,255,0.18)", borderRadius: 999, paddingHorizontal: 10, paddingVertical: 4 }
});
