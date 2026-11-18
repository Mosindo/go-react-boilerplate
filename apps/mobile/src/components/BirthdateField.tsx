import React, { useRef } from "react";
import { StyleSheet, TextInput, View } from "react-native";
import { spacing } from "../theme";
import { Text, TextField } from "../ui";

export type BirthdateParts = { day: string; month: string; year: string };

type Props = {
  value: BirthdateParts;
  onChange: (value: BirthdateParts) => void;
  error?: string | null;
};

/** Three numeric inputs: accessible, keyboard friendly, no native picker needed. */
export function BirthdateField({ value, onChange, error }: Props) {
  const monthRef = useRef<TextInput>(null);
  const yearRef = useRef<TextInput>(null);
  const digits = (v: string) => v.replace(/\D/g, "");
  return (
    <View style={styles.wrap}>
      <View style={styles.row}>
        <View style={styles.small}>
          <TextField
            inputMode="numeric"
            keyboardType="number-pad"
            label="Jour"
            maxLength={2}
            onChangeText={(v) => {
              onChange({ ...value, day: digits(v) });
              if (digits(v).length === 2) {
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
            ref={monthRef}
            inputMode="numeric"
            keyboardType="number-pad"
            label="Mois"
            maxLength={2}
            onChangeText={(v) => {
              onChange({ ...value, month: digits(v) });
              if (digits(v).length === 2) {
                yearRef.current?.focus();
              }
            }}
            placeholder="MM"
            testID="birth-month"
            value={value.month}
          />
        </View>
        <View style={styles.large}>
          <TextField
            ref={yearRef}
            inputMode="numeric"
            keyboardType="number-pad"
            label="Année"
            maxLength={4}
            onChangeText={(v) => onChange({ ...value, year: digits(v) })}
            placeholder="AAAA"
            testID="birth-year"
            value={value.year}
          />
        </View>
      </View>
      {error ? (
        <Text color="danger" variant="caption">
          {error}
        </Text>
      ) : (
        <Text color="textSubtle" variant="caption">
          Seul votre âge sera visible. Elle ne pourra plus être modifiée.
        </Text>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { gap: spacing.xs },
  row: { flexDirection: "row", gap: spacing.sm },
  small: { flex: 1 },
  large: { flex: 1.6 }
});
