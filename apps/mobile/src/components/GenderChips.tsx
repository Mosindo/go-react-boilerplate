import React from "react";
import { StyleSheet, View } from "react-native";
import { GENDERS, type Gender } from "../api/types";
import { spacing } from "../theme";
import { Chip } from "../shared/ui";

export type GenderChipsProps = {
  labels: Record<Gender, string>;
  selected: Gender[];
  onToggle: (gender: Gender) => void;
};

export function GenderChips({ labels, selected, onToggle }: GenderChipsProps) {
  return (
    <View style={styles.row}>
      {GENDERS.map((gender) => (
        <Chip
          key={gender}
          label={labels[gender]}
          onPress={() => onToggle(gender)}
          selected={selected.includes(gender)}
        />
      ))}
    </View>
  );
}

const styles = StyleSheet.create({ row: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm } });
