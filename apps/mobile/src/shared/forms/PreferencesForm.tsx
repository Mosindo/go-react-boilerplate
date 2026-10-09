import React from 'react';
import { View } from 'react-native';
import { GENDERS, type Gender, type Preferences } from '../../api/types';
import { fr } from '../../i18n/fr';
import { spacing } from '../../theme';
import { ChoiceGroup, Stepper, Text } from '../ui';

const choices = GENDERS.map((g) => ({ value: g, label: fr.genderPlural[g] }));

export function PreferencesForm({
  value,
  onChange,
  error,
}: {
  value: Preferences;
  onChange: (next: Preferences) => void;
  error?: string | null;
}) {
  const toggle = (g: Gender) =>
    onChange({
      ...value,
      interestedIn: value.interestedIn.includes(g)
        ? value.interestedIn.filter((x) => x !== g)
        : [...value.interestedIn, g],
    });
  return (
    <View>
      <Text variant="heading" style={{ marginBottom: spacing.sm }}>
        {fr.preferences.interestedIn}
      </Text>
      <ChoiceGroup
        mode="multiple"
        choices={choices}
        selected={value.interestedIn}
        onToggle={toggle}
        testIDPrefix="pref-interestedIn"
        label={fr.preferences.interestedIn}
      />
      {error ? (
        <Text variant="caption" tone="danger" accessibilityRole="alert" testID="pref-error">
          {error}
        </Text>
      ) : null}
      <Text variant="heading" style={{ marginTop: spacing.lg }}>
        {fr.preferences.ageRange}
      </Text>
      <Stepper
        testID="pref-minAge"
        label={fr.preferences.minAge}
        value={value.minAge}
        min={18}
        max={value.maxAge}
        onChange={(minAge) => onChange({ ...value, minAge })}
      />
      <Stepper
        testID="pref-maxAge"
        label={fr.preferences.maxAge}
        value={value.maxAge}
        min={value.minAge}
        max={99}
        onChange={(maxAge) => onChange({ ...value, maxAge })}
      />
      <Text variant="heading" style={{ marginTop: spacing.lg }}>
        {fr.preferences.maxDistance}
      </Text>
      <Stepper
        testID="pref-distance"
        label={fr.preferences.maxDistance}
        value={value.maxDistanceKm}
        min={5}
        max={500}
        step={5}
        format={fr.preferences.distanceValue}
        onChange={(maxDistanceKm) => onChange({ ...value, maxDistanceKm })}
      />
    </View>
  );
}
