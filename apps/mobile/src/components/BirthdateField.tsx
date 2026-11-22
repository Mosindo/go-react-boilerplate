import React, { useRef } from "react";
import { StyleSheet, TextInput, View } from "react-native";
import { Text, TextField, spacing } from "../ui";

export type BirthdateParts = { day: string; month: string; year: string };

type Props = {
  value: BirthdateParts;
  onChange: (next: BirthdateParts) => void;
  error?: string | null;
};

const digits = (v: string, max: number) => v.replace(/\D/g, "").slice(0, max);

/** Three numeric fields: works everywhere without a native date picker. */
export function BirthdateField({ value, onChange, error }: Props) {
  const monthRef = useRef<TextInput>(null);
  const yearRef = useRef<TextInput>(null);
  return (
    <View style={styles.wrap}>
      <View style={styles.row}>
        <View style={styles.small}>
          <TextField
            inputMode="numeric"
            keyboardType="number-pad"
            label="Jour"
            maxLength={2}
            onChangeText={(t) => {
              const day = digits(t, 2);
              onChange({ ...value, day });
              if (day.length === 2) {
                monthRef.current?.focus();
              }
            }}
            placeholder="JJ"
            testID="birth-day"
            value={value.day}
          />
        </View>
        <View style={styles.small}>
          <TextField
            inputMode="numeric"
            keyboardType="number-pad"
            label="Mois"
            maxLength={2}
            onChangeText={(t) => {
              const month = digits(t, 2);
              onChange({ ...value, month });
              if (month.length === 2) {
                yearRef.current?.focus();
              }
            }}
            placeholder="MM"
            ref={monthRef}
            testID="birth-month"
            value={value.month}
          />
        </View>
        <View style={styles.large}>
          <TextField
            inputMode="numeric"
            keyboardType="number-pad"
            label="Année"
            maxLength={4}
            onChangeText={(t) => onChange({ ...value, year: digits(t, 4) })}
            placeholder="AAAA"
            ref={yearRef}
            testID="birth-year"
            value={value.year}
          />
        </View>
      </View>
      {error ? (
        <Text accessibilityLiveRegion="polite" tone="danger" variant="caption">
          {error}
        </Text>
      ) : (
        <Text tone="subtle" variant="caption">
          Seul votre âge sera affiché. Il ne pourra plus être modifié ensuite.
        </Text>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: {
    gap: spacing.xs
  },
  row: {
    flexDirection: "row",
    gap: spacing.sm
  },
  small: {
    flex: 1
  },
  large: {
    flex: 1.6
  }
});
