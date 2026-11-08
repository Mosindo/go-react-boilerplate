import React from "react";
import { StyleSheet, View } from "react-native";
import type { Gender } from "../../api/types";
import { spacing } from "../../theme/theme";
import { Chip } from "../../ui/Chip";
import { Text } from "../../ui/Text";

export const GENDER_LABEL: Record<Gender, string> = {
  man: "Homme",
  woman: "Femme",
  non_binary: "Non-binaire"
};
export const GENDERS: Gender[] = ["man", "woman", "non_binary"];

export function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <View style={styles.section}>
      <Text variant="label" tone="textSubtle" style={styles.title}>
        {title.toUpperCase()}
      </Text>
      {children}
    </View>
  );
}

export function GenderChips({
  value,
  onToggle,
  multi
}: {
  value: Gender[];
  onToggle: (g: Gender) => void;
  multi?: boolean;
}) {
  return (
    <View style={styles.chips} accessibilityRole={multi ? undefined : "radiogroup"}>
      {GENDERS.map((g) => (
        <Chip
          key={g}
          label={GENDER_LABEL[g]}
          selected={value.includes(g)}
          onPress={() => onToggle(g)}
          testID={`${multi ? "pref" : "gender"}-${g}`}
        />
      ))}
    </View>
  );
}

const styles = StyleSheet.create({
  section: { marginBottom: spacing.lg },
  title: { letterSpacing: 1, marginBottom: spacing.xs },
  chips: { flexDirection: "row", flexWrap: "wrap" }
});
