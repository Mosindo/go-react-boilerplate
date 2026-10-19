import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import * as Location from "expo-location";
import { Button, Chip, Input, Text, useToast } from "../ui";
import { spacing } from "../theme/tokens";
import { ageFromIso, formatBirthInput, parseBirthInput, toBirthInput } from "../lib/dates";
import { errorMessage } from "../lib/errors";
import { useInterests } from "../hooks/queries";
import type { Gender, OwnProfile, ProfileInput } from "../api/types";

export const GENDERS: { value: Gender; label: string }[] = [
  { value: "woman", label: "Femme" },
  { value: "man", label: "Homme" },
  { value: "non_binary", label: "Non-binaire" }
];

const MAX_INTERESTS = 10;

type Props = {
  initial?: OwnProfile | null;
  submitLabel: string;
  onSubmit: (input: ProfileInput) => Promise<void>;
};

/** Builds and validates the payload; exported for unit tests. */
export function buildProfileInput(fields: {
  firstName: string;
  birth: string;
  gender: Gender | null;
  bio: string;
  city: string;
  interests: number[];
  coords: { lat: number; lng: number } | null;
}): { input?: ProfileInput; errors: Record<string, string> } {
  const errors: Record<string, string> = {};
  const firstName = fields.firstName.trim();
  if (!firstName) errors.firstName = "Indiquez votre prénom.";
  const iso = parseBirthInput(fields.birth);
  if (!iso) errors.birth = "Date invalide (JJ/MM/AAAA).";
  else if (ageFromIso(iso) < 18) errors.birth = "Vous devez avoir au moins 18 ans.";
  else if (ageFromIso(iso) > 120) errors.birth = "Date invalide.";
  if (!fields.gender) errors.gender = "Choisissez une option.";
  if (fields.bio.length > 500) errors.bio = "500 caractères maximum.";
  if (Object.keys(errors).length > 0 || !iso || !fields.gender) return { errors };
  return {
    errors,
    input: {
      firstName,
      birthDate: iso,
      gender: fields.gender,
      bio: fields.bio.trim(),
      city: fields.city.trim(),
      interests: fields.interests,
      ...(fields.coords ? { location: fields.coords } : {})
    }
  };
}

export function ProfileForm({ initial, submitLabel, onSubmit }: Props) {
  const toast = useToast();
  const interests = useInterests();
  const [firstName, setFirstName] = useState(initial?.firstName ?? "");
  const [birth, setBirth] = useState(initial ? toBirthInput(initial.birthDate) : "");
  const [gender, setGender] = useState<Gender | null>(initial?.gender ?? null);
  const [bio, setBio] = useState(initial?.bio ?? "");
  const [city, setCity] = useState(initial?.city ?? "");
  const [selected, setSelected] = useState<number[]>(initial?.interests.map((i) => i.id) ?? []);
  const [coords, setCoords] = useState<{ lat: number; lng: number } | null>(null);
  const [locating, setLocating] = useState(false);
  const [saving, setSaving] = useState(false);
  const [errors, setErrors] = useState<Record<string, string>>({});

  const locate = async () => {
    setLocating(true);
    try {
      const permission = await Location.requestForegroundPermissionsAsync();
      if (permission.status !== "granted") {
        toast.show("Autorisez la localisation ou saisissez votre ville.", "error");
        return;
      }
      const position = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Balanced });
      const next = { lat: position.coords.latitude, lng: position.coords.longitude };
      setCoords(next);
      const places = await Location.reverseGeocodeAsync({ latitude: next.lat, longitude: next.lng }).catch(
        () => []
      );
      const name = places[0]?.city ?? places[0]?.subregion;
      if (name) setCity(name);
      toast.show("Position approximative enregistrée.", "success");
    } catch {
      toast.show("Impossible de vous localiser pour le moment.", "error");
    } finally {
      setLocating(false);
    }
  };

  const toggleInterest = (id: number) =>
    setSelected((current) =>
      current.includes(id)
        ? current.filter((x) => x !== id)
        : current.length < MAX_INTERESTS
          ? [...current, id]
          : current
    );

  const submit = async () => {
    const result = buildProfileInput({ firstName, birth, gender, bio, city, interests: selected, coords });
    setErrors(result.errors);
    if (!result.input) return;
    setSaving(true);
    try {
      await onSubmit(result.input);
    } catch (error) {
      toast.show(errorMessage(error), "error");
    } finally {
      setSaving(false);
    }
  };

  return (
    <View style={styles.form}>
      <Input
        label="Prénom"
        value={firstName}
        onChangeText={setFirstName}
        error={errors.firstName}
        maxLength={40}
        autoCapitalize="words"
        textContentType="givenName"
        testID="profile-first-name"
      />
      <Input
        label="Date de naissance"
        value={birth}
        onChangeText={(v) => setBirth(formatBirthInput(v))}
        error={errors.birth}
        placeholder="JJ/MM/AAAA"
        keyboardType="number-pad"
        maxLength={10}
        hint="Seul votre âge sera visible."
        testID="profile-birth"
      />
      <View style={styles.group}>
        <Text variant="caption" tone="muted" bold>
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
          <Text variant="caption" tone="danger">
            {errors.gender}
          </Text>
        ) : null}
      </View>
      <Input label="Ville" value={city} onChangeText={setCity} maxLength={80} placeholder="Votre ville" />
      <Button
        label={coords ? "Position mise à jour ✓" : "Utiliser ma position approximative"}
        variant="secondary"
        onPress={locate}
        loading={locating}
      />
      <Text variant="caption" tone="muted">
        Votre position exacte n&apos;est jamais affichée ni partagée : seule une distance approximative est
        visible.
      </Text>
      <Input
        label="À propos de moi"
        value={bio}
        onChangeText={setBio}
        error={errors.bio}
        multiline
        maxLength={500}
        placeholder="Quelques mots sur vous…"
        hint={`${bio.length}/500`}
      />
      <View style={styles.group}>
        <Text variant="caption" tone="muted" bold>
          Centres d&apos;intérêt ({selected.length}/{MAX_INTERESTS})
        </Text>
        <View style={styles.wrap}>
          {(interests.data ?? []).map((interest) => (
            <Chip
              key={interest.id}
              label={interest.label}
              selected={selected.includes(interest.id)}
              onPress={() => toggleInterest(interest.id)}
            />
          ))}
        </View>
      </View>
      <Button label={submitLabel} onPress={submit} loading={saving} testID="profile-submit" />
    </View>
  );
}

const styles = StyleSheet.create({
  form: { gap: spacing.lg },
  group: { gap: spacing.sm },
  row: { flexDirection: "row", gap: spacing.sm, flexWrap: "wrap" },
  wrap: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }
});
