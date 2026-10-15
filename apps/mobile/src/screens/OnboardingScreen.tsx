import React, { useEffect, useState } from "react";
import { View } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import { saveLocation, savePreferences, saveProfile } from "../api/platform";
import type { Gender, MyProfileResponse, PhotoView } from "../api/types";
import { BirthDateInput } from "../components/BirthDateInput";
import { OptionCard } from "../components/OptionCard";
import { PhotoManager } from "../components/PhotoManager";
import { Stepper } from "../components/Stepper";
import { useAuth } from "../hooks/useAuth";
import { queryKeys, useInterests, useMyProfile } from "../hooks/queries";
import { errorMessage } from "../i18n/errors";
import { APP_NAME, distanceChoices, genderLabels, interestedInLabels } from "../i18n/fr";
import { checkBirthDate, fromIsoDate, type BirthDateParts } from "../lib/dates";
import { readApproximatePosition } from "../lib/location";
import { cleanName, isValidFirstName } from "../lib/validation";
import { ErrorView, LoadingView } from "../shared/feedback";
import { ScreenContainer } from "../shared/layout";
import { Button, Chip, FormField, Input, Notice, Text, spacing } from "../shared/ui";
import { useTheme } from "../theme";

const STEPS = ["Vous", "À propos", "Recherche", "Position", "Photos"] as const;
const GENDERS: Gender[] = ["woman", "man", "nonbinary"];
const MAX_INTERESTS = 10;
const BIO_MAX = 500;

function firstIncompleteStep(data: MyProfileResponse | undefined): number {
  if (!data?.onboarding.profile) return 0;
  if (!data.onboarding.preferences) return 2;
  return data.profile?.hasLocation ? 4 : 3;
}

export default function OnboardingScreen() {
  const palette = useTheme();
  const queryClient = useQueryClient();
  const { signOut } = useAuth();
  const profileQuery = useMyProfile();
  const interestsQuery = useInterests();

  const [step, setStep] = useState<number | null>(null);
  const [firstName, setFirstName] = useState("");
  const [birth, setBirth] = useState<BirthDateParts>({ day: "", month: "", year: "" });
  const [gender, setGender] = useState<Gender | null>(null);
  const [bio, setBio] = useState("");
  const [interests, setInterests] = useState<string[]>([]);
  const [seeks, setSeeks] = useState<Gender[]>([]);
  const [minAge, setMinAge] = useState(22);
  const [maxAge, setMaxAge] = useState(40);
  const [distance, setDistance] = useState(50);
  const [photos, setPhotos] = useState<PhotoView[]>([]);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [locationNote, setLocationNote] = useState<string | null>(null);
  const [locationSaved, setLocationSaved] = useState(false);

  // Seed the form once from whatever the server already knows (resume after a crash or a reinstall).
  const data = profileQuery.data;
  useEffect(() => {
    if (!data || step !== null) {
      return;
    }
    const profile = data.profile;
    if (profile) {
      setFirstName(profile.firstName);
      setBirth(fromIsoDate(profile.birthDate));
      setGender(profile.gender);
      setBio(profile.bio);
      setInterests(profile.interests.map((i) => i.slug));
      setPhotos(profile.photos);
      setLocationSaved(profile.hasLocation);
      if (profile.preferences) {
        setSeeks(profile.preferences.interestedIn);
        setMinAge(profile.preferences.minAge);
        setMaxAge(profile.preferences.maxAge);
        setDistance(profile.preferences.maxDistanceKm);
      }
    }
    setStep(firstIncompleteStep(data));
  }, [data, step]);

  if (profileQuery.isError) {
    return (
      <ScreenContainer>
        <ErrorView error={profileQuery.error} onRetry={() => void profileQuery.refetch()} />
      </ScreenContainer>
    );
  }
  if (step === null) {
    return (
      <ScreenContainer>
        <LoadingView fullScreen label="Préparation…" />
      </ScreenContainer>
    );
  }

  const profileExists = Boolean(data?.profile);
  const birthCheck = checkBirthDate(birth);
  const birthError =
    birthCheck.status === "underage"
      ? "Alma est réservée aux personnes de 18 ans et plus."
      : birthCheck.status === "too_old" || birthCheck.status === "invalid"
        ? "Saisissez une date de naissance valide."
        : null;

  const store = (response: MyProfileResponse) => queryClient.setQueryData(queryKeys.myProfile, response);

  const next = async () => {
    setError(null);
    setBusy(true);
    try {
      if (step === 0) {
        if (!isValidFirstName(firstName)) {
          setError("Indiquez votre prénom (40 caractères maximum).");
          return;
        }
        if (birthCheck.status !== "ok") {
          setError(birthError ?? "Saisissez une date de naissance valide.");
          return;
        }
        if (!gender) {
          setError("Choisissez ce qui vous correspond.");
          return;
        }
      }
      if (step === 1) {
        if (!gender || birthCheck.status !== "ok" || !birthCheck.iso) {
          setStep(0);
          return;
        }
        store(await saveProfile({ firstName: cleanName(firstName), birthDate: birthCheck.iso, gender, bio: bio.trim(), interests }));
      }
      if (step === 2) {
        if (seeks.length === 0) {
          setError("Choisissez au moins une option.");
          return;
        }
        store(await savePreferences({ interestedIn: seeks, minAge, maxAge, maxDistanceKm: distance }));
      }
      setStep(step + 1);
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setBusy(false);
    }
  };

  const locateMe = async () => {
    setError(null);
    setLocationNote(null);
    setBusy(true);
    try {
      const position = await readApproximatePosition();
      if (position.status === "denied") {
        setLocationNote("Position refusée. Vous pourrez l'activer plus tard dans vos préférences.");
        return;
      }
      if (position.status === "unavailable") {
        setLocationNote("Position indisponible pour le moment. Réessayez ou passez cette étape.");
        return;
      }
      store(await saveLocation(position.latitude, position.longitude, position.city));
      setLocationSaved(true);
      setStep(4);
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setBusy(false);
    }
  };

  const finish = async () => {
    if (photos.length === 0) {
      setError("Ajoutez au moins une photo pour continuer.");
      return;
    }
    setBusy(true);
    try {
      // The navigator reacts to onboarding.complete and moves to the main tabs.
      await queryClient.invalidateQueries({ queryKey: queryKeys.myProfile });
    } finally {
      setBusy(false);
    }
  };

  const toggleInterest = (slug: string) =>
    setInterests((current) =>
      current.includes(slug) ? current.filter((s) => s !== slug) : current.length >= MAX_INTERESTS ? current : [...current, slug]
    );
  const toggleSeek = (value: Gender) =>
    setSeeks((current) => (current.includes(value) ? current.filter((g) => g !== value) : [...current, value]));

  return (
    <ScreenContainer keyboardAware scroll testID="onboarding-screen">
      <View style={{ gap: spacing.sm, paddingTop: spacing.md }}>
        <View style={{ flexDirection: "row", gap: 6 }} accessibilityLabel={`Étape ${step + 1} sur ${STEPS.length}`}>
          {STEPS.map((label, index) => (
            <View
              key={label}
              style={{ flex: 1, height: 5, borderRadius: 3, backgroundColor: index <= step ? palette.primary : palette.border }}
            />
          ))}
        </View>
        <Text tone="muted" variant="caption">
          Étape {step + 1} sur {STEPS.length} · {STEPS[step]}
        </Text>
      </View>

      {step === 0 ? (
        <View style={{ gap: spacing.lg }}>
          <View style={{ gap: spacing.xs }}>
            <Text accessibilityRole="header" variant="title" weight="bold">
              Bienvenue sur {APP_NAME}
            </Text>
            <Text tone="muted">Commençons par l&apos;essentiel. Ces informations seront visibles sur votre profil.</Text>
          </View>
          <FormField
            label="Prénom"
            maxLength={40}
            onChangeText={setFirstName}
            testID="onboarding-first-name"
            textContentType="givenName"
            value={firstName}
          />
          <View style={{ gap: spacing.xs }}>
            <Text tone="muted" variant="label" weight="medium">
              Date de naissance
            </Text>
            <BirthDateInput
              disabled={profileExists}
              error={birth.day || birth.month || birth.year ? birthError : null}
              onChange={setBirth}
              value={birth}
            />
            <Text tone="subtle" variant="caption">
              {profileExists
                ? "La date de naissance ne peut plus être modifiée."
                : "Seul votre âge sera affiché. Vous devez avoir 18 ans ou plus."}
            </Text>
          </View>
          <View style={{ gap: spacing.sm }}>
            <Text tone="muted" variant="label" weight="medium">
              Je suis…
            </Text>
            {GENDERS.map((value) => (
              <OptionCard
                key={value}
                label={genderLabels[value]}
                onPress={() => setGender(value)}
                selected={gender === value}
                testID={`gender-${value}`}
              />
            ))}
          </View>
        </View>
      ) : null}

      {step === 1 ? (
        <View style={{ gap: spacing.lg }}>
          <View style={{ gap: spacing.xs }}>
            <Text accessibilityRole="header" variant="title" weight="bold">
              Parlez-nous de vous
            </Text>
            <Text tone="muted">Quelques mots suffisent. Vous pourrez modifier tout cela plus tard.</Text>
          </View>
          <View style={{ gap: spacing.xs }}>
            <Text tone="muted" variant="label" weight="medium">
              Présentation
            </Text>
            <Input
              accessibilityLabel="Présentation"
              maxLength={BIO_MAX}
              multiline
              onChangeText={setBio}
              placeholder="Ce qui vous passionne, ce que vous cherchez…"
              testID="onboarding-bio"
              value={bio}
            />
            <Text style={{ textAlign: "right" }} tone="subtle" variant="caption">
              {bio.length}/{BIO_MAX}
            </Text>
          </View>
          <View style={{ gap: spacing.sm }}>
            <Text tone="muted" variant="label" weight="medium">
              Centres d&apos;intérêt ({interests.length}/{MAX_INTERESTS})
            </Text>
            {interestsQuery.isLoading ? <LoadingView /> : null}
            {interestsQuery.isError ? <ErrorView error={interestsQuery.error} onRetry={() => void interestsQuery.refetch()} /> : null}
            <View style={{ flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }}>
              {interestsQuery.data?.map((interest) => (
                <Chip
                  disabled={!interests.includes(interest.slug) && interests.length >= MAX_INTERESTS}
                  key={interest.slug}
                  label={interest.label}
                  onPress={() => toggleInterest(interest.slug)}
                  selected={interests.includes(interest.slug)}
                />
              ))}
            </View>
          </View>
        </View>
      ) : null}

      {step === 2 ? (
        <View style={{ gap: spacing.lg }}>
          <View style={{ gap: spacing.xs }}>
            <Text accessibilityRole="header" variant="title" weight="bold">
              Qui souhaitez-vous rencontrer ?
            </Text>
            <Text tone="muted">Vous ne verrez que des profils qui vous correspondent, et eux aussi.</Text>
          </View>
          <View style={{ gap: spacing.sm }}>
            {GENDERS.map((value) => (
              <OptionCard
                key={value}
                label={interestedInLabels[value]}
                mode="check"
                onPress={() => toggleSeek(value)}
                selected={seeks.includes(value)}
                testID={`seeks-${value}`}
              />
            ))}
          </View>
          <Stepper label="Âge minimum" max={Math.min(maxAge, 99)} min={18} onChange={setMinAge} suffix=" ans" value={minAge} />
          <Stepper label="Âge maximum" max={99} min={Math.max(minAge, 18)} onChange={setMaxAge} suffix=" ans" value={maxAge} />
          <View style={{ gap: spacing.sm }}>
            <Text weight="semibold">Distance maximale</Text>
            <View style={{ flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }}>
              {distanceChoices.map((km) => (
                <Chip key={km} label={`${km} km`} onPress={() => setDistance(km)} selected={distance === km} />
              ))}
            </View>
          </View>
        </View>
      ) : null}

      {step === 3 ? (
        <View style={{ gap: spacing.lg }}>
          <View style={{ gap: spacing.xs }}>
            <Text accessibilityRole="header" variant="title" weight="bold">
              Où êtes-vous ?
            </Text>
            <Text tone="muted">
              Pour vous proposer des personnes proches. Votre position est arrondie à environ 1 km et n&apos;est jamais affichée : les
              autres voient seulement une distance approximative.
            </Text>
          </View>
          {locationSaved ? <Notice message="Votre position approximative est enregistrée." tone="success" /> : null}
          {locationNote ? <Notice message={locationNote} tone="warning" /> : null}
          <Button
            fullWidth
            label={locationSaved ? "Mettre à jour ma position" : "Utiliser ma position"}
            loading={busy}
            onPress={() => void locateMe()}
            size="lg"
            testID="onboarding-location"
          />
          <Button fullWidth label={locationSaved ? "Continuer" : "Passer pour l'instant"} onPress={() => setStep(4)} variant="ghost" />
          {!locationSaved ? (
            <Text tone="subtle" variant="caption">
              Sans position, le filtre de distance ne s&apos;applique pas et aucune distance n&apos;est affichée.
            </Text>
          ) : null}
        </View>
      ) : null}

      {step === 4 ? (
        <View style={{ gap: spacing.lg }}>
          <View style={{ gap: spacing.xs }}>
            <Text accessibilityRole="header" variant="title" weight="bold">
              Vos plus belles photos
            </Text>
            <Text tone="muted">Ajoutez au moins une photo où l&apos;on vous voit bien. Maximum 6.</Text>
          </View>
          <PhotoManager onChanged={setPhotos} photos={photos} />
        </View>
      ) : null}

      {error ? <Notice message={error} tone="danger" /> : null}

      {step !== 3 ? (
        <View style={{ gap: spacing.sm, marginTop: spacing.md }}>
          {step === 4 ? (
            <Button fullWidth label="Terminer" loading={busy} onPress={() => void finish()} size="lg" testID="onboarding-finish" />
          ) : (
            <Button fullWidth label="Continuer" loading={busy} onPress={() => void next()} size="lg" testID="onboarding-next" />
          )}
          {step > 0 ? <Button fullWidth label="Retour" onPress={() => setStep(step - 1)} variant="ghost" /> : null}
          {step === 0 ? <Button fullWidth label="Se déconnecter" onPress={() => void signOut()} size="sm" variant="ghost" /> : null}
        </View>
      ) : (
        <Button fullWidth label="Retour" onPress={() => setStep(2)} variant="ghost" />
      )}
    </ScreenContainer>
  );
}
