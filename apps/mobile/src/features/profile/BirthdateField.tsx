import React, { useRef } from "react";
import { StyleSheet, TextInput, View } from "react-native";
import { radii, spacing, type Palette } from "../../theme/palette";
import { useStyles, useTheme } from "../../theme/ThemeProvider";
import { Text } from "../../ui";

export type BirthParts = { day: string; month: string; year: string };

export function splitBirthdate(iso?: string | null): BirthParts {
  if (!iso) return { day: "", month: "", year: "" };
  const [year, month, day] = iso.split("-");
  return { day, month, year };
}

/** Three numeric boxes (JJ / MM / AAAA): fast, accessible, no date picker. */
export function BirthdateField({ value, onChange, error }: { value: BirthParts; onChange: (v: BirthParts) => void; error?: string | null }) {
  const styles = useStyles(makeStyles);
  const { colors } = useTheme();
  const monthRef = useRef<TextInput>(null);
  const yearRef = useRef<TextInput>(null);
  const digits = (s: string) => s.replace(/\D/g, "");
  const box = (key: keyof BirthParts, placeholder: string, max: number, next?: React.RefObject<TextInput | null>, ref?: React.Ref<TextInput>) => (
    <TextInput
      ref={ref}
      accessibilityLabel={{ day: "Jour", month: "Mois", year: "Année" }[key]}
      inputMode="numeric"
      keyboardType="number-pad"
      maxLength={max}
      onChangeText={(t) => {
        const v = digits(t);
        onChange({ ...value, [key]: v });
        if (v.length === max) next?.current?.focus();
      }}
      placeholder={placeholder}
      placeholderTextColor={colors.textSubtle}
      style={[styles.box, key === "year" ? styles.year : null, error ? { borderColor: colors.danger } : null]}
      testID={`birth-${key}`}
      value={value[key]}
    />
  );
  return (
    <View style={styles.wrap}>
      <View style={styles.row}>
        {box("day", "JJ", 2, monthRef)}
        <Text tone="subtle">/</Text>
        {box("month", "MM", 2, yearRef, monthRef)}
        <Text tone="subtle">/</Text>
        {box("year", "AAAA", 4, undefined, yearRef)}
      </View>
      {error ? (
        <Text tone="danger" variant="caption">
          {error}
        </Text>
      ) : null}
    </View>
  );
}

const makeStyles = (c: Palette) =>
  StyleSheet.create({
    wrap: { gap: spacing.xs },
    row: { flexDirection: "row", alignItems: "center", gap: spacing.xs },
    box: {
      width: 64,
      height: 56,
      borderRadius: radii.md,
      borderWidth: 1,
      borderColor: c.border,
      backgroundColor: c.surface,
      textAlign: "center",
      fontSize: 20,
      fontWeight: "600",
      color: c.text
    },
    year: { width: 96 }
  });
