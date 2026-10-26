import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import type { Gender, OwnProfile, ProfileInput } from "../api";
import { useInterests } from "../hooks/useQueries";
import { ageFromIso, maskDate, parseBirthDate, toDisplayDate } from "../lib/format";
import { spacing, useTheme } from "../theme";
import { Button } from "./Button";
import { Chip } from "./Chip";
import { LoadingState } from "./States";
import { Text } from "./Text";
import { TextField } from "./TextField";

const GENDERS: { value: Gender; label: string }[] = [
  { value: "woman", label: "Femme" },
  { value: "man", label: "Homme" },
  { value: "nonbinary", label: "Non-binaire" }
];

const MAX_INTERESTS = 10;

type Props = {
  initial?: OwnProfile | null;
  submitLabel: string;
  loading?: boolean;
  serverError?: string | null;
  onSubmit: (input: ProfileInput) => void;
};

export function ProfileForm({ initial, submitLabel, loading, serverError, onSubmit }: Props) {
  const theme = useTheme();
  const interests = useInterests();
  const [firstName, setFirstName] = useState(initial?.firstName ?? "");
  const [birth, setBirth] = useState(initial ? toDisplayDate(initial.birthDate) : "");
  const [gender, setGender] = useState<Gender | null>(initial?.gender ?? null);
  const [jobTitle, setJobTitle] = useState(initial?.jobTitle ?? "");
  const [bio, setBio] = useState(initial?.bio ?? "");
  const [selected, setSelected] = useState<string[]>(initial?.interests.map((i) => i.slug) ?? []);
  const [errors, setErrors] = useState<Record<string, string>>({});

  const locked = !!initial; // the birth date is fixed once the profile exists

  const toggle = (slug: string) =>
    setSelected((cur) =>
      cur.includes(slug) ? cur.filter((s) => s !== slug) : cur.length < MAX_INTERESTS ? [...cur, slug] : cur
    );

  const submit = () => {
    const next: Record<string, string> = {};
    if (!firstName.trim()) next.firstName = "Votre prénom est requis.";
    const iso = locked ? initial.birthDate : parseBirthDate(birth);
    if (!iso) next.birth = "Utilisez le format JJ/MM/AAAA.";
    else if (ageFromIso(iso) < 18) next.birth = "Lien est réservé aux personnes majeures.";
    if (!gender) next.gender = "Choisissez une option.";
    setErrors(next);
    if (Object.keys(next).length > 0 || !iso || !gender) return;
    onSubmit({
      firstName: firstName.trim(),
      birthDate: iso,
      gender,
      bio: bio.trim(),
      jobTitle: jobTitle.trim(),
      interests: selected
    });
  };

  if (interests.isPending) return <LoadingState />;

  return (
    <View style={styles.form}>
      <TextField
        label="Prénom"
        value={firstName}
        onChangeText={setFirstName}
        error={errors.firstName}
        maxLength={40}
        autoCapitalize="words"
        testID="first-name"
      />
      <TextField
        label="Date de naissance"
        value={birth}
        onChangeText={(t) => setBirth(maskDate(t))}
        error={errors.birth}
        hint={locked ? "La date de naissance ne peut pas être modifiée." : "Vous devez avoir 18 ans ou plus."}
        placeholder="JJ/MM/AAAA"
        keyboardType="number-pad"
        editable={!locked}
        maxLength={10}
        testID="birth-date"
      />
      <View style={styles.group}>
        <Text variant="label" muted>
          Je suis
        </Text>
        <View style={styles.row}>
          {GENDERS.map((g) => (
            <Chip
              key={g.value}
              label={g.label}
              selected={gender === g.value}
              onPress={() => setGender(g.value)}
              testID={`gender-${g.value}`}
            />
          ))}
        </View>
        {errors.gender ? (
          <Text variant="caption" color={theme.danger}>
            {errors.gender}
          </Text>
        ) : null}
      </View>
      <TextField label="Métier (facultatif)" value={jobTitle} onChangeText={setJobTitle} maxLength={80} />
      <TextField
        label="À propos de moi (facultatif)"
        value={bio}
        onChangeText={setBio}
        multiline
        maxLength={500}
        hint={`${bio.length}/500`}
        testID="bio"
      />
      <View style={styles.group}>
        <Text variant="label" muted>
          Centres d&apos;intérêt ({selected.length}/{MAX_INTERESTS})
        </Text>
        <View style={styles.row}>
          {(interests.data ?? []).map((i) => (
            <Chip
              key={i.slug}
              label={i.label}
              selected={selected.includes(i.slug)}
              onPress={() => toggle(i.slug)}
            />
          ))}
        </View>
      </View>
      {serverError ? (
        <Text color={theme.danger} accessibilityRole="alert">
          {serverError}
        </Text>
      ) : null}
      <Button label={submitLabel} onPress={submit} loading={loading} testID="save-profile" />
    </View>
  );
}

const styles = StyleSheet.create({
  form: { gap: spacing.lg },
  group: { gap: spacing.sm },
  row: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }
});
