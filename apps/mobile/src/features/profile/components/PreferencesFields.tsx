import { View } from 'react-native';
import type { Gender, Preferences } from '../../../api';
import { GENDERS } from '../../../api';
import { fr } from '../../../i18n';
import {
  MAX_AGE,
  MIN_AGE,
  adjustAgeRange,
  distanceProgress,
  stepDistance,
} from '../../../lib/validation';
import { useStyles } from '../../../theme';
import { Chip, Text } from '../../../ui';
import { Stepper } from './Stepper';

interface PreferencesFieldsProps {
  value: Preferences;
  onChange: (next: Preferences) => void;
  interestedInError?: string | null;
}

export function PreferencesFields({ value, onChange, interestedInError }: PreferencesFieldsProps) {
  const styles = useStyles((t) => ({
    wrap: { gap: t.spacing.xl },
    group: { gap: t.spacing.sm },
    chips: { flexDirection: 'row', flexWrap: 'wrap', gap: t.spacing.sm },
  }));

  const toggle = (gender: Gender) => {
    const next = value.interestedIn.includes(gender)
      ? value.interestedIn.filter((g) => g !== gender)
      : [...value.interestedIn, gender];
    onChange({ ...value, interestedIn: next });
  };
  const setAges = (change: Partial<{ minAge: number; maxAge: number }>) =>
    onChange({ ...value, ...adjustAgeRange(value, change) });

  return (
    <View style={styles.wrap}>
      <View style={styles.group}>
        <Text variant="label">{fr.preferences.interestedIn}</Text>
        <View style={styles.chips}>
          {GENDERS.map((gender) => (
            <Chip
              key={gender}
              label={fr.gender.plural[gender]}
              selected={value.interestedIn.includes(gender)}
              onPress={() => toggle(gender)}
            />
          ))}
        </View>
        {interestedInError ? (
          <Text variant="caption" color="danger" accessibilityLiveRegion="polite">
            {interestedInError}
          </Text>
        ) : null}
      </View>

      <Stepper
        label={fr.preferences.minAge}
        valueText={fr.preferences.years(value.minAge)}
        progress={(value.minAge - MIN_AGE) / (MAX_AGE - MIN_AGE)}
        decrementLabel={fr.preferences.minAgeLess}
        incrementLabel={fr.preferences.minAgeMore}
        onDecrement={() => setAges({ minAge: value.minAge - 1 })}
        onIncrement={() => setAges({ minAge: value.minAge + 1 })}
        canDecrement={value.minAge > MIN_AGE}
        canIncrement={value.minAge < MAX_AGE}
      />
      <Stepper
        label={fr.preferences.maxAge}
        valueText={fr.preferences.years(value.maxAge)}
        progress={(value.maxAge - MIN_AGE) / (MAX_AGE - MIN_AGE)}
        decrementLabel={fr.preferences.maxAgeLess}
        incrementLabel={fr.preferences.maxAgeMore}
        onDecrement={() => setAges({ maxAge: value.maxAge - 1 })}
        onIncrement={() => setAges({ maxAge: value.maxAge + 1 })}
        canDecrement={value.maxAge > MIN_AGE}
        canIncrement={value.maxAge < MAX_AGE}
      />
      <Stepper
        label={fr.preferences.maxDistance}
        valueText={
          value.maxDistanceKm === null
            ? fr.preferences.anywhere
            : fr.profile.distanceKm(value.maxDistanceKm)
        }
        progress={distanceProgress(value.maxDistanceKm)}
        decrementLabel={fr.preferences.distanceLess}
        incrementLabel={fr.preferences.distanceMore}
        onDecrement={() =>
          onChange({ ...value, maxDistanceKm: stepDistance(value.maxDistanceKm, -1) })
        }
        onIncrement={() =>
          onChange({ ...value, maxDistanceKm: stepDistance(value.maxDistanceKm, 1) })
        }
        canDecrement={stepDistance(value.maxDistanceKm, -1) !== value.maxDistanceKm}
        canIncrement={stepDistance(value.maxDistanceKm, 1) !== value.maxDistanceKm}
      />
    </View>
  );
}
