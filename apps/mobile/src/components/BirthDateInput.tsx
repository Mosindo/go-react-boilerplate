import React, { useRef } from "react";
import { View, type TextInput } from "react-native";
import type { BirthDateParts } from "../lib/dates";
import { FormField, Text, spacing } from "../shared/ui";

type BirthDateInputProps = {
  value: BirthDateParts;
  onChange: (value: BirthDateParts) => void;
  error?: string | null;
  disabled?: boolean;
};

const digits = (text: string) => text.replace(/\D/g, "");

/** Three numeric fields (DD / MM / YYYY): reliable on every keyboard, no date-picker dependency. */
export function BirthDateInput({ disabled = false, error, onChange, value }: BirthDateInputProps) {
  const monthRef = useRef<TextInput>(null);
  const yearRef = useRef<TextInput>(null);

  return (
    <View style={{ gap: spacing.xs }}>
      <View style={{ flexDirection: "row", gap: spacing.sm }}>
        <View style={{ flex: 1 }}>
          <FormField
            editable={!disabled}
            keyboardType="number-pad"
            label="Jour"
            maxLength={2}
            onChangeText={(text) => {
              const day = digits(text);
              onChange({ ...value, day });
              if (day.length === 2) monthRef.current?.focus();
            }}
            placeholder="JJ"
            testID="birth-day"
            value={value.day}
          />
        </View>
        <View style={{ flex: 1 }}>
          <FormField
            editable={!disabled}
            keyboardType="number-pad"
            label="Mois"
            maxLength={2}
            onChangeText={(text) => {
              const month = digits(text);
              onChange({ ...value, month });
              if (month.length === 2) yearRef.current?.focus();
            }}
            placeholder="MM"
            ref={monthRef}
            testID="birth-month"
            value={value.month}
          />
        </View>
        <View style={{ flex: 1.4 }}>
          <FormField
            editable={!disabled}
            keyboardType="number-pad"
            label="Année"
            maxLength={4}
            onChangeText={(text) => onChange({ ...value, year: digits(text) })}
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
      ) : null}
    </View>
  );
}
