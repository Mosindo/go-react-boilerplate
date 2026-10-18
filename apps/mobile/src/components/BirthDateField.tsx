import React, { useState } from "react";
import { StyleSheet, TextInput, View } from "react-native";
import { FormField, colors, controls, radii, spacing, typography } from "../shared/ui";

type BirthDateFieldProps = {
  /** ISO date (YYYY-MM-DD) or an empty string while incomplete/invalid. */
  value: string;
  onChange: (iso: string) => void;
  error?: string | null;
};

function split(iso: string): { day: string; month: string; year: string } {
  const [year = "", month = "", day = ""] = iso.split("-");
  return { day, month, year };
}

export function toIsoDate(day: string, month: string, year: string): string {
  if (!/^\d{1,2}$/.test(day) || !/^\d{1,2}$/.test(month) || !/^\d{4}$/.test(year)) {
    return "";
  }
  const d = Number(day);
  const m = Number(month);
  const y = Number(year);
  const date = new Date(Date.UTC(y, m - 1, d));
  if (date.getUTCFullYear() !== y || date.getUTCMonth() !== m - 1 || date.getUTCDate() !== d) {
    return "";
  }
  return `${year}-${String(m).padStart(2, "0")}-${String(d).padStart(2, "0")}`;
}

/** Whole years between an ISO birth date and today (local time). */
export function ageFromIso(iso: string, now = new Date()): number | null {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(iso)) {
    return null;
  }
  const [y, m, d] = iso.split("-").map(Number);
  let age = now.getFullYear() - y;
  if (now.getMonth() + 1 < m || (now.getMonth() + 1 === m && now.getDate() < d)) {
    age -= 1;
  }
  return age;
}

export function BirthDateField({ value, onChange, error }: BirthDateFieldProps) {
  const initial = split(value);
  const [day, setDay] = useState(initial.day.replace(/^0/, ""));
  const [month, setMonth] = useState(initial.month.replace(/^0/, ""));
  const [year, setYear] = useState(initial.year);

  const update = (nextDay: string, nextMonth: string, nextYear: string) => {
    setDay(nextDay);
    setMonth(nextMonth);
    setYear(nextYear);
    onChange(toIsoDate(nextDay, nextMonth, nextYear));
  };
  const digits = (text: string, max: number) => text.replace(/\D/g, "").slice(0, max);

  return (
    <FormField
      error={error}
      label="Date de naissance"
      helperText="Vous devez avoir 18 ans ou plus. Seul votre âge sera visible."
    >
      <View style={styles.row}>
        <TextInput
          accessibilityLabel="Jour de naissance"
          inputMode="numeric"
          maxLength={2}
          onChangeText={(t) => update(digits(t, 2), month, year)}
          placeholder="JJ"
          placeholderTextColor={colors.textSubtle}
          style={[styles.input, styles.small, error ? styles.error : null]}
          testID="birth-day"
          value={day}
        />
        <TextInput
          accessibilityLabel="Mois de naissance"
          inputMode="numeric"
          maxLength={2}
          onChangeText={(t) => update(day, digits(t, 2), year)}
          placeholder="MM"
          placeholderTextColor={colors.textSubtle}
          style={[styles.input, styles.small, error ? styles.error : null]}
          testID="birth-month"
          value={month}
        />
        <TextInput
          accessibilityLabel="Année de naissance"
          inputMode="numeric"
          maxLength={4}
          onChangeText={(t) => update(day, month, digits(t, 4))}
          placeholder="AAAA"
          placeholderTextColor={colors.textSubtle}
          style={[styles.input, styles.large, error ? styles.error : null]}
          testID="birth-year"
          value={year}
        />
      </View>
    </FormField>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", gap: spacing.sm },
  input: {
    borderWidth: 1,
    borderColor: colors.border,
    borderRadius: radii.lg,
    backgroundColor: colors.backgroundElevated,
    color: colors.text,
    minHeight: controls.input.md,
    paddingHorizontal: spacing.lg,
    textAlign: "center",
    ...typography.body
  },
  small: { flex: 1 },
  large: { flex: 1.6 },
  error: { borderColor: colors.danger }
});
