import React from "react";
import { StyleSheet, View } from "react-native";
import type { Gender, Interest } from "../../../api/types";
import { radii, spacing, useTheme } from "../../../theme";
import { Chip, IconButton, Text } from "../../../ui";
import { genderLabels, genderPluralLabels } from "../../../utils/format";

const genders: Gender[] = ["woman", "man", "nonbinary"];

export function GenderPicker({ value, onChange }: { value: Gender | ""; onChange: (g: Gender) => void }) {
  return (
    <View style={styles.group}>
      <Text tone="muted" variant="label">
        Je suis
      </Text>
      <View style={styles.wrap}>
        {genders.map((g) => (
          <Chip key={g} label={genderLabels[g]} onPress={() => onChange(g)} selected={value === g} testID={`gender-${g}`} />
        ))}
      </View>
    </View>
  );
}

export function InterestedInPicker({ value, onChange }: { value: Gender[]; onChange: (g: Gender[]) => void }) {
  const toggle = (g: Gender) => onChange(value.includes(g) ? value.filter((x) => x !== g) : [...value, g]);
  return (
    <View style={styles.group}>
      <Text tone="muted" variant="label">
        Je souhaite rencontrer
      </Text>
      <View style={styles.wrap}>
        {genders.map((g) => (
          <Chip key={g} label={genderPluralLabels[g]} onPress={() => toggle(g)} selected={value.includes(g)} testID={`interested-${g}`} />
        ))}
      </View>
    </View>
  );
}

export const MAX_INTERESTS = 10;

export function InterestPicker({ catalog, value, onChange }: { catalog: Interest[]; value: string[]; onChange: (v: string[]) => void }) {
  const byCategory = catalog.reduce<Record<string, Interest[]>>((acc, i) => {
    (acc[i.category] ??= []).push(i);
    return acc;
  }, {});
  const toggle = (slug: string) =>
    onChange(value.includes(slug) ? value.filter((s) => s !== slug) : value.length < MAX_INTERESTS ? [...value, slug] : value);
  return (
    <View style={styles.group}>
      <Text tone="muted" variant="label">
        Centres d’intérêt ({value.length}/{MAX_INTERESTS})
      </Text>
      {Object.entries(byCategory).map(([category, items]) => (
        <View key={category} style={styles.category}>
          <Text tone="subtle" variant="overline">
            {category}
          </Text>
          <View style={styles.wrap}>
            {items.map((i) => (
              <Chip
                disabled={!value.includes(i.slug) && value.length >= MAX_INTERESTS}
                key={i.slug}
                label={i.label}
                onPress={() => toggle(i.slug)}
                selected={value.includes(i.slug)}
                testID={`interest-${i.slug}`}
              />
            ))}
          </View>
        </View>
      ))}
    </View>
  );
}

type StepperProps = {
  label: string;
  value: number;
  min: number;
  max: number;
  step?: number;
  onChange: (v: number) => void;
  suffix?: string;
  testID?: string;
};

export function Stepper({ label, value, min, max, step = 1, onChange, suffix, testID }: StepperProps) {
  const { colors } = useTheme();
  return (
    <View style={[styles.stepper, { backgroundColor: colors.surface, borderColor: colors.border }]}>
      <Text style={styles.stepperLabel} variant="bodyStrong">
        {label}
      </Text>
      <IconButton
        disabled={value <= min}
        icon="remove"
        label={`Diminuer ${label}`}
        onPress={() => onChange(Math.max(min, value - step))}
        size={36}
        testID={testID ? `${testID}-minus` : undefined}
        variant="outline"
      />
      <Text accessibilityLiveRegion="polite" style={styles.stepperValue} testID={testID ? `${testID}-value` : undefined} variant="heading">
        {value}
        {suffix ?? ""}
      </Text>
      <IconButton
        disabled={value >= max}
        icon="add"
        label={`Augmenter ${label}`}
        onPress={() => onChange(Math.min(max, value + step))}
        size={36}
        testID={testID ? `${testID}-plus` : undefined}
        variant="outline"
      />
    </View>
  );
}

const distances = [5, 10, 25, 50, 100, 200, 500];

export function DistancePicker({ value, onChange }: { value: number; onChange: (v: number) => void }) {
  return (
    <View style={styles.group}>
      <Text tone="muted" variant="label">
        Distance maximale
      </Text>
      <View style={styles.wrap}>
        {distances.map((d) => (
          <Chip
            key={d}
            label={d === 500 ? "500 km+" : `${d} km`}
            onPress={() => onChange(d)}
            selected={value === d}
            testID={`distance-${d}`}
          />
        ))}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  group: { gap: spacing.sm },
  wrap: { flexDirection: "row", flexWrap: "wrap", gap: spacing.xs },
  category: { gap: spacing.xs, marginTop: spacing.xs },
  stepper: { flexDirection: "row", alignItems: "center", gap: spacing.sm, borderWidth: 1, borderRadius: radii.lg, padding: spacing.sm },
  stepperLabel: { flex: 1 },
  stepperValue: { minWidth: 48, textAlign: "center" }
});
