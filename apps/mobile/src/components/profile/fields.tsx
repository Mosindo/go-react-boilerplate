import React from "react";
import { View } from "react-native";
import type { Gender, Interest } from "../../api/types";
import { MAX_BIO, MAX_INTERESTS } from "../../lib/validation";
import { spacing } from "../../theme";
import { Chip } from "../Chip";
import { Text } from "../Text";
import { TextField } from "../TextField";
import { Skeleton } from "../Feedback";

export const GENDER_OPTIONS: { value: Gender; label: string }[] = [
  { value: "woman", label: "Woman" },
  { value: "man", label: "Man" },
  { value: "nonbinary", label: "Non-binary" }
];

export function GenderPicker({
  label,
  value,
  onChange,
  testIDPrefix
}: {
  label: string;
  value: Gender | null;
  onChange: (gender: Gender) => void;
  testIDPrefix: string;
}) {
  return (
    <View style={{ gap: spacing.sm }}>
      <Text variant="label">{label}</Text>
      <View style={{ flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }}>
        {GENDER_OPTIONS.map((option) => (
          <Chip
            key={option.value}
            label={option.label}
            selected={value === option.value}
            onPress={() => onChange(option.value)}
            testID={`${testIDPrefix}-${option.value}`}
          />
        ))}
      </View>
    </View>
  );
}

export function GenderMultiPicker({
  value,
  onChange
}: {
  value: Gender[];
  onChange: (genders: Gender[]) => void;
}) {
  return (
    <View style={{ gap: spacing.sm }}>
      <Text variant="label">Interested in</Text>
      <View style={{ flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }}>
        {GENDER_OPTIONS.map((option) => {
          const selected = value.includes(option.value);
          return (
            <Chip
              key={option.value}
              label={option.label}
              selected={selected}
              testID={`pref-interested-${option.value}`}
              onPress={() => onChange(selected ? value.filter((g) => g !== option.value) : [...value, option.value])}
            />
          );
        })}
      </View>
    </View>
  );
}

export function InterestPicker({
  interests,
  loading,
  selectedIds,
  onChange
}: {
  interests: Interest[] | undefined;
  loading: boolean;
  selectedIds: number[];
  onChange: (ids: number[]) => void;
}) {
  const toggle = (id: number): void => {
    if (selectedIds.includes(id)) {
      onChange(selectedIds.filter((x) => x !== id));
    } else if (selectedIds.length < MAX_INTERESTS) {
      onChange([...selectedIds, id]);
    }
  };
  return (
    <View style={{ gap: spacing.sm }}>
      <Text variant="label">
        Interests ({selectedIds.length}/{MAX_INTERESTS})
      </Text>
      {loading ? (
        <Skeleton height={80} />
      ) : (
        <View style={{ flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }}>
          {(interests ?? []).map((interest) => (
            <Chip
              key={interest.id}
              label={interest.label}
              selected={selectedIds.includes(interest.id)}
              onPress={() => toggle(interest.id)}
              testID={`interest-${interest.slug}`}
            />
          ))}
        </View>
      )}
    </View>
  );
}

export function BioField({ value, onChange, error }: { value: string; onChange: (v: string) => void; error?: string | null }) {
  return (
    <TextField
      label="About you"
      value={value}
      onChangeText={onChange}
      error={error}
      multiline
      maxLength={MAX_BIO + 50}
      hint={`${value.length}/${MAX_BIO}`}
      placeholder="A few words about what you love…"
      testID="profile-bio-input"
    />
  );
}
