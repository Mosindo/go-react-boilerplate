import React from "react";
import { StyleSheet, View } from "react-native";
import { useInterests } from "../hooks/data";
import { useDeviceLocation, type DeviceLocation } from "../hooks/useDeviceLocation";
import { GENDER_LABELS } from "../shared/format";
import { Button, Chip, FormField, Loader, Notice, Stepper, Text, spacing } from "../shared/ui";
import type { Gender, Preferences } from "../api/types";

export type Option<T extends string> = { value: T; label: string };

type ChoiceGroupProps<T extends string> = {
  options: Option<T>[];
  value: T[];
  onChange: (next: T[]) => void;
  /** Single choice replaces the value; multi toggles (never emptying the list when `required`). */
  multi?: boolean;
  required?: boolean;
};

export function ChoiceGroup<T extends string>({ options, value, onChange, multi = false, required = false }: ChoiceGroupProps<T>) {
  const toggle = (option: T) => {
    if (!multi) return onChange([option]);
    if (value.includes(option)) {
      if (required && value.length === 1) return;
      return onChange(value.filter((v) => v !== option));
    }
    onChange([...value, option]);
  };
  return (
    <View style={styles.wrap}>
      {options.map((option) => (
        <Chip key={option.value} label={option.label} onPress={() => toggle(option.value)} selected={value.includes(option.value)} />
      ))}
    </View>
  );
}

export const GENDER_OPTIONS: Option<Gender>[] = (Object.keys(GENDER_LABELS) as Gender[]).map((value) => ({ value, label: GENDER_LABELS[value] }));

const MAX_INTERESTS = 10;

export function InterestPicker({ selected, onChange }: { selected: string[]; onChange: (next: string[]) => void }) {
  const { data, isLoading, isError, refetch } = useInterests();
  if (isLoading) return <Loader />;
  if (isError || !data)
    return (
      <Button onPress={() => void refetch()} variant="outline">
        Recharger les centres d'intérêt
      </Button>
    );
  const full = selected.length >= MAX_INTERESTS;
  return (
    <View style={styles.block}>
      <View style={styles.wrap}>
        {data.map((interest) => {
          const isSelected = selected.includes(interest.slug);
          return (
            <Chip
              disabled={full && !isSelected}
              key={interest.slug}
              label={interest.label}
              onPress={() => onChange(isSelected ? selected.filter((s) => s !== interest.slug) : [...selected, interest.slug])}
              selected={isSelected}
            />
          );
        })}
      </View>
      <Text tone="subtle" variant="caption">
        {selected.length}/{MAX_INTERESTS} sélectionnés
      </Text>
    </View>
  );
}

const DISTANCE_OPTIONS: Option<string>[] = [
  { value: "10", label: "10 km" },
  { value: "25", label: "25 km" },
  { value: "50", label: "50 km" },
  { value: "100", label: "100 km" },
  { value: "300", label: "300 km" },
  { value: "any", label: "Partout" }
];

type PreferencesFieldsProps = { value: Preferences; onChange: (next: Preferences) => void };

/** Who I want to meet: genders, age range and maximum distance. */
export function PreferencesFields({ value, onChange }: PreferencesFieldsProps) {
  return (
    <View style={styles.block}>
      <Text variant="label" tone="muted">
        Je souhaite rencontrer
      </Text>
      <ChoiceGroup multi onChange={(interestedIn) => onChange({ ...value, interestedIn })} options={GENDER_OPTIONS} required value={value.interestedIn} />

      <Text variant="label" tone="muted">
        Tranche d'âge
      </Text>
      <Stepper label="Âge minimum" max={Math.min(99, value.maxAge)} min={18} onChange={(minAge) => onChange({ ...value, minAge })} value={value.minAge} />
      <Stepper label="Âge maximum" max={99} min={Math.max(18, value.minAge)} onChange={(maxAge) => onChange({ ...value, maxAge })} value={value.maxAge} />

      <Text variant="label" tone="muted">
        Distance maximale
      </Text>
      <ChoiceGroup
        onChange={([choice]) => onChange({ ...value, maxDistanceKm: choice === "any" || choice === undefined ? null : Number(choice) })}
        options={DISTANCE_OPTIONS}
        value={[value.maxDistanceKm === null ? "any" : String(value.maxDistanceKm)]}
      />
      <Text tone="subtle" variant="caption">
        La distance ne s'applique qu'aux profils qui ont partagé leur position approximative.
      </Text>
    </View>
  );
}

type LocationFieldProps = {
  city: string;
  hasLocation: boolean;
  onCityChange: (city: string) => void;
  onLocated: (location: DeviceLocation) => void;
  onClear?: () => void;
};

/** City label plus an explicit "use my position" action. Nothing is requested until the user taps. */
export function LocationField({ city, hasLocation, onCityChange, onLocated, onClear }: LocationFieldProps) {
  const { locate, loading, error } = useDeviceLocation();
  return (
    <View style={styles.block}>
      <FormField hint="Visible sur votre profil. Facultatif." label="Ville" maxLength={80} onChangeText={onCityChange} value={city} />
      <Button
        icon="locate-outline"
        loading={loading}
        onPress={async () => {
          const location = await locate();
          if (location) onLocated(location);
        }}
        variant="outline"
      >
        {hasLocation ? "Actualiser ma position" : "Utiliser ma position"}
      </Button>
      {hasLocation && onClear ? (
        <Button onPress={onClear} size="sm" variant="ghost">
          Ne plus partager ma position
        </Button>
      ) : null}
      {error ? <Notice message={error} /> : null}
      <Text tone="subtle" variant="caption">
        Votre position exacte n'est jamais affichée ni conservée : elle est arrondie à environ 1 km, et les autres voient seulement une distance approximative.
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  block: { gap: spacing.md }
});
