import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../api/client";
import { saveMyProfile } from "../api/profile";
import type { Gender, MyProfile, Preferences } from "../api/types";
import { BirthDateField, ageFromIso } from "../components/BirthDateField";
import { Chip, ChipRow } from "../components/Chip";
import { InterestPicker } from "../components/InterestPicker";
import { LocationPicker, type Coordinates } from "../components/LocationPicker";
import { NavHeader } from "../components/NavHeader";
import { PhotoManager } from "../components/PhotoManager";
import { PreferencesEditor } from "../components/PreferencesEditor";
import { Screen } from "../components/Screen";
import { GENDERS, GENDER_LABELS, MAX_BIO } from "../components/labels";
import { useAuth } from "../hooks/useAuth";
import { queryKeys } from "../hooks/queries";
import { Button, Input, Notice, Text } from "../shared/ui";

const STEPS = ["À propos de vous", "Vos préférences", "Votre position", "Votre profil", "Vos photos"] as const;
const NAME_PATTERN = /^[\p{L}][\p{L}\p{M} '’.-]*$/u;

type OnboardingScreenProps = {
  /** An existing profile means only the photos are missing. */
  profile: MyProfile | null;
};

export default function OnboardingScreen({ profile }: OnboardingScreenProps) {
  const client = useQueryClient();
  const { signOut } = useAuth();
  const [step, setStep] = useState(profile ? 4 : 0);
  const [tried, setTried] = useState(false);

  const [firstName, setFirstName] = useState(profile?.firstName ?? "");
  const [birthDate, setBirthDate] = useState(profile?.birthDate ?? "");
  const [gender, setGender] = useState<Gender | null>(profile?.gender ?? null);
  const [preferences, setPreferences] = useState<Preferences>(
    profile?.preferences ?? { interestedIn: [], ageMin: 18, ageMax: 50, maxDistanceKm: 50 }
  );
  const [coordinates, setCoordinates] = useState<Coordinates | null>(
    profile ? { latitude: profile.latitude, longitude: profile.longitude } : null
  );
  const [city, setCity] = useState(profile?.city ?? "");
  const [bio, setBio] = useState(profile?.bio ?? "");
  const [interests, setInterests] = useState<string[]>(profile?.interests.map((i) => i.slug) ?? []);

  const age = ageFromIso(birthDate);
  const errors = {
    firstName: !NAME_PATTERN.test(firstName.trim()) ? "Saisissez votre prénom (lettres uniquement)." : null,
    birthDate: !birthDate ? "Saisissez une date valide." : age !== null && age < 18 ? "Vous devez avoir au moins 18 ans." : null,
    gender: !gender ? "Choisissez une option." : null,
    prefs: preferences.interestedIn.length === 0 ? "Choisissez au moins une option." : null,
    location: !coordinates ? "Indiquez votre position ou choisissez une ville." : null
  };
  const stepValid = [
    !errors.firstName && !errors.birthDate && !errors.gender,
    !errors.prefs,
    !errors.location,
    true,
    true
  ];

  const save = useMutation({
    mutationFn: () =>
      saveMyProfile({
        firstName: firstName.trim(),
        birthDate,
        gender: gender as Gender,
        bio: bio.trim(),
        city: city.trim(),
        latitude: coordinates?.latitude ?? 0,
        longitude: coordinates?.longitude ?? 0,
        interests,
        preferences
      }),
    onSuccess: (saved) => {
      client.setQueryData(queryKeys.profile, saved);
      setStep(4);
    }
  });

  const next = () => {
    setTried(true);
    if (!stepValid[step]) {
      return;
    }
    setTried(false);
    if (step === 3) {
      save.mutate();
    } else {
      setStep(step + 1);
    }
  };

  const photos = profile?.photos ?? [];
  const finish = () => client.invalidateQueries({ queryKey: queryKeys.profile });

  return (
    <Screen
      footer={
        step < 4 ? (
          <Button fullWidth label={step === 3 ? "Enregistrer mon profil" : "Continuer"} loading={save.isPending} onPress={next} size="lg" style={styles.footerButton} testID="onboarding-next" />
        ) : (
          <Button disabled={photos.length === 0} fullWidth label="Commencer à découvrir" onPress={finish} size="lg" style={styles.footerButton} testID="onboarding-finish" />
        )
      }
      header={
        <NavHeader
          onBack={step > 0 && step < 4 ? () => setStep(step - 1) : undefined}
          right={step === 0 || step === 4 ? <Button label="Quitter" onPress={() => void signOut()} size="sm" variant="ghost" /> : undefined}
          title={STEPS[step]}
        />
      }
      scroll
      testID={`onboarding-step-${step}`}
    >
      <Text tone="muted" variant="caption">
        Étape {step + 1} sur {STEPS.length}
      </Text>

      {step === 0 ? (
        <>
          <Input autoCapitalize="words" error={tried ? errors.firstName : null} label="Prénom" maxLength={40} onChangeText={setFirstName} testID="ob-name" value={firstName} />
          <BirthDateField error={tried ? errors.birthDate : null} onChange={setBirthDate} value={birthDate} />
          <View style={styles.block}>
            <Text tone="secondary" variant="label" weight="semibold">
              Je suis
            </Text>
            <ChipRow>
              {GENDERS.map((g) => (
                <Chip key={g} label={GENDER_LABELS[g]} onPress={() => setGender(g)} selected={gender === g} testID={`gender-${g}`} />
              ))}
            </ChipRow>
            {tried && errors.gender ? (
              <Text tone="danger" variant="caption">
                {errors.gender}
              </Text>
            ) : null}
          </View>
        </>
      ) : null}

      {step === 1 ? (
        <>
          {tried && errors.prefs ? <Notice title={errors.prefs} tone="danger" /> : null}
          <PreferencesEditor onChange={setPreferences} value={preferences} />
        </>
      ) : null}

      {step === 2 ? (
        <>
          {tried && errors.location ? <Notice title={errors.location} tone="danger" /> : null}
          <LocationPicker
            city={city}
            coordinates={coordinates}
            onChange={(coords, detectedCity) => {
              setCoordinates(coords);
              if (detectedCity) setCity(detectedCity);
            }}
            onCityChange={setCity}
          />
        </>
      ) : null}

      {step === 3 ? (
        <>
          {save.isError ? <Notice title={errorMessage(save.error)} tone="danger" /> : null}
          <Input
            helperText={`${bio.length}/${MAX_BIO}`}
            label="À propos de moi (facultatif)"
            maxLength={MAX_BIO}
            multiline
            onChangeText={setBio}
            placeholder="Quelques mots sur vous, ce qui vous fait vibrer…"
            testID="ob-bio"
            value={bio}
          />
          <Text tone="secondary" variant="label" weight="semibold">
            Centres d'intérêt
          </Text>
          <InterestPicker onChange={setInterests} selected={interests} />
        </>
      ) : null}

      {step === 4 ? (
        <>
          <Text tone="muted">Ajoutez au moins une photo claire de vous. Vous pourrez en changer à tout moment.</Text>
          <PhotoManager photos={photos} />
        </>
      ) : null}
    </Screen>
  );
}

const styles = StyleSheet.create({
  block: { gap: 10 },
  footerButton: { maxWidth: 560 }
});
