import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { profileApi } from "../api/endpoints";
import type { Gender, OwnProfile } from "../api/types";
import { InterestPicker } from "../components/InterestPicker";
import { errorMessage, keys, useProfile } from "../hooks/queries";
import { readCoarsePosition } from "../hooks/useLocation";
import { isoToDisplay } from "../lib/dates";
import { validateDisplayName } from "../lib/validation";
import type { MainStackParamList } from "../navigation/types";
import { discoverStore } from "../state/discoverStore";
import { Button, Chip, ErrorView, Input, LoadingView, Screen, Text, spacing, useToast } from "../ui";

const GENDERS: { value: Gender; label: string }[] = [
  { value: "woman", label: "Une femme" },
  { value: "man", label: "Un homme" },
  { value: "non_binary", label: "Non-binaire" }
];

type Props = NativeStackScreenProps<MainStackParamList, "EditProfile">;

function Form({ profile, onSaved }: { profile: OwnProfile; onSaved: () => void }) {
  const toast = useToast();
  const queryClient = useQueryClient();
  const [name, setName] = useState(profile.displayName);
  const [gender, setGender] = useState<Gender>(profile.gender);
  const [bio, setBio] = useState(profile.bio);
  const [city, setCity] = useState(profile.city);
  const [interests, setInterests] = useState(profile.interests.map((i) => i.slug));
  const [coords, setCoords] = useState<{ latitude: number; longitude: number } | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const locate = async () => {
    try {
      const found = await readCoarsePosition();
      setCoords({ latitude: found.latitude, longitude: found.longitude });
      if (found.city) setCity(found.city);
      toast.show("Position actualisée. Enregistrez pour confirmer.");
    } catch (e) {
      toast.show(errorMessage(e), "error");
    }
  };

  const save = async () => {
    const invalid = validateDisplayName(name);
    if (invalid) return setError(invalid);
    setError(null);
    setBusy(true);
    try {
      await profileApi.save({
        displayName: name.trim(),
        birthDate: profile.birthDate,
        gender,
        bio: bio.trim(),
        city: city.trim(),
        interests,
        ...(coords ?? {})
      });
      await queryClient.invalidateQueries({ queryKey: keys.profile });
      if (coords || gender !== profile.gender) discoverStore.refresh();
      toast.show("Profil enregistré.");
      onSaved();
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <View style={styles.form}>
      <Input label="Prénom" maxLength={50} onChangeText={setName} testID="edit-name" value={name} />
      <Input
        editable={false}
        hint="La date de naissance ne peut pas être modifiée."
        label="Date de naissance"
        value={isoToDisplay(profile.birthDate)}
      />
      <Text variant="label" tone="muted">
        Je suis
      </Text>
      <View style={styles.chips}>
        {GENDERS.map((g) => (
          <Chip key={g.value} label={g.label} onPress={() => setGender(g.value)} selected={gender === g.value} />
        ))}
      </View>
      <Input label="Présentation" maxLength={500} multiline onChangeText={setBio} testID="edit-bio" value={bio} />
      <Input label="Ville" maxLength={100} onChangeText={setCity} value={city} />
      <Button label="Actualiser ma position" onPress={locate} small variant="secondary" />
      <Text variant="heading">Centres d'intérêt</Text>
      <InterestPicker onChange={setInterests} selected={interests} />
      {error ? (
        <Text accessibilityRole="alert" tone="danger">
          {error}
        </Text>
      ) : null}
      <Button label="Enregistrer" loading={busy} onPress={save} testID="edit-save" />
    </View>
  );
}

export default function EditProfileScreen({ navigation }: Props) {
  const query = useProfile();
  if (query.isLoading) return <LoadingView />;
  if (!query.data) return <ErrorView message={errorMessage(query.error)} onRetry={() => void query.refetch()} />;
  return (
    <Screen scroll testID="edit-profile">
      <Form onSaved={() => navigation.goBack()} profile={query.data} />
    </Screen>
  );
}

const styles = StyleSheet.create({
  form: { gap: spacing.lg, paddingTop: spacing.lg },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }
});
