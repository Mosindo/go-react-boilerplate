import React from "react";
import { StyleSheet, View } from "react-native";
import { strings } from "../../i18n/strings";
import { FormField, Input } from "../ui";
import { spacing } from "../ui/tokens";

export type BirthDateParts = { day: string; month: string; year: string };

type BirthDateInputProps = {
  value: BirthDateParts;
  onChange: (value: BirthDateParts) => void;
  error?: string | null;
};

function digitsOnly(value: string, max: number): string {
  return value.replace(/\D/g, "").slice(0, max);
}

export function BirthDateInput({ error, onChange, value }: BirthDateInputProps) {
  return (
    <FormField
      error={error}
      helperText={strings.onboarding.birthDateHelp}
      label={strings.onboarding.birthDateLabel}
    >
      <View style={styles.row}>
        <Input
          accessibilityLabel={strings.onboarding.day}
          containerStyle={styles.small}
          keyboardType="number-pad"
          maxLength={2}
          onChangeText={(day) => onChange({ ...value, day: digitsOnly(day, 2) })}
          placeholder={strings.onboarding.day}
          value={value.day}
        />
        <Input
          accessibilityLabel={strings.onboarding.month}
          containerStyle={styles.small}
          keyboardType="number-pad"
          maxLength={2}
          onChangeText={(month) => onChange({ ...value, month: digitsOnly(month, 2) })}
          placeholder={strings.onboarding.month}
          value={value.month}
        />
        <Input
          accessibilityLabel={strings.onboarding.year}
          containerStyle={styles.year}
          keyboardType="number-pad"
          maxLength={4}
          onChangeText={(year) => onChange({ ...value, year: digitsOnly(year, 4) })}
          placeholder={strings.onboarding.year}
          value={value.year}
        />
      </View>
    </FormField>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", gap: spacing.sm },
  small: { flex: 1 },
  year: { flex: 1.6 }
});
