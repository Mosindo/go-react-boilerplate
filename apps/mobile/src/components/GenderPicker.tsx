import { StyleSheet, View } from "react-native";
import type { Gender } from "../api/types";
import { GENDERS, GENDER_LABELS, INTERESTED_IN_LABELS } from "../lib/labels";
import { Chip } from "../shared/ui";
import { spacing } from "../theme";

export function GenderPicker({ value, onChange }: { value: Gender | null; onChange: (value: Gender) => void }) {
  return (
    <View style={styles.row} accessibilityRole="radiogroup">
      {GENDERS.map((gender) => (
        <Chip key={gender} label={GENDER_LABELS[gender]} selected={value === gender} onPress={() => onChange(gender)} />
      ))}
    </View>
  );
}

export function InterestedInPicker({ value, onChange }: { value: Gender[]; onChange: (value: Gender[]) => void }) {
  const toggle = (gender: Gender) => {
    if (value.includes(gender)) {
      if (value.length > 1) onChange(value.filter((g) => g !== gender)); // at least one choice must remain
    } else {
      onChange([...value, gender]);
    }
  };
  return (
    <View style={styles.row}>
      {GENDERS.map((gender) => (
        <Chip key={gender} label={INTERESTED_IN_LABELS[gender]} selected={value.includes(gender)} onPress={() => toggle(gender)} />
      ))}
    </View>
  );
}

const styles = StyleSheet.create({ row: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm } });
