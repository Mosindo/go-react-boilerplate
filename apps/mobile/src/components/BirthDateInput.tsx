import React from "react";
import { StyleSheet, View } from "react-native";
import { spacing } from "../theme";
import { FormField, Input } from "../shared/ui";

export type BirthDateParts = { day: string; month: string; year: string };

export function BirthDateInput({
  value,
  onChange,
  error
}: {
  value: BirthDateParts;
  onChange: (next: BirthDateParts) => void;
  error?: string | null;
}) {
  return (
    <FormField error={error} label="Birth date">
      <View style={styles.row}>
        <View style={styles.small}>
          <Input
            keyboardType="number-pad"
            label="Day"
            maxLength={2}
            onChangeText={(day) => onChange({ ...value, day })}
            placeholder="DD"
            value={value.day}
          />
        </View>
        <View style={styles.small}>
          <Input
            keyboardType="number-pad"
            label="Month"
            maxLength={2}
            onChangeText={(month) => onChange({ ...value, month })}
            placeholder="MM"
            value={value.month}
          />
        </View>
        <View style={styles.year}>
          <Input
            keyboardType="number-pad"
            label="Year"
            maxLength={4}
            onChangeText={(year) => onChange({ ...value, year })}
            placeholder="YYYY"
            value={value.year}
          />
        </View>
      </View>
    </FormField>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", gap: spacing.md },
  small: { flex: 1 },
  year: { flex: 1.6 }
});
