import React from "react";
import { StyleSheet, View } from "react-native";
import { GENDERS, type Gender, type Interest, type Preferences } from "../../api/types";
import { MAX_AGE, MAX_BIO_LENGTH, MAX_INTERESTS, MIN_AGE } from "../../lib/validation";
import { spacing } from "../../theme";
import { Button, Chip, Input, Text } from "../../ui";

export function GenderPicker({
  label,
  value,
  onChange
}: {
  label: string;
  value: Gender | null;
  onChange: (gender: Gender) => void;
}) {
  return (
    <View style={styles.block}>
      <Text variant="label" style={styles.heading}>
        {label}
      </Text>
      <View style={styles.row}>
        {GENDERS.map((g) => (
          <Chip
            key={g.value}
            label={g.label}
            selected={value === g.value}
            onPress={() => onChange(g.value)}
            testID={`gender-${g.value}`}
          />
        ))}
      </View>
    </View>
  );
}

export function toggleValue<T>(list: T[], value: T): T[] {
  return list.includes(value) ? list.filter((v) => v !== value) : [...list, value];
}

export function InterestedInPicker({
  value,
  onChange
}: {
  value: Gender[];
  onChange: (v: Gender[]) => void;
}) {
  return (
    <View style={styles.block}>
      <Text variant="label" style={styles.heading}>
        I would like to meet
      </Text>
      <View style={styles.row}>
        {GENDERS.map((g) => (
          <Chip
            key={g.value}
            label={g.label}
            selected={value.includes(g.value)}
            onPress={() => onChange(toggleValue(value, g.value))}
            testID={`interested-${g.value}`}
          />
        ))}
      </View>
    </View>
  );
}

function Stepper({
  label,
  value,
  min,
  max,
  step = 1,
  onChange
}: {
  label: string;
  value: number;
  min: number;
  max: number;
  step?: number;
  onChange: (v: number) => void;
}) {
  return (
    <View style={styles.stepper}>
      <Text style={styles.flex}>{label}</Text>
      <Button
        label="−"
        variant="secondary"
        accessibilityLabel={`Decrease ${label}`}
        onPress={() => onChange(Math.max(min, value - step))}
        style={styles.stepBtn}
      />
      <Text variant="label" style={styles.stepValue} accessibilityLabel={`${label}: ${value}`}>
        {value}
      </Text>
      <Button
        label="+"
        variant="secondary"
        accessibilityLabel={`Increase ${label}`}
        onPress={() => onChange(Math.min(max, value + step))}
        style={styles.stepBtn}
      />
    </View>
  );
}

export const DISTANCE_STEPS = [5, 10, 25, 50, 100, 200];

export function nextDistance(current: number, direction: 1 | -1): number {
  const index = DISTANCE_STEPS.findIndex((d) => d >= current);
  const base = index === -1 ? DISTANCE_STEPS.length - 1 : index;
  return DISTANCE_STEPS[Math.min(DISTANCE_STEPS.length - 1, Math.max(0, base + direction))];
}

export function RangeFields({ value, onChange }: { value: Preferences; onChange: (p: Preferences) => void }) {
  return (
    <View style={styles.block}>
      <Stepper
        label="Minimum age"
        value={value.minAge}
        min={MIN_AGE}
        max={value.maxAge}
        onChange={(minAge) => onChange({ ...value, minAge })}
      />
      <Stepper
        label="Maximum age"
        value={value.maxAge}
        min={value.minAge}
        max={MAX_AGE}
        onChange={(maxAge) => onChange({ ...value, maxAge })}
      />
      <View style={styles.stepper}>
        <Text style={styles.flex}>Distance (km)</Text>
        <Button
          label="−"
          variant="secondary"
          accessibilityLabel="Decrease distance"
          onPress={() => onChange({ ...value, maxDistanceKm: nextDistance(value.maxDistanceKm, -1) })}
          style={styles.stepBtn}
        />
        <Text variant="label" style={styles.stepValue}>
          {value.maxDistanceKm}
        </Text>
        <Button
          label="+"
          variant="secondary"
          accessibilityLabel="Increase distance"
          onPress={() => onChange({ ...value, maxDistanceKm: nextDistance(value.maxDistanceKm, 1) })}
          style={styles.stepBtn}
        />
      </View>
    </View>
  );
}

export function InterestPicker({
  options,
  selected,
  onChange
}: {
  options: Interest[];
  selected: string[];
  onChange: (slugs: string[]) => void;
}) {
  return (
    <View style={styles.block}>
      <Text variant="label" style={styles.heading}>
        Interests ({selected.length}/{MAX_INTERESTS})
      </Text>
      <View style={styles.row}>
        {options.map((interest) => {
          const isSelected = selected.includes(interest.slug);
          return (
            <Chip
              key={interest.slug}
              label={interest.label}
              selected={isSelected}
              onPress={() => {
                if (!isSelected && selected.length >= MAX_INTERESTS) return;
                onChange(toggleValue(selected, interest.slug));
              }}
            />
          );
        })}
      </View>
    </View>
  );
}

export function BioField({
  value,
  onChange,
  error
}: {
  value: string;
  onChange: (v: string) => void;
  error?: string | null;
}) {
  return (
    <Input
      label={`About you (${value.trim().length}/${MAX_BIO_LENGTH})`}
      value={value}
      onChangeText={onChange}
      multiline
      numberOfLines={4}
      style={styles.bio}
      error={error}
      testID="bio-input"
    />
  );
}

const styles = StyleSheet.create({
  block: { marginBottom: spacing.lg },
  heading: { marginBottom: spacing.sm },
  row: { flexDirection: "row", flexWrap: "wrap" },
  flex: { flex: 1 },
  stepper: { flexDirection: "row", alignItems: "center", marginBottom: spacing.sm },
  stepBtn: { minHeight: 40, width: 48, paddingHorizontal: 0 },
  stepValue: { width: 48, textAlign: "center" },
  bio: { minHeight: 110, textAlignVertical: "top" }
});
