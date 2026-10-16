import React, { useRef } from "react";
import { StyleSheet, TextInput, View } from "react-native";
import type { DateParts } from "../lib/validation";
import { FormField, Text, colors, controls, radii, spacing, typography } from "../shared/ui";

type DateInputProps = {
  value: DateParts;
  onChange: (value: DateParts) => void;
  error?: string | null;
  disabled?: boolean;
};

export function DateInput({ value, onChange, error, disabled }: DateInputProps) {
  const monthRef = useRef<TextInput>(null);
  const yearRef = useRef<TextInput>(null);

  const field = (
    key: keyof DateParts,
    placeholder: string,
    maxLength: number,
    width: number,
    next?: React.RefObject<TextInput | null>,
    ref?: React.RefObject<TextInput | null>
  ) => (
    <TextInput
      accessibilityLabel={placeholder}
      editable={!disabled}
      keyboardType="number-pad"
      maxLength={maxLength}
      onChangeText={(text) => {
        const digits = text.replace(/\D/g, "");
        onChange({ ...value, [key]: digits });
        if (next && digits.length === maxLength) {
          next.current?.focus();
        }
      }}
      placeholder={placeholder}
      placeholderTextColor={colors.textSubtle}
      ref={ref}
      style={[styles.input, { width }, error ? styles.error : null]}
      testID={`date-${key}`}
      value={value[key]}
    />
  );

  return (
    <FormField error={error} label="Date de naissance" helperText="Votre âge sera affiché, pas votre date de naissance.">
      <View style={styles.row}>
        {field("day", "JJ", 2, 64, monthRef)}
        <Text tone="muted">/</Text>
        {field("month", "MM", 2, 64, yearRef, monthRef)}
        <Text tone="muted">/</Text>
        {field("year", "AAAA", 4, 96, undefined, yearRef)}
      </View>
    </FormField>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center", gap: spacing.sm },
  input: {
    minHeight: controls.input.md,
    borderWidth: 1,
    borderColor: colors.border,
    borderRadius: radii.lg,
    backgroundColor: colors.backgroundElevated,
    color: colors.text,
    textAlign: "center",
    ...typography.body
  },
  error: { borderColor: colors.danger }
});
