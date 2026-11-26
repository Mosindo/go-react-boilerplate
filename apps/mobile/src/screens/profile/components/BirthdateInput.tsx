import React, { useRef } from "react";
import { StyleSheet, TextInput, View } from "react-native";
import { radii, spacing, typography, useTheme } from "../../../theme";
import { Text } from "../../../ui";

export type BirthdateParts = { day: string; month: string; year: string };

export function partsFromIso(iso: string): BirthdateParts {
  const [year = "", month = "", day = ""] = iso ? iso.split("-") : [];
  return { day, month, year };
}

type Props = {
  value: BirthdateParts;
  onChange: (value: BirthdateParts) => void;
  error?: string | null;
  disabled?: boolean;
};

/** Three numeric fields (JJ / MM / AAAA): simple, accessible and identical on every platform. */
export function BirthdateInput({ value, onChange, error, disabled }: Props) {
  const { colors } = useTheme();
  const monthRef = useRef<TextInput>(null);
  const yearRef = useRef<TextInput>(null);
  const field = (
    key: keyof BirthdateParts,
    placeholder: string,
    maxLength: number,
    label: string,
    next?: React.RefObject<TextInput | null>,
    ref?: React.RefObject<TextInput | null>
  ) => (
    <TextInput
      accessibilityLabel={label}
      editable={!disabled}
      inputMode="numeric"
      keyboardType="number-pad"
      maxLength={maxLength}
      onChangeText={(text) => {
        const digits = text.replace(/\D/g, "");
        onChange({ ...value, [key]: digits });
        if (digits.length === maxLength) next?.current?.focus();
      }}
      placeholder={placeholder}
      placeholderTextColor={colors.textSubtle}
      ref={ref}
      style={[
        typography.heading,
        styles.input,
        {
          color: disabled ? colors.textMuted : colors.text,
          borderColor: error ? colors.danger : colors.border,
          backgroundColor: colors.surface
        },
        key === "year" ? styles.year : null
      ]}
      testID={`birthdate-${key}`}
      value={value[key]}
    />
  );
  return (
    <View style={styles.wrap}>
      <Text tone="muted" variant="label">
        Date de naissance
      </Text>
      <View style={styles.row}>
        {field("day", "JJ", 2, "Jour", monthRef)}
        {field("month", "MM", 2, "Mois", yearRef, monthRef)}
        {field("year", "AAAA", 4, "Année", undefined, yearRef)}
      </View>
      {error ? (
        <Text tone="danger" variant="caption">
          {error}
        </Text>
      ) : (
        <Text tone="subtle" variant="caption">
          {disabled ? "La date de naissance ne peut plus être modifiée." : "Seul votre âge sera affiché."}
        </Text>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { gap: 6 },
  row: { flexDirection: "row", gap: spacing.sm },
  input: { width: 72, height: 56, borderWidth: 1.5, borderRadius: radii.md, textAlign: "center", outlineStyle: "none" } as object,
  year: { width: 100 }
});
