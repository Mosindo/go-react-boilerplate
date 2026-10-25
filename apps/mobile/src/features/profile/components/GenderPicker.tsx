import React from "react";
import { StyleSheet, View } from "react-native";
import type { Gender } from "../../../api";
import { Chip, Text } from "../../../shared/ui";
import { GENDERS, GENDER_LABELS } from "../../../shared/labels";
import { spacing } from "../../../theme";

type Props = { value: Gender | null; onChange: (gender: Gender) => void; error?: string | null };

export function GenderPicker({ value, onChange, error }: Props) {
  return (
    <View style={styles.wrapper}>
      <Text color="textMuted" variant="caption">
        Je suis
      </Text>
      <View accessibilityRole="radiogroup" style={styles.row}>
        {GENDERS.map((gender) => (
          <Chip
            key={gender}
            label={GENDER_LABELS[gender]}
            onPress={() => onChange(gender)}
            selected={value === gender}
            testID={`gender-${gender}`}
          />
        ))}
      </View>
      {error ? (
        <Text color="danger" variant="caption">
          {error}
        </Text>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  wrapper: { gap: spacing.sm },
  row: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }
});
