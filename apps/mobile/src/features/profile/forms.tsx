import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { useQuery } from "@tanstack/react-query";
import { profileApi } from "../../api/endpoints";
import { GENDER_LABELS, type Gender, type Preferences } from "../../api/types";
import { queryKeys } from "../../realtime/cache";
import { radius, spacing, useTheme } from "../../theme/theme";
import { Chip, FormError, Icon, Text, TextField } from "../../ui";

const GENDERS: Gender[] = ["woman", "man", "nonbinary"];

export function GenderPicker({
  value,
  onChange,
  testIDPrefix,
}: {
  value: Gender | null;
  onChange: (g: Gender) => void;
  testIDPrefix: string;
}) {
  return (
    <View style={styles.row}>
      {GENDERS.map((g) => (
        <Chip
          key={g}
          testID={`${testIDPrefix}-${g}`}
          label={GENDER_LABELS[g]}
          selected={value === g}
          onPress={() => onChange(g)}
        />
      ))}
    </View>
  );
}

export function MultiGenderPicker({
  value,
  onChange,
  testIDPrefix,
}: {
  value: Gender[];
  onChange: (g: Gender[]) => void;
  testIDPrefix: string;
}) {
  const toggle = (g: Gender) =>
    onChange(value.includes(g) ? value.filter((v) => v !== g) : [...value, g]);
  return (
    <View style={styles.row}>
      {GENDERS.map((g) => (
        <Chip
          key={g}
          testID={`${testIDPrefix}-${g}`}
          label={GENDER_LABELS[g]}
          selected={value.includes(g)}
          onPress={() => toggle(g)}
        />
      ))}
    </View>
  );
}

export const MAX_INTERESTS = 10;

export function InterestPicker({
  value,
  onChange,
}: {
  value: string[];
  onChange: (slugs: string[]) => void;
}) {
  const { data } = useQuery({
    queryKey: queryKeys.interests,
    queryFn: profileApi.interests,
    staleTime: 3_600_000,
  });
  const toggle = (slug: string) => {
    if (value.includes(slug)) onChange(value.filter((s) => s !== slug));
    else if (value.length < MAX_INTERESTS) onChange([...value, slug]);
  };
  return (
    <View style={{ gap: spacing.sm }}>
      <Text variant="label" muted>
        Centres d&apos;intérêt ({value.length}/{MAX_INTERESTS})
      </Text>
      <View style={styles.row}>
        {(data ?? []).map((i) => (
          <Chip
            key={i.slug}
            testID={`interest-${i.slug}`}
            label={i.label}
            selected={value.includes(i.slug)}
            onPress={() => toggle(i.slug)}
          />
        ))}
      </View>
    </View>
  );
}

export function BirthDateFields({
  day,
  month,
  year,
  onChange,
  editable = true,
  error,
}: {
  day: string;
  month: string;
  year: string;
  onChange: (part: "day" | "month" | "year", value: string) => void;
  editable?: boolean;
  error?: string | null;
}) {
  const digits = (v: string, max: number) => v.replace(/\D/g, "").slice(0, max);
  return (
    <View style={{ gap: spacing.xs }}>
      <Text variant="label" muted>
        Date de naissance
      </Text>
      <View style={styles.dateRow}>
        <TextField
          testID="birth-day"
          label="Jour"
          placeholder="jj"
          keyboardType="number-pad"
          value={day}
          editable={editable}
          onChangeText={(v) => onChange("day", digits(v, 2))}
          style={styles.dateInput}
        />
        <TextField
          testID="birth-month"
          label="Mois"
          placeholder="mm"
          keyboardType="number-pad"
          value={month}
          editable={editable}
          onChangeText={(v) => onChange("month", digits(v, 2))}
          style={styles.dateInput}
        />
        <TextField
          testID="birth-year"
          label="Année"
          placeholder="aaaa"
          keyboardType="number-pad"
          value={year}
          editable={editable}
          onChangeText={(v) => onChange("year", digits(v, 4))}
          style={styles.dateInputWide}
        />
      </View>
      <FormError testID="birth-error" message={error} />
    </View>
  );
}

function Stepper({
  label,
  value,
  min,
  max,
  onChange,
  testID,
}: {
  label: string;
  value: number;
  min: number;
  max: number;
  onChange: (v: number) => void;
  testID: string;
}) {
  const { colors } = useTheme();
  const btn = (icon: "remove" | "add", delta: number, disabled: boolean, name: string) => (
    <Pressable
      testID={`${testID}-${name}`}
      accessibilityRole="button"
      accessibilityLabel={`${name === "minus" ? "Diminuer" : "Augmenter"} ${label}`}
      disabled={disabled}
      onPress={() => onChange(Math.min(max, Math.max(min, value + delta)))}
      style={[
        styles.stepBtn,
        {
          borderColor: colors.border,
          backgroundColor: colors.surface,
          opacity: disabled ? 0.4 : 1,
        },
      ]}
    >
      <Icon name={icon} size={20} />
    </Pressable>
  );
  return (
    <View style={styles.stepper}>
      <Text variant="label" muted>
        {label}
      </Text>
      <View style={styles.stepControls}>
        {btn("remove", -1, value <= min, "minus")}
        <Text
          variant="heading"
          testID={`${testID}-value`}
          style={{ minWidth: 40, textAlign: "center" }}
        >
          {value}
        </Text>
        {btn("add", 1, value >= max, "plus")}
      </View>
    </View>
  );
}

const DISTANCES: { label: string; value: number | null }[] = [
  { label: "10 km", value: 10 },
  { label: "25 km", value: 25 },
  { label: "50 km", value: 50 },
  { label: "100 km", value: 100 },
  { label: "250 km", value: 250 },
  { label: "Partout", value: null },
];

export function PreferencesForm({
  value,
  onChange,
}: {
  value: Preferences;
  onChange: (p: Preferences) => void;
}) {
  return (
    <View style={{ gap: spacing.lg }}>
      <View style={{ gap: spacing.sm }}>
        <Text variant="label" muted>
          Je souhaite rencontrer
        </Text>
        <MultiGenderPicker
          value={value.interestedIn}
          onChange={(interestedIn) => onChange({ ...value, interestedIn })}
          testIDPrefix="pref-gender"
        />
      </View>
      <View style={styles.dateRow}>
        <Stepper
          testID="pref-age-min"
          label="Âge minimum"
          value={value.ageMin}
          min={18}
          max={value.ageMax}
          onChange={(ageMin) => onChange({ ...value, ageMin })}
        />
        <Stepper
          testID="pref-age-max"
          label="Âge maximum"
          value={value.ageMax}
          min={value.ageMin}
          max={99}
          onChange={(ageMax) => onChange({ ...value, ageMax })}
        />
      </View>
      <View style={{ gap: spacing.sm }}>
        <Text variant="label" muted>
          Distance maximale
        </Text>
        <View style={styles.row}>
          {DISTANCES.map((d) => (
            <Chip
              key={d.label}
              testID={`pref-distance-${d.value ?? "any"}`}
              label={d.label}
              selected={value.maxDistanceKm === d.value}
              onPress={() => onChange({ ...value, maxDistanceKm: d.value })}
            />
          ))}
        </View>
        <Text variant="caption" muted>
          La distance ne s&apos;applique que si vous partagez votre position approximative.
        </Text>
      </View>
    </View>
  );
}

export const DEFAULT_PREFERENCES: Preferences = {
  interestedIn: [],
  ageMin: 18,
  ageMax: 45,
  maxDistanceKm: 50,
};

const styles = StyleSheet.create({
  row: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  dateRow: { flexDirection: "row", gap: spacing.md, alignItems: "flex-start" },
  dateInput: { width: 84 },
  dateInputWide: { width: 120 },
  stepper: { flex: 1, gap: spacing.sm },
  stepControls: { flexDirection: "row", alignItems: "center", gap: spacing.sm },
  stepBtn: {
    width: 44,
    height: 44,
    borderRadius: radius.pill,
    borderWidth: 1,
    alignItems: "center",
    justifyContent: "center",
  },
});
