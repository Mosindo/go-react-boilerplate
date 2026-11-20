import React, { useMemo } from "react";
import { StyleSheet, View } from "react-native";
import type { Gender, Interest, RelationshipGoal } from "../../api/types";
import { genderLabels, genderPluralLabels, goalLabels } from "../../lib/format";
import { toggle } from "../../lib/validation";
import { useTheme } from "../../theme";
import { Chip, SectionTitle, Stepper, Text, TextField } from "../../ui";

export const MAX_INTERESTS = 10;

export type BasicsValue = { firstName: string; day: string; month: string; year: string; gender: Gender | null };

export function BasicsFields({
  value,
  onChange,
  errors,
  birthdateLocked
}: {
  value: BasicsValue;
  onChange: (v: BasicsValue) => void;
  errors: { firstName?: string | null; birthdate?: string | null; gender?: string | null };
  birthdateLocked?: boolean;
}) {
  const { spacing } = useTheme();
  return (
    <View style={{ gap: spacing.lg }}>
      <TextField
        autoCapitalize="words"
        autoComplete="given-name"
        error={errors.firstName}
        label="Prénom"
        maxLength={40}
        onChangeText={(firstName) => onChange({ ...value, firstName })}
        placeholder="Ton prénom"
        testID="field-firstname"
        value={value.firstName}
      />
      <View style={{ gap: 6 }}>
        <Text tone="muted" variant="label">
          Date de naissance
        </Text>
        <View style={styles.dateRow}>
          <View style={styles.dateSmall}>
            <TextField
              editable={!birthdateLocked}
              inputMode="numeric"
              label="Jour"
              maxLength={2}
              onChangeText={(day) => onChange({ ...value, day })}
              placeholder="JJ"
              testID="field-day"
              value={value.day}
            />
          </View>
          <View style={styles.dateSmall}>
            <TextField
              editable={!birthdateLocked}
              inputMode="numeric"
              label="Mois"
              maxLength={2}
              onChangeText={(month) => onChange({ ...value, month })}
              placeholder="MM"
              testID="field-month"
              value={value.month}
            />
          </View>
          <View style={styles.dateLarge}>
            <TextField
              editable={!birthdateLocked}
              inputMode="numeric"
              label="Année"
              maxLength={4}
              onChangeText={(year) => onChange({ ...value, year })}
              placeholder="AAAA"
              testID="field-year"
              value={value.year}
            />
          </View>
        </View>
        {errors.birthdate ? (
          <Text tone="danger" variant="caption">
            {errors.birthdate}
          </Text>
        ) : (
          <Text tone="subtle" variant="caption">
            {birthdateLocked ? "La date de naissance ne peut pas être modifiée." : "Seul ton âge sera visible. Elle ne pourra plus être modifiée."}
          </Text>
        )}
      </View>
      <View style={{ gap: spacing.xs }}>
        <Text tone="muted" variant="label">
          Je suis
        </Text>
        <View style={styles.wrap}>
          {(Object.keys(genderLabels) as Gender[]).map((g) => (
            <Chip key={g} label={genderLabels[g]} onPress={() => onChange({ ...value, gender: g })} selected={value.gender === g} testID={`gender-${g}`} />
          ))}
        </View>
        {errors.gender ? (
          <Text tone="danger" variant="caption">
            {errors.gender}
          </Text>
        ) : null}
      </View>
    </View>
  );
}

export type PreferencesValue = { interestedIn: Gender[]; ageMin: number; ageMax: number; maxDistanceKm: number };

export function PreferencesFields({ value, onChange, error }: { value: PreferencesValue; onChange: (v: PreferencesValue) => void; error?: string | null }) {
  const { spacing } = useTheme();
  return (
    <View style={{ gap: spacing.lg }}>
      <View style={{ gap: spacing.xs }}>
        <Text tone="muted" variant="label">
          Je souhaite rencontrer
        </Text>
        <View style={styles.wrap}>
          {(Object.keys(genderPluralLabels) as Gender[]).map((g) => (
            <Chip
              key={g}
              label={genderPluralLabels[g]}
              onPress={() => onChange({ ...value, interestedIn: toggle(value.interestedIn, g) })}
              selected={value.interestedIn.includes(g)}
              testID={`interested-${g}`}
            />
          ))}
        </View>
        {error ? (
          <Text tone="danger" variant="caption">
            {error}
          </Text>
        ) : null}
      </View>
      <View style={{ gap: spacing.xs }}>
        <Stepper label="Âge minimum" max={value.ageMax} min={18} onChange={(ageMin) => onChange({ ...value, ageMin })} value={value.ageMin} />
        <Stepper label="Âge maximum" max={99} min={value.ageMin} onChange={(ageMax) => onChange({ ...value, ageMax })} value={value.ageMax} />
        <Stepper
          format={(v) => `${v} km`}
          label="Distance maximale"
          max={500}
          min={5}
          onChange={(maxDistanceKm) => onChange({ ...value, maxDistanceKm })}
          step={5}
          value={value.maxDistanceKm}
        />
      </View>
    </View>
  );
}

export type AboutValue = { bio: string; jobTitle: string; city: string; relationshipGoal: RelationshipGoal | null };

export function AboutFields({ value, onChange }: { value: AboutValue; onChange: (v: AboutValue) => void }) {
  const { spacing } = useTheme();
  return (
    <View style={{ gap: spacing.lg }}>
      <TextField
        counter={500}
        label="Quelques mots sur toi"
        maxLength={500}
        multiline
        onChangeText={(bio) => onChange({ ...value, bio })}
        placeholder="Ce qui te fait vibrer, ton dimanche idéal…"
        testID="field-bio"
        value={value.bio}
      />
      <TextField label="Métier (facultatif)" maxLength={60} onChangeText={(jobTitle) => onChange({ ...value, jobTitle })} value={value.jobTitle} />
      <TextField
        hint="Affichée sur ton profil à la place d’une adresse."
        label="Ville (facultatif)"
        maxLength={60}
        onChangeText={(city) => onChange({ ...value, city })}
        value={value.city}
      />
      <View style={{ gap: spacing.xs }}>
        <Text tone="muted" variant="label">
          Je cherche (facultatif)
        </Text>
        <View style={styles.wrap}>
          {(Object.keys(goalLabels) as RelationshipGoal[]).map((g) => (
            <Chip
              key={g}
              label={goalLabels[g]}
              onPress={() => onChange({ ...value, relationshipGoal: value.relationshipGoal === g ? null : g })}
              selected={value.relationshipGoal === g}
            />
          ))}
        </View>
      </View>
    </View>
  );
}

export function InterestPicker({ interests, selected, onChange }: { interests: Interest[]; selected: number[]; onChange: (ids: number[]) => void }) {
  const { spacing } = useTheme();
  const byCategory = useMemo(() => {
    const groups = new Map<string, Interest[]>();
    for (const interest of interests) {
      groups.set(interest.category, [...(groups.get(interest.category) ?? []), interest]);
    }
    return [...groups.entries()];
  }, [interests]);
  return (
    <View style={{ gap: spacing.lg }}>
      <Text tone="muted" variant="caption">
        {selected.length}/{MAX_INTERESTS} sélectionnés
      </Text>
      {byCategory.map(([category, items]) => (
        <View key={category} style={{ gap: spacing.xs }}>
          <SectionTitle>{category}</SectionTitle>
          <View style={styles.wrap}>
            {items.map((interest) => (
              <Chip
                key={interest.id}
                label={interest.label}
                onPress={() => onChange(toggle(selected, interest.id, MAX_INTERESTS))}
                selected={selected.includes(interest.id)}
                testID={`interest-${interest.slug}`}
              />
            ))}
          </View>
        </View>
      ))}
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { flexDirection: "row", flexWrap: "wrap", gap: 8 },
  dateRow: { flexDirection: "row", gap: 10 },
  dateSmall: { flex: 1 },
  dateLarge: { flex: 1.5 }
});
