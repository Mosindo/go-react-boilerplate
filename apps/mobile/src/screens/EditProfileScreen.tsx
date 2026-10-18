import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../api/client";
import { saveMyProfile } from "../api/profile";
import type { Gender, MyProfile, Preferences } from "../api/types";
import { Chip, ChipRow } from "../components/Chip";
import { InterestPicker } from "../components/InterestPicker";
import { LocationPicker, type Coordinates } from "../components/LocationPicker";
import { NavHeader } from "../components/NavHeader";
import { PhotoManager } from "../components/PhotoManager";
import { PreferencesEditor } from "../components/PreferencesEditor";
import { Screen } from "../components/Screen";
import { GENDERS, GENDER_LABELS, MAX_BIO } from "../components/labels";
import { queryKeys, useMyProfile } from "../hooks/queries";
import type { MainStackParamList } from "../navigation/types";
import { showToast } from "../shared/feedback";
import { Button, Input, Loader, Notice, Text, spacing } from "../shared/ui";
import { formatLongDate } from "../utils/format";

type Props = NativeStackScreenProps<MainStackParamList, "EditProfile">;

const NAME_PATTERN = /^[\p{L}][\p{L}\p{M} '’.-]*$/u;

export default function EditProfileScreen({ navigation }: Props) {
  const { data: profile } = useMyProfile();
  return (
    <Screen
      header={<NavHeader onBack={() => navigation.goBack()} title="Modifier mon profil" />}
      scroll
      testID="edit-profile-screen"
    >
      {profile ? <EditForm onSaved={() => navigation.goBack()} profile={profile} /> : <Loader label="Chargement…" />}
    </Screen>
  );
}

function EditForm({ profile, onSaved }: { profile: MyProfile; onSaved: () => void }) {
  const client = useQueryClient();
  const [firstName, setFirstName] = useState(profile.firstName);
  const [gender, setGender] = useState<Gender>(profile.gender);
  const [bio, setBio] = useState(profile.bio);
  const [interests, setInterests] = useState(profile.interests.map((i) => i.slug));
  const [preferences, setPreferences] = useState<Preferences>(profile.preferences);
  const [coordinates, setCoordinates] = useState<Coordinates>({
    latitude: profile.latitude,
    longitude: profile.longitude
  });
  const [city, setCity] = useState(profile.city);
  const [tried, setTried] = useState(false);

  const nameError = !NAME_PATTERN.test(firstName.trim()) ? "Saisissez votre prénom (lettres uniquement)." : null;

  const save = useMutation({
    mutationFn: () =>
      saveMyProfile({
        firstName: firstName.trim(),
        birthDate: profile.birthDate,
        gender,
        bio: bio.trim(),
        city: city.trim(),
        latitude: coordinates.latitude,
        longitude: coordinates.longitude,
        interests,
        preferences
      }),
    onSuccess: (saved) => {
      client.setQueryData(queryKeys.profile, saved);
      // Preferences and location change who is shown: refresh the deck on next visit.
      void client.invalidateQueries({ queryKey: queryKeys.discover });
      showToast("Profil enregistré.", "success");
      onSaved();
    }
  });

  const submit = () => {
    setTried(true);
    if (!nameError) save.mutate();
  };

  return (
    <>
      {save.isError ? <Notice title={errorMessage(save.error)} tone="danger" /> : null}

      <Section title="Photos">
        <PhotoManager photos={profile.photos} />
      </Section>

      <Section title="À propos">
        <Input
          error={tried ? nameError : null}
          label="Prénom"
          maxLength={40}
          onChangeText={setFirstName}
          testID="edit-name"
          value={firstName}
        />
        <Text tone="muted" variant="caption">
          Né(e) le {formatLongDate(profile.birthDate)} · la date de naissance ne peut pas être modifiée.
        </Text>
        <Text tone="secondary" variant="label" weight="semibold">
          Je suis
        </Text>
        <ChipRow>
          {GENDERS.map((g) => (
            <Chip
              key={g}
              label={GENDER_LABELS[g]}
              onPress={() => setGender(g)}
              selected={gender === g}
              testID={`edit-gender-${g}`}
            />
          ))}
        </ChipRow>
        <Input
          helperText={`${bio.length}/${MAX_BIO}`}
          label="Bio"
          maxLength={MAX_BIO}
          multiline
          onChangeText={setBio}
          testID="edit-bio"
          value={bio}
        />
      </Section>

      <Section title="Centres d'intérêt">
        <InterestPicker onChange={setInterests} selected={interests} />
      </Section>

      <Section title="Mes critères">
        <PreferencesEditor onChange={setPreferences} value={preferences} />
      </Section>

      <Section title="Position">
        <LocationPicker
          city={city}
          coordinates={coordinates}
          onChange={(coords, detected) => {
            setCoordinates(coords);
            if (detected) setCity(detected);
          }}
          onCityChange={setCity}
        />
      </Section>

      <Button label="Enregistrer" loading={save.isPending} onPress={submit} size="lg" testID="edit-save" />
    </>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <View style={styles.section}>
      <Text variant="heading" weight="bold">
        {title}
      </Text>
      {children}
    </View>
  );
}

const styles = StyleSheet.create({ section: { gap: spacing.md } });
