import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import type { Gender, OwnProfile } from "../../api/types";
import { genderLabel } from "../../lib/format";
import { formatBirthdateInput, isoToBirthdateInput, parseBirthdate, validFirstName } from "../../lib/validation";
import { Badge, Button, FormField, Input, Notice, spacing } from "../../shared/ui";

const GENDERS: Gender[] = ["woman", "man", "non_binary"];

export type BasicsValues = { firstName: string; birthdate: string; gender: Gender; bio: string };

type ProfileBasicsFormProps = {
  initial?: OwnProfile | null;
  submitLabel: string;
  /** Show the bio field (hidden in the first onboarding step). */
  withBio?: boolean;
  busy?: boolean;
  error?: string | null;
  onSubmit: (values: BasicsValues) => void;
};

export function ProfileBasicsForm({ initial, submitLabel, withBio = false, busy = false, error, onSubmit }: ProfileBasicsFormProps) {
  const [firstName, setFirstName] = useState(initial?.firstName ?? "");
  const [birthdate, setBirthdate] = useState(initial ? isoToBirthdateInput(initial.birthdate) : "");
  const [gender, setGender] = useState<Gender | null>(initial?.gender ?? null);
  const [bio, setBio] = useState(initial?.bio ?? "");
  const [errors, setErrors] = useState<{ firstName?: string; birthdate?: string; gender?: string }>({});

  const submit = () => {
    const next: typeof errors = {};
    const nameProblem = validFirstName(firstName);
    if (nameProblem) {
      next.firstName = nameProblem;
    }
    const parsed = parseBirthdate(birthdate);
    if (!parsed.ok) {
      next.birthdate = parsed.error;
    }
    if (!gender) {
      next.gender = "Choisissez une option.";
    }
    setErrors(next);
    if (nameProblem || !parsed.ok || !gender) {
      return;
    }
    onSubmit({ firstName: firstName.trim().replace(/\s+/g, " "), birthdate: parsed.iso, gender, bio: bio.trim() });
  };

  return (
    <View style={styles.root}>
      <Input
        autoCapitalize="words"
        autoComplete="given-name"
        error={errors.firstName}
        label="Prénom"
        maxLength={40}
        onChangeText={setFirstName}
        testID="basics-first-name"
        value={firstName}
      />
      <Input
        error={errors.birthdate}
        helperText="Votre âge sera affiché, pas votre date de naissance."
        keyboardType="number-pad"
        label="Date de naissance"
        maxLength={10}
        onChangeText={(text) => setBirthdate(formatBirthdateInput(text))}
        placeholder="JJ/MM/AAAA"
        testID="basics-birthdate"
        value={birthdate}
      />
      <FormField error={errors.gender} label="Je suis">
        <View style={styles.chips}>
          {GENDERS.map((option) => (
            <Badge key={option} label={genderLabel(option)} onPress={() => setGender(option)} selected={gender === option} />
          ))}
        </View>
      </FormField>
      {withBio ? (
        <Input
          helperText={`${bio.length}/500`}
          label="À propos de moi"
          maxLength={500}
          multiline
          onChangeText={setBio}
          placeholder="Quelques mots pour briser la glace…"
          testID="basics-bio"
          value={bio}
        />
      ) : null}
      {error ? <Notice message={error} tone="danger" /> : null}
      <Button label={submitLabel} loading={busy} onPress={submit} testID="basics-submit" />
    </View>
  );
}

const styles = StyleSheet.create({
  root: { gap: spacing.lg },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }
});
