import { Ionicons } from "@expo/vector-icons";
import React from "react";
import { StyleSheet, View } from "react-native";

import type { Card } from "../api/types";
import { formatDistance } from "../lib/format";
import { spacing, useTheme } from "../theme";
import { Chip } from "./Chip";
import { Text } from "./Text";

export function ProfileBody({ card }: { card: Card }) {
  const { colors } = useTheme();
  const distance = formatDistance(card.distanceKm);
  return (
    <View style={styles.body}>
      <View style={styles.row}>
        <Text variant="title">{card.firstName}</Text>
        <Text variant="title" tone="muted">
          {card.age}
        </Text>
      </View>
      {card.city || distance ? (
        <View style={styles.row}>
          <Ionicons name="location-outline" size={16} color={colors.textMuted} />
          <Text tone="muted">{[card.city, distance].filter(Boolean).join(" · ")}</Text>
        </View>
      ) : null}
      {card.bio ? <Text>{card.bio}</Text> : null}
      {card.interests.length > 0 ? (
        <View style={styles.chips}>
          {card.interests.map((i) => (
            <Chip key={i.slug} label={i.label} highlight={i.shared} />
          ))}
        </View>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  body: { gap: spacing.md },
  row: { flexDirection: "row", alignItems: "center", gap: spacing.sm },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
});
