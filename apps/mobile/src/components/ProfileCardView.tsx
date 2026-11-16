import React from "react";
import { StyleSheet, View } from "react-native";
import { LinearGradient } from "expo-linear-gradient";
import { Ionicons } from "@expo/vector-icons";
import type { ProfileCard } from "../api/types";
import { colors, spacing } from "../theme/tokens";
import { Chip, IconButton, Text } from "../ui";
import { distanceLabel, lookingForShort } from "../utils/format";
import { PhotoCarousel } from "./PhotoCarousel";

type Props = {
  card: ProfileCard;
  onOpen?: () => void;
};

export function ProfileCardView({ card, onOpen }: Props) {
  const distance = distanceLabel(card.distanceKm);
  const shared = card.interests.filter((i) => i.shared).slice(0, 3);
  return (
    <PhotoCarousel photos={card.photos} name={card.firstName}>
      <LinearGradient
        pointerEvents="box-none"
        colors={["transparent", "rgba(10,7,6,0.55)", "rgba(10,7,6,0.92)"]}
        locations={[0, 0.45, 1]}
        style={styles.overlay}
      >
        <View style={styles.infoRow} pointerEvents="box-none">
          <View style={styles.info} pointerEvents="none">
            <Text variant="display" style={styles.name} numberOfLines={1}>
              {card.firstName}
              <Text variant="title" style={styles.age}>
                {"  "}
                {card.age}
              </Text>
            </Text>
            <View style={styles.metaRow}>
              {card.jobTitle ? <Meta icon="briefcase-outline" label={card.jobTitle} /> : null}
              {distance ? <Meta icon="location-outline" label={distance} /> : card.city ? <Meta icon="location-outline" label={card.city} /> : null}
            </View>
            <View style={styles.chips}>
              <Chip label={lookingForShort[card.lookingFor]} icon="sparkles-outline" />
              {shared.map((i) => (
                <Chip key={i.id} label={i.label} highlighted />
              ))}
            </View>
          </View>
          {onOpen ? (
            <IconButton
              icon="arrow-up"
              label={`Voir le profil de ${card.firstName}`}
              onPress={onOpen}
              background="rgba(255,255,255,0.18)"
              size={42}
              testID="open-profile"
            />
          ) : null}
        </View>
      </LinearGradient>
    </PhotoCarousel>
  );
}

function Meta({ icon, label }: { icon: keyof typeof Ionicons.glyphMap; label: string }) {
  return (
    <View style={styles.meta}>
      <Ionicons name={icon} size={15} color={colors.textMuted} />
      <Text variant="caption" tone="muted" numberOfLines={1}>
        {label}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  overlay: { position: "absolute", left: 0, right: 0, bottom: 0, paddingTop: 120, paddingHorizontal: spacing.xl, paddingBottom: spacing.xl, zIndex: 3 },
  infoRow: { flexDirection: "row", alignItems: "flex-end", gap: spacing.md },
  info: { flex: 1, gap: spacing.sm },
  name: { color: colors.white },
  age: { color: colors.white, fontWeight: "400" },
  metaRow: { flexDirection: "row", flexWrap: "wrap", gap: spacing.md },
  meta: { flexDirection: "row", alignItems: "center", gap: 4, maxWidth: "100%" },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.xs }
});
