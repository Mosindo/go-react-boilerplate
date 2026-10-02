import * as Location from "expo-location";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import React, { useState } from "react";
import { Platform, StyleSheet, View } from "react-native";

import { profileApi } from "../../api/endpoints";
import type { Gender, OwnProfile } from "../../api/types";
import { ApiError } from "../../api/client";
import { useInterests } from "../../hooks";
import {
  birthDateError,
  isoToBirthInput,
  maskBirthInput,
  parseBirthDate,
} from "../../lib/validation";
import { keys } from "../../realtime/RealtimeProvider";
import { spacing } from "../../theme";
import { Button } from "../../ui/Button";
import { Chip } from "../../ui/Chip";
import { useFeedback } from "../../ui/Feedback";
import { Screen } from "../../ui/Screen";
import { errorMessage, Loading } from "../../ui/States";
import { Text } from "../../ui/Text";
import { TextField } from "../../ui/TextField";

export const GENDER_LABELS: Record<Gender, string> = {
  woman: "Femme",
  man: "Homme",
  nonbinary: "Non-binaire",
};
const MAX_INTERESTS = 10;

export interface ProfileFormProps {
  /** Existing profile when editing; absent during onboarding. */
  initial?: OwnProfile | null;
  submitLabel: string;
  onSaved: () => void;
}

async function currentPosition(): Promise<{ latitude: number; longitude: number; city: string }> {
  if (Platform.OS === "web") {
    const pos = await new Promise<GeolocationPosition>((resolve, reject) =>
      navigator.geolocation.getCurrentPosition(resolve, reject, { timeout: 10_000 }),
    );
    return { latitude: pos.coords.latitude, longitude: pos.coords.longitude, city: "" };
  }
  const perm = await Location.requestForegroundPermissionsAsync();
  if (perm.status !== "granted") throw new Error("denied");
  const pos = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Balanced });
  let city = "";
  try {
    const [place] = await Location.reverseGeocodeAsync(pos.coords);
    city = place?.city ?? place?.subregion ?? "";
  } catch {
    // reverse geocoding is best effort
  }
  return { latitude: pos.coords.latitude, longitude: pos.coords.longitude, city };
}

export function ProfileForm({ initial, submitLabel, onSaved }: ProfileFormProps) {
  const qc = useQueryClient();
  const { toast } = useFeedback();
  const interestsQuery = useInterests();
  const birthLocked = Boolean(initial);

  const [firstName, setFirstName] = useState(initial?.firstName ?? "");
  const [birth, setBirth] = useState(initial ? isoToBirthInput(initial.birthDate) : "");
  const [gender, setGender] = useState<Gender | null>(initial?.gender ?? null);
  const [city, setCity] = useState(initial?.city ?? "");
  const [coords, setCoords] = useState<{ latitude: number; longitude: number } | null>(
    initial ? { latitude: initial.latitude, longitude: initial.longitude } : null,
  );
  const [bio, setBio] = useState(initial?.bio ?? "");
  const [selected, setSelected] = useState<string[]>(initial?.interests.map((i) => i.slug) ?? []);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [locating, setLocating] = useState(false);

  const save = useMutation({
    mutationFn: profileApi.save,
    onSuccess: (status) => {
      qc.setQueryData(keys.profile, status);
      void qc.invalidateQueries({ queryKey: keys.discover });
      onSaved();
    },
    onError: (err) => {
      if (err instanceof ApiError && err.code === "underage") setErrors({ birth: err.message });
      else setErrors({ form: errorMessage(err) });
    },
  });

  const locate = async () => {
    setLocating(true);
    try {
      const pos = await currentPosition();
      setCoords({ latitude: pos.latitude, longitude: pos.longitude });
      if (pos.city) setCity(pos.city);
      setErrors((e) => ({ ...e, location: "" }));
    } catch {
      toast("Position indisponible. Autorisez la localisation dans les réglages.", "error");
    } finally {
      setLocating(false);
    }
  };

  const toggleInterest = (slug: string) =>
    setSelected((cur) =>
      cur.includes(slug)
        ? cur.filter((s) => s !== slug)
        : cur.length >= MAX_INTERESTS
          ? cur
          : [...cur, slug],
    );

  const submit = () => {
    const next: Record<string, string> = {};
    if (!firstName.trim()) next.firstName = "Votre prénom est requis.";
    if (!birthLocked) {
      const birthErr = birthDateError(birth);
      if (birthErr) next.birth = birthErr;
    }
    if (!gender) next.gender = "Choisissez une option.";
    if (!coords) next.location = "Indiquez votre position approximative.";
    setErrors(next);
    if (Object.keys(next).length > 0 || !gender || !coords) return;

    const birthDate = initial?.birthDate ?? parseBirthDate(birth);
    if (!birthDate) return;
    save.mutate({
      firstName: firstName.trim(),
      birthDate,
      gender,
      bio: bio.trim(),
      city: city.trim(),
      latitude: coords.latitude,
      longitude: coords.longitude,
      interests: selected,
      discoverable: initial?.discoverable,
      showDistance: initial?.showDistance,
    });
  };

  if (interestsQuery.isLoading) return <Loading />;

  return (
    <>
      <TextField
        label="Prénom"
        value={firstName}
        onChangeText={setFirstName}
        error={errors.firstName}
        maxLength={50}
        autoComplete="given-name"
        testID="profile-firstname"
      />
      <TextField
        label="Date de naissance"
        value={birth}
        onChangeText={(v) => setBirth(maskBirthInput(v))}
        error={errors.birth}
        hint={birthLocked ? "Non modifiable." : "JJ/MM/AAAA. Seul votre âge sera affiché."}
        disabled={birthLocked}
        keyboardType="number-pad"
        maxLength={10}
        placeholder="JJ/MM/AAAA"
        testID="profile-birth"
      />
      <View style={styles.group}>
        <Text variant="label" tone="muted">
          Je suis
        </Text>
        <View style={styles.wrap}>
          {(Object.keys(GENDER_LABELS) as Gender[]).map((g) => (
            <Chip
              key={g}
              label={GENDER_LABELS[g]}
              selected={gender === g}
              onPress={() => setGender(g)}
              testID={`gender-${g}`}
            />
          ))}
        </View>
        {errors.gender ? (
          <Text variant="caption" tone="danger">
            {errors.gender}
          </Text>
        ) : null}
      </View>
      <View style={styles.group}>
        <TextField
          label="Ville"
          value={city}
          onChangeText={setCity}
          maxLength={80}
          hint="Votre position exacte n'est jamais partagée : seule une distance approximative est visible."
        />
        <Button
          label={coords ? "Mettre à jour ma position" : "Utiliser ma position"}
          variant="secondary"
          onPress={locate}
          loading={locating}
          testID="profile-locate"
        />
        {errors.location ? (
          <Text variant="caption" tone="danger">
            {errors.location}
          </Text>
        ) : null}
      </View>
      <TextField
        label="Bio"
        value={bio}
        onChangeText={setBio}
        multiline
        maxLength={500}
        hint={`${bio.length}/500`}
        style={styles.bio}
        placeholder="Quelques mots sur vous…"
      />
      <View style={styles.group}>
        <Text variant="label" tone="muted">
          Centres d&apos;intérêt ({selected.length}/{MAX_INTERESTS})
        </Text>
        <View style={styles.wrap}>
          {interestsQuery.data?.map((i) => (
            <Chip
              key={i.slug}
              label={i.label}
              selected={selected.includes(i.slug)}
              onPress={() => toggleInterest(i.slug)}
            />
          ))}
        </View>
      </View>
      {errors.form ? <Text tone="danger">{errors.form}</Text> : null}
      <Button
        label={submitLabel}
        onPress={submit}
        loading={save.isPending}
        testID="profile-submit"
      />
    </>
  );
}

export function ProfileSetupScreen({ onDone }: { onDone: () => void }) {
  return (
    <Screen scroll>
      <Text variant="title">Faisons connaissance</Text>
      <Text tone="muted">Étape 1 sur 3. Ces informations apparaîtront sur votre profil.</Text>
      <ProfileForm submitLabel="Continuer" onSaved={onDone} />
    </Screen>
  );
}

const styles = StyleSheet.create({
  group: { gap: spacing.sm },
  wrap: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  bio: { minHeight: 96, textAlignVertical: "top" },
});
