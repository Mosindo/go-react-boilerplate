import React from "react";
import { StyleSheet, View } from "react-native";
import { audienceLabels, genderLabels, type Gender, type Interest } from "../api/types";
import { Chip, FormField, Input, Stepper, Text, spacing } from "../shared/ui";

const GENDERS: Gender[] = ["woman", "man", "nonbinary"];

export function BirthdateField({
  day,
  month,
  year,
  onChange,
  error
}: {
  day: string;
  month: string;
  year: string;
  onChange: (next: { day: string; month: string; year: string }) => void;
  error?: string | null;
}) {
  const digits = (v: string, max: number) => v.replace(/\D/g, "").slice(0, max);
  return (
    <FormField error={error} helperText="Seul ton âge sera visible." label="Date de naissance">
      <View style={styles.row}>
        <Input
          accessibilityLabel="Jour"
          containerStyle={styles.small}
          inputMode="numeric"
          keyboardType="number-pad"
          maxLength={2}
          onChangeText={(v) => onChange({ day: digits(v, 2), month, year })}
          placeholder="JJ"
          testID="birth-day"
          value={day}
        />
        <Input
          accessibilityLabel="Mois"
          containerStyle={styles.small}
          inputMode="numeric"
          keyboardType="number-pad"
          maxLength={2}
          onChangeText={(v) => onChange({ day, month: digits(v, 2), year })}
          placeholder="MM"
          testID="birth-month"
          value={month}
        />
        <Input
          accessibilityLabel="Année"
          containerStyle={styles.large}
          inputMode="numeric"
          keyboardType="number-pad"
          maxLength={4}
          onChangeText={(v) => onChange({ day, month, year: digits(v, 4) })}
          placeholder="AAAA"
          testID="birth-year"
          value={year}
        />
      </View>
    </FormField>
  );
}

export function GenderField({ value, onChange }: { value: Gender | ""; onChange: (g: Gender) => void }) {
  return (
    <FormField label="Je suis">
      <View style={styles.chips}>
        {GENDERS.map((g) => (
          <Chip key={g} label={genderLabels[g]} onPress={() => onChange(g)} selected={value === g} testID={`gender-${g}`} />
        ))}
      </View>
    </FormField>
  );
}

export function AudienceField({ value, onChange }: { value: Gender[]; onChange: (g: Gender[]) => void }) {
  const toggle = (g: Gender) => onChange(value.includes(g) ? value.filter((x) => x !== g) : [...value, g]);
  return (
    <FormField helperText="Plusieurs choix possibles." label="Je souhaite rencontrer">
      <View style={styles.chips}>
        {GENDERS.map((g) => (
          <Chip key={g} label={audienceLabels[g]} onPress={() => toggle(g)} selected={value.includes(g)} testID={`audience-${g}`} />
        ))}
      </View>
    </FormField>
  );
}

export function RangeFields({
  minAge,
  maxAge,
  distance,
  onChange
}: {
  minAge: number;
  maxAge: number;
  distance: number;
  onChange: (next: { minAge: number; maxAge: number; distance: number }) => void;
}) {
  return (
    <View style={styles.stack}>
      <Stepper label="Âge minimum" max={maxAge} min={18} onChange={(v) => onChange({ minAge: v, maxAge, distance })} value={minAge} />
      <Stepper label="Âge maximum" max={99} min={minAge} onChange={(v) => onChange({ minAge, maxAge: v, distance })} value={maxAge} />
      <Stepper
        format={(v) => `${v} km`}
        label="Distance maximum"
        max={300}
        min={5}
        onChange={(v) => onChange({ minAge, maxAge, distance: v })}
        step={5}
        value={distance}
      />
    </View>
  );
}

export function InterestsField({
  all,
  selected,
  max,
  onChange
}: {
  all: Interest[];
  selected: string[];
  max: number;
  onChange: (slugs: string[]) => void;
}) {
  const toggle = (slug: string) =>
    onChange(selected.includes(slug) ? selected.filter((s) => s !== slug) : selected.length < max ? [...selected, slug] : selected);
  return (
    <FormField label={`Centres d'intérêt (${selected.length}/${max})`}>
      <View style={styles.chips}>
        {all.map((i) => (
          <Chip
            disabled={!selected.includes(i.slug) && selected.length >= max}
            key={i.slug}
            label={i.label}
            onPress={() => toggle(i.slug)}
            selected={selected.includes(i.slug)}
            testID={`interest-${i.slug}`}
          />
        ))}
      </View>
      {all.length === 0 ? <Text tone="subtle">Chargement…</Text> : null}
    </FormField>
  );
}

const styles = StyleSheet.create({
  row: {
    flexDirection: "row",
    gap: spacing.sm
  },
  small: {
    flex: 1
  },
  large: {
    flex: 1.6
  },
  chips: {
    flexDirection: "row",
    flexWrap: "wrap",
    gap: spacing.sm
  },
  stack: {
    gap: spacing.sm
  }
});
