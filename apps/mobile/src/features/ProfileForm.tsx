import React, { useState } from "react";
import { View } from "react-native";
import type { Gender, ProfileInfo, ProfileInput } from "../api/types";
import { Button, Chip, Text, TextField } from "../ui";
import { ageFromIso, genderLabels, isoToFrenchDate, maskDate, parseFrenchDate } from "../utils/format";

type Props = {
  initial: ProfileInfo | null;
  submitLabel: string;
  loading: boolean;
  error?: string | null;
  onSubmit: (input: ProfileInput) => void;
};

export function ProfileForm({ initial, submitLabel, loading, error, onSubmit }: Props) {
  const [firstName, setFirstName] = useState(initial?.firstName ?? "");
  const [birth, setBirth] = useState(initial ? isoToFrenchDate(initial.birthdate) : "");
  const [gender, setGender] = useState<Gender | null>(initial?.gender ?? null);
  const [bio, setBio] = useState(initial?.bio ?? "");
  const [jobTitle, setJobTitle] = useState(initial?.jobTitle ?? "");
  const [city, setCity] = useState(initial?.city ?? "");

  const iso = parseFrenchDate(birth);
  const age = iso ? ageFromIso(iso) : null;
  const birthError =
    birth.length === 10 && !iso ? "Date invalide." : age !== null && age < 18 ? "Tu dois avoir au moins 18 ans." : null;
  const valid = firstName.trim().length > 0 && !!iso && age !== null && age >= 18 && age <= 100 && !!gender;

  return (
    <View style={{ gap: 18 }}>
      <TextField
        testID="profile-first-name"
        label="Prénom"
        value={firstName}
        onChangeText={setFirstName}
        maxLength={40}
        autoComplete="given-name"
        textContentType="givenName"
      />
      <TextField
        testID="profile-birthdate"
        label="Date de naissance"
        placeholder="JJ/MM/AAAA"
        value={birth}
        onChangeText={(v) => setBirth(maskDate(v))}
        keyboardType="number-pad"
        maxLength={10}
        error={birthError}
        hint={age !== null && !birthError ? `${age} ans — seul ton âge sera affiché.` : "Seul ton âge sera visible."}
      />
      <View style={{ gap: 8 }}>
        <Text variant="label" tone="muted">
          Je suis
        </Text>
        <View style={{ flexDirection: "row", flexWrap: "wrap", gap: 8 }}>
          {(Object.keys(genderLabels) as Gender[]).map((g) => (
            <Chip
              key={g}
              testID={`gender-${g}`}
              label={genderLabels[g]}
              selected={gender === g}
              onPress={() => setGender(g)}
            />
          ))}
        </View>
      </View>
      <TextField
        testID="profile-bio"
        label="Bio"
        value={bio}
        onChangeText={setBio}
        multiline
        maxLength={500}
        placeholder="Ce qui te rend unique, ce que tu cherches…"
        hint={`${bio.length}/500`}
      />
      <TextField label="Métier (facultatif)" value={jobTitle} onChangeText={setJobTitle} maxLength={60} />
      <TextField
        label="Ville (facultatif)"
        value={city}
        onChangeText={setCity}
        maxLength={60}
        hint="Affichée telle quelle sur ton profil."
      />
      {error ? <Text tone="danger">{error}</Text> : null}
      <Button
        testID="profile-submit"
        label={submitLabel}
        loading={loading}
        disabled={!valid}
        onPress={() =>
          iso &&
          gender &&
          onSubmit({
            firstName: firstName.trim(),
            birthdate: iso,
            gender,
            bio: bio.trim(),
            jobTitle: jobTitle.trim(),
            city: city.trim()
          })
        }
      />
    </View>
  );
}
