import { View } from 'react-native';
import type { Gender } from '../../../api';
import { GENDERS } from '../../../api';
import { fr } from '../../../i18n';
import { BIO_MAX, FIRST_NAME_MAX } from '../../../lib/validation';
import { useStyles } from '../../../theme';
import { Chip, Input, Text } from '../../../ui';

export function GenderPicker({
  value,
  onChange,
  error,
}: {
  value: Gender | null;
  onChange: (gender: Gender) => void;
  error?: string | null;
}) {
  const styles = useStyles((t) => ({
    group: { gap: t.spacing.sm },
    chips: { flexDirection: 'row', flexWrap: 'wrap', gap: t.spacing.sm },
  }));
  return (
    <View style={styles.group} accessibilityRole="radiogroup">
      <Text variant="label">{fr.profile.gender}</Text>
      <View style={styles.chips}>
        {GENDERS.map((gender) => (
          <Chip
            key={gender}
            role="radio"
            label={fr.gender.single[gender]}
            selected={value === gender}
            onPress={() => onChange(gender)}
          />
        ))}
      </View>
      {error ? (
        <Text variant="caption" color="danger" accessibilityLiveRegion="polite">
          {error}
        </Text>
      ) : null}
    </View>
  );
}

export interface BirthDateFields {
  day: string;
  month: string;
  year: string;
}

export function BirthDateInput({
  value,
  onChange,
  error,
  disabled,
}: {
  value: BirthDateFields;
  onChange: (next: BirthDateFields) => void;
  error?: string | null;
  disabled?: boolean;
}) {
  const styles = useStyles((t) => ({
    group: { gap: t.spacing.xs },
    row: { flexDirection: 'row', gap: t.spacing.md },
    narrow: { flex: 1 },
    wide: { flex: 1.6 },
  }));
  const digits = (text: string) => text.replace(/\D/g, '');
  return (
    <View style={styles.group}>
      <Text variant="label">{fr.profile.birthDate}</Text>
      <View style={styles.row}>
        <View style={styles.narrow}>
          <Input
            label={fr.profile.day}
            value={value.day}
            onChangeText={(day) => onChange({ ...value, day: digits(day) })}
            keyboardType="number-pad"
            maxLength={2}
            placeholder={fr.profile.dayPlaceholder}
            editable={!disabled}
            testID="birth-day"
          />
        </View>
        <View style={styles.narrow}>
          <Input
            label={fr.profile.month}
            value={value.month}
            onChangeText={(month) => onChange({ ...value, month: digits(month) })}
            keyboardType="number-pad"
            maxLength={2}
            placeholder={fr.profile.monthPlaceholder}
            editable={!disabled}
            testID="birth-month"
          />
        </View>
        <View style={styles.wide}>
          <Input
            label={fr.profile.year}
            value={value.year}
            onChangeText={(year) => onChange({ ...value, year: digits(year) })}
            keyboardType="number-pad"
            maxLength={4}
            placeholder={fr.profile.yearPlaceholder}
            editable={!disabled}
            testID="birth-year"
          />
        </View>
      </View>
      {error ? (
        <Text variant="caption" color="danger" accessibilityLiveRegion="polite">
          {error}
        </Text>
      ) : disabled ? (
        <Text variant="caption" color="textMuted">
          {fr.profile.birthDateLocked}
        </Text>
      ) : null}
    </View>
  );
}

export function FirstNameInput({
  value,
  onChange,
  error,
}: {
  value: string;
  onChange: (value: string) => void;
  error?: string | null;
}) {
  return (
    <Input
      label={fr.profile.firstName}
      value={value}
      onChangeText={onChange}
      error={error}
      maxLength={FIRST_NAME_MAX}
      autoCapitalize="words"
      autoComplete="given-name"
      textContentType="givenName"
      testID="first-name"
    />
  );
}

export function BioInput({
  value,
  onChange,
}: {
  value: string;
  onChange: (value: string) => void;
}) {
  return (
    <Input
      label={fr.profile.bio}
      helper={fr.profile.bioHelper}
      value={value}
      onChangeText={onChange}
      multiline
      maxLength={BIO_MAX}
      showCounter
      placeholder={fr.profile.bioPlaceholder}
      testID="bio"
    />
  );
}
