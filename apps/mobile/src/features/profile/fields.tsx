import React, { useRef } from "react";
import { StyleSheet, TextInput, View } from "react-native";
import type { Gender, Interest } from "../../api/types";
import { genderLabels, genderPlural } from "../../lib/format";
import { MAX_AGE, MIN_AGE } from "../../lib/validation";
import { radii, spacing, type, useTheme } from "../../theme";
import { Chip, ChipGroup, Stepper, Text } from "../../ui";

const genders: Gender[] = ["woman", "man", "nonbinary"];

export function GenderPicker({ value, onChange }: { value: Gender | ""; onChange: (g: Gender) => void }) {
  return (
    <ChipGroup>
      {genders.map((g) => (
        <Chip key={g} label={genderLabels[g]} selected={value === g} onPress={() => onChange(g)} testID={`gender-${g}`} />
      ))}
    </ChipGroup>
  );
}

export function InterestedInPicker({ value, onChange }: { value: Gender[]; onChange: (g: Gender[]) => void }) {
  const toggle = (g: Gender) => {
    const next = value.includes(g) ? value.filter((x) => x !== g) : [...value, g];
    if (next.length > 0) onChange(next);
  };
  return (
    <ChipGroup>
      {genders.map((g) => (
        <Chip key={g} label={genderPlural[g]} selected={value.includes(g)} onPress={() => toggle(g)} testID={`seek-${g}`} />
      ))}
    </ChipGroup>
  );
}

export function InterestPicker({
  all,
  selected,
  onChange,
  max = 10
}: {
  all: Interest[];
  selected: number[];
  onChange: (ids: number[]) => void;
  max?: number;
}) {
  const toggle = (id: number) => {
    if (selected.includes(id)) onChange(selected.filter((x) => x !== id));
    else if (selected.length < max) onChange([...selected, id]);
  };
  return (
    <View style={{ gap: spacing.sm }}>
      <Text variant="caption" color="textMuted">
        {selected.length}/{max} sélectionnés
      </Text>
      <ChipGroup>
        {all.map((i) => (
          <Chip key={i.id} label={i.label} selected={selected.includes(i.id)} onPress={() => toggle(i.id)} testID={`interest-${i.slug}`} />
        ))}
      </ChipGroup>
    </View>
  );
}

export type BirthdateParts = { day: string; month: string; year: string };

export function BirthdateField({
  value,
  onChange,
  error
}: {
  value: BirthdateParts;
  onChange: (v: BirthdateParts) => void;
  error?: string | null;
}) {
  const { colors } = useTheme();
  const monthRef = useRef<TextInput>(null);
  const yearRef = useRef<TextInput>(null);
  const box = [styles.box, { backgroundColor: colors.surfaceAlt, borderColor: error ? colors.danger : colors.border, color: colors.text }];
  return (
    <View style={{ gap: spacing.xs }}>
      <View style={styles.row}>
        <TextInput
          style={[type.heading, ...box, styles.small]}
          value={value.day}
          placeholder="JJ"
          placeholderTextColor={colors.textSubtle}
          keyboardType="number-pad"
          maxLength={2}
          accessibilityLabel="Jour de naissance"
          testID="birth-day"
          onChangeText={(day) => {
            onChange({ ...value, day: day.replace(/\D/g, "") });
            if (day.length === 2) monthRef.current?.focus();
          }}
        />
        <TextInput
          ref={monthRef}
          style={[type.heading, ...box, styles.small]}
          value={value.month}
          placeholder="MM"
          placeholderTextColor={colors.textSubtle}
          keyboardType="number-pad"
          maxLength={2}
          accessibilityLabel="Mois de naissance"
          testID="birth-month"
          onChangeText={(month) => {
            onChange({ ...value, month: month.replace(/\D/g, "") });
            if (month.length === 2) yearRef.current?.focus();
          }}
        />
        <TextInput
          ref={yearRef}
          style={[type.heading, ...box, styles.large]}
          value={value.year}
          placeholder="AAAA"
          placeholderTextColor={colors.textSubtle}
          keyboardType="number-pad"
          maxLength={4}
          accessibilityLabel="Année de naissance"
          testID="birth-year"
          onChangeText={(year) => onChange({ ...value, year: year.replace(/\D/g, "") })}
        />
      </View>
      {error ? (
        <Text variant="caption" color="danger">
          {error}
        </Text>
      ) : null}
    </View>
  );
}

const distanceOptions = [5, 10, 25, 50, 100, 200];

export function AgeRangeField({ min, max, onChange }: { min: number; max: number; onChange: (min: number, max: number) => void }) {
  return (
    <View>
      <Stepper label="Âge minimum" value={min} min={MIN_AGE} max={max} onChange={(v) => onChange(v, max)} suffix=" ans" />
      <Stepper label="Âge maximum" value={max} min={min} max={MAX_AGE} onChange={(v) => onChange(min, v)} suffix=" ans" />
    </View>
  );
}

export function DistanceField({ value, onChange }: { value: number; onChange: (v: number) => void }) {
  return (
    <ChipGroup>
      {distanceOptions.map((d) => (
        <Chip key={d} label={`${d} km`} selected={value === d} onPress={() => onChange(d)} />
      ))}
    </ChipGroup>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", gap: spacing.xs },
  box: { borderWidth: 1.5, borderRadius: radii.md, paddingVertical: 14, textAlign: "center" },
  small: { width: 72 },
  large: { width: 110 }
});
