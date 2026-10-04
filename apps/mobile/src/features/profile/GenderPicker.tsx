import React from "react";
import { StyleSheet, View } from "react-native";
import { GENDERS, type Gender } from "../../api/types";
import { spacing } from "../../theme";
import { Chip, Text } from "../../ui";
import { GENDER_LABELS } from "./labels";

type Props = { value: Gender | null; onChange: (g: Gender) => void; error?: string | null };

export function GenderPicker({ value, onChange, error }: Props) {
  return (
    <View style={styles.wrap}>
      <Text tone="muted" variant="small" weight="medium">
        I am
      </Text>
      <View style={styles.chips}>
        {GENDERS.map((gender) => (
          <Chip
            key={gender}
            label={GENDER_LABELS[gender]}
            onPress={() => onChange(gender)}
            selected={value === gender}
          />
        ))}
      </View>
      {error ? (
        <Text tone="danger" variant="small">
          {error}
        </Text>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { gap: 2 },
  chips: { flexDirection: "row", flexWrap: "wrap", columnGap: spacing.xs }
});
