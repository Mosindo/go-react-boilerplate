import React from "react";
import { StyleSheet, View } from "react-native";
import type { ProfileCard } from "../../api/types";
import { formatDistance } from "../../lib/format";
import { spacing } from "../../theme";
import { Chip, ChipGroup, Icon, Text, type IconName } from "../../ui";

export function Fact({ icon, label, light }: { icon: IconName; label: string; light?: boolean }) {
  return (
    <View style={styles.fact}>
      <Icon name={icon} size={15} color="textMuted" rawColor={light ? "#F7EFE9" : undefined} />
      <Text variant="caption" color={light ? "text" : "textMuted"} style={light ? styles.lightText : undefined}>
        {label}
      </Text>
    </View>
  );
}

export function facts(card: ProfileCard): { icon: IconName; label: string }[] {
  const items: { icon: IconName; label: string }[] = [];
  if (card.jobTitle) items.push({ icon: "briefcase-outline", label: card.jobTitle });
  const distance = formatDistance(card.distanceKm);
  if (distance) items.push({ icon: "location-outline", label: distance });
  else if (card.city) items.push({ icon: "location-outline", label: card.city });
  if (card.heightCm) items.push({ icon: "resize-outline", label: `${card.heightCm} cm` });
  return items;
}

export function InterestChips({ card, mine }: { card: ProfileCard; mine: number[] }) {
  if (card.interests.length === 0) return null;
  const shared = card.interests.filter((i) => mine.includes(i.id));
  const others = card.interests.filter((i) => !mine.includes(i.id));
  return (
    <ChipGroup>
      {[...shared, ...others].map((i) => (
        <Chip key={i.id} label={i.label} highlighted={mine.includes(i.id)} icon={mine.includes(i.id) ? "sparkles" : undefined} />
      ))}
    </ChipGroup>
  );
}

const styles = StyleSheet.create({
  fact: { flexDirection: "row", alignItems: "center", gap: 5 },
  lightText: { color: "#F7EFE9" }
});

export const factStyles = StyleSheet.create({ row: { flexDirection: "row", flexWrap: "wrap", columnGap: spacing.md, rowGap: 4 } });
