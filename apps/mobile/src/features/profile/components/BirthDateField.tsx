import React, { useRef } from "react";
import { StyleSheet, type TextInput, View } from "react-native";
import { Text, TextField } from "../../../shared/ui";
import { spacing } from "../../../theme";

export type BirthDateValue = { day: string; month: string; year: string };

type Props = { value: BirthDateValue; onChange: (value: BirthDateValue) => void; error?: string | null };

const digitsOnly = (text: string) => text.replace(/\D/g, "");

export function BirthDateField({ value, onChange, error }: Props) {
  const monthRef = useRef<TextInput>(null);
  const yearRef = useRef<TextInput>(null);

  return (
    <View style={styles.wrapper}>
      <View style={styles.row}>
        <View style={styles.small}>
          <TextField
            keyboardType="number-pad"
            label="Jour"
            maxLength={2}
            onChangeText={(text) => {
              const day = digitsOnly(text);
              onChange({ ...value, day });
              if (day.length === 2) monthRef.current?.focus();
            }}
            placeholder="JJ"
            testID="birth-day-input"
            value={value.day}
          />
        </View>
        <View style={styles.small}>
          <TextField
            keyboardType="number-pad"
            label="Mois"
            maxLength={2}
            onChangeText={(text) => {
              const month = digitsOnly(text);
              onChange({ ...value, month });
              if (month.length === 2) yearRef.current?.focus();
            }}
            placeholder="MM"
            ref={monthRef}
            testID="birth-month-input"
            value={value.month}
          />
        </View>
        <View style={styles.large}>
          <TextField
            keyboardType="number-pad"
            label="Année"
            maxLength={4}
            onChangeText={(text) => onChange({ ...value, year: digitsOnly(text) })}
            placeholder="AAAA"
            ref={yearRef}
            testID="birth-year-input"
            value={value.year}
          />
        </View>
      </View>
      {error ? (
        <Text accessibilityLiveRegion="polite" color="danger" variant="caption">
          {error}
        </Text>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  wrapper: { gap: spacing.xs },
  row: { flexDirection: "row", gap: spacing.md },
  small: { flex: 1 },
  large: { flex: 1.6 }
});
