import React from "react";
import { StyleSheet, View } from "react-native";
import type { Gender, Interest } from "../api/types";
import { GENDER_LABELS } from "../api/dating";
import { useInterests } from "../hooks/queries";
import { spacing } from "../theme";
import { Chip, Input, Text } from "../ui";

export const MAX_INTERESTS = 10;

export type BasicsValue = { firstName: string; day: string; month: string; year: string; gender: Gender | null };
export type AboutValue = { bio: string; occupation: string; city: string; interests: string[] };

type BasicsProps = {
  value: BasicsValue;
  onChange: (next: BasicsValue) => void;
  errors: { firstName?: string | null; birthDate?: string | null; gender?: string | null };
  /** The birth date is locked once the profile exists (age gate). */
  lockBirthDate?: boolean;
};

export function BasicsFields({ value, onChange, errors, lockBirthDate }: BasicsProps) {
  const set = <K extends keyof BasicsValue>(key: K, next: BasicsValue[K]) => onChange({ ...value, [key]: next });
  return (
    <View style={styles.group}>
      <Input
        label="Prénom"
        value={value.firstName}
        onChangeText={(t) => set("firstName", t)}
        error={errors.firstName}
        autoCapitalize="words"
        autoComplete="given-name"
        maxLength={40}
      />
      <View>
        <Text variant="label" tone="muted" style={styles.label}>
          Date de naissance
        </Text>
        <View style={styles.dateRow}>
          <Input
            label="Jour"
            value={value.day}
            onChangeText={(t) => set("day", t.replace(/\D/g, "").slice(0, 2))}
            keyboardType="number-pad"
            editable={!lockBirthDate}
            placeholder="JJ"
            style={styles.date}
          />
          <Input
            label="Mois"
            value={value.month}
            onChangeText={(t) => set("month", t.replace(/\D/g, "").slice(0, 2))}
            keyboardType="number-pad"
            editable={!lockBirthDate}
            placeholder="MM"
            style={styles.date}
          />
          <Input
            label="Année"
            value={value.year}
            onChangeText={(t) => set("year", t.replace(/\D/g, "").slice(0, 4))}
            keyboardType="number-pad"
            editable={!lockBirthDate}
            placeholder="AAAA"
            style={styles.date}
          />
        </View>
        {errors.birthDate ? (
          <Text variant="caption" tone="danger">
            {errors.birthDate}
          </Text>
        ) : (
          <Text variant="caption" tone="muted">
            {lockBirthDate
              ? "La date de naissance ne peut plus être modifiée."
              : "Seul votre âge sera visible. Réservé aux 18 ans et plus."}
          </Text>
        )}
      </View>
      <View style={styles.group}>
        <Text variant="label" tone="muted">
          Je suis
        </Text>
        <View style={styles.chips}>
          {(Object.keys(GENDER_LABELS) as Gender[]).map((g) => (
            <Chip key={g} label={GENDER_LABELS[g]} selected={value.gender === g} onPress={() => set("gender", g)} />
          ))}
        </View>
        {errors.gender ? (
          <Text variant="caption" tone="danger">
            {errors.gender}
          </Text>
        ) : null}
      </View>
    </View>
  );
}

export function AboutFields({ value, onChange }: { value: AboutValue; onChange: (next: AboutValue) => void }) {
  const interests = useInterests();
  const toggle = (slug: string) => {
    const has = value.interests.includes(slug);
    if (!has && value.interests.length >= MAX_INTERESTS) {
      return;
    }
    onChange({ ...value, interests: has ? value.interests.filter((s) => s !== slug) : [...value.interests, slug] });
  };
  return (
    <View style={styles.group}>
      <Input
        label="Bio"
        value={value.bio}
        onChangeText={(t) => onChange({ ...value, bio: t })}
        multiline
        maxLength={500}
        hint={`${value.bio.length}/500`}
        placeholder="Quelques mots sur vous…"
      />
      <Input
        label="Métier ou études (facultatif)"
        value={value.occupation}
        onChangeText={(t) => onChange({ ...value, occupation: t })}
        maxLength={80}
      />
      <Input
        label="Ville (facultatif)"
        value={value.city}
        onChangeText={(t) => onChange({ ...value, city: t })}
        maxLength={80}
        autoComplete="off"
      />
      <View style={styles.group}>
        <Text variant="label" tone="muted">
          Centres d'intérêt ({value.interests.length}/{MAX_INTERESTS})
        </Text>
        <View style={styles.chips}>
          {(interests.data ?? []).map((i: Interest) => (
            <Chip key={i.slug} label={i.label} selected={value.interests.includes(i.slug)} onPress={() => toggle(i.slug)} />
          ))}
        </View>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  group: { gap: spacing.md },
  label: { marginBottom: spacing.xs },
  dateRow: { flexDirection: "row", gap: spacing.sm },
  date: { textAlign: "center" },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }
});
