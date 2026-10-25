import React, { useState } from "react";
import { Alert } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { profileApi, queryKeys, type ProfileState } from "../../api";
import { MAX_BIO_LENGTH, MAX_FIRST_NAME_LENGTH } from "../../config";
import { Button, Text, TextField } from "../../shared/ui";
import { parseBirthDate, validateBio, validateFirstName } from "../../shared/validation";
import type { OnboardingStackParamList } from "../../navigation/types";
import { BirthDateField } from "../profile/components/BirthDateField";
import { GenderPicker } from "../profile/components/GenderPicker";
import { InterestsPicker } from "../profile/components/InterestsPicker";
import { PhotoManager } from "../profile/components/PhotoManager";
import { PreferencesForm } from "../profile/components/PreferencesForm";
import { useProfileState } from "../profile/hooks";
import { detectLocation } from "../profile/location";
import { useOnboarding } from "./OnboardingContext";
import { OnboardingStep } from "./OnboardingStep";

type Props<T extends keyof OnboardingStackParamList> = NativeStackScreenProps<OnboardingStackParamList, T>;

export function BasicsScreen({ navigation }: Props<"Basics">) {
  const { draft, update } = useOnboarding();
  const [errors, setErrors] = useState<{ firstName?: string; birth?: string; gender?: string }>({});

  const next = () => {
    const birth = parseBirthDate(draft.birth.day, draft.birth.month, draft.birth.year);
    const nextErrors = {
      firstName: validateFirstName(draft.firstName) ?? undefined,
      birth: birth.ok ? undefined : birth.error,
      gender: draft.gender ? undefined : "Choisissez une option."
    };
    setErrors(nextErrors);
    if (birth.ok && !nextErrors.firstName && !nextErrors.gender) {
      update({ isoBirthDate: birth.isoDate });
      navigation.navigate("About");
    }
  };

  return (
    <OnboardingStep
      footer={<Button label="Continuer" onPress={next} testID="onboarding-basics-next" />}
      step={1}
      subtitle="Ces informations apparaissent sur votre profil. Votre date de naissance reste privée, seul votre âge est affiché."
      testID="onboarding-basics"
      title="Faisons connaissance"
    >
      <TextField
        autoCapitalize="words"
        error={errors.firstName}
        label="Prénom"
        maxLength={MAX_FIRST_NAME_LENGTH}
        onChangeText={(firstName) => update({ firstName })}
        testID="onboarding-first-name"
        value={draft.firstName}
      />
      <Text color="textMuted" variant="caption">
        Date de naissance
      </Text>
      <BirthDateField error={errors.birth} onChange={(birth) => update({ birth })} value={draft.birth} />
      <GenderPicker error={errors.gender} onChange={(gender) => update({ gender })} value={draft.gender} />
    </OnboardingStep>
  );
}

export function AboutScreen({ navigation }: Props<"About">) {
  const { draft, update } = useOnboarding();
  const queryClient = useQueryClient();
  const [bioError, setBioError] = useState<string | null>(null);
  const existing = useProfileState().data?.profile;

  const save = useMutation({
    mutationFn: profileApi.save,
    onSuccess: (profile) => {
      queryClient.setQueryData<ProfileState>(queryKeys.profile, (state) => ({
        profile,
        preferences: state?.preferences ?? null,
        complete: state?.complete ?? false
      }));
      navigation.navigate("Preferences");
    }
  });

  const next = () => {
    const error = validateBio(draft.bio);
    setBioError(error);
    if (error || !draft.gender) {
      return;
    }
    save.mutate({
      firstName: draft.firstName.trim(),
      birthDate: draft.isoBirthDate,
      gender: draft.gender,
      bio: draft.bio.trim(),
      interests: draft.interests,
      discoverable: existing?.discoverable ?? true,
      showDistance: existing?.showDistance ?? true
    });
  };

  return (
    <OnboardingStep
      footer={<Button label="Continuer" loading={save.isPending} onPress={next} testID="onboarding-about-next" />}
      step={2}
      subtitle="Quelques mots et vos passions aident à briser la glace."
      testID="onboarding-about"
      title="Parlez-nous de vous"
    >
      <TextField
        error={bioError}
        hint={`${draft.bio.length}/${MAX_BIO_LENGTH}`}
        label="Présentation (facultatif)"
        maxLength={MAX_BIO_LENGTH}
        multiline
        onChangeText={(bio) => update({ bio })}
        style={{ minHeight: 96, textAlignVertical: "top" }}
        testID="onboarding-bio"
        value={draft.bio}
      />
      <InterestsPicker onChange={(interests) => update({ interests })} selected={draft.interests} />
    </OnboardingStep>
  );
}

export function PreferencesScreen({ navigation }: Props<"Preferences">) {
  const { draft, update } = useOnboarding();
  const queryClient = useQueryClient();
  const [error, setError] = useState<string | null>(null);

  const save = useMutation({
    mutationFn: profileApi.savePreferences,
    onSuccess: (preferences) => {
      queryClient.setQueryData<ProfileState>(queryKeys.profile, (state) => (state ? { ...state, preferences } : state));
      navigation.navigate("Photos");
    }
  });

  const next = () => {
    if (draft.preferences.interestedIn.length === 0) {
      setError("Choisissez au moins une option.");
      return;
    }
    setError(null);
    save.mutate(draft.preferences);
  };

  return (
    <OnboardingStep
      footer={<Button label="Continuer" loading={save.isPending} onPress={next} testID="onboarding-preferences-next" />}
      step={3}
      subtitle="Vous pourrez les modifier à tout moment."
      testID="onboarding-preferences"
      title="Qui souhaitez-vous rencontrer ?"
    >
      <PreferencesForm onChange={(preferences) => update({ preferences })} value={draft.preferences} />
      {error ? (
        <Text accessibilityLiveRegion="polite" color="danger" variant="caption">
          {error}
        </Text>
      ) : null}
    </OnboardingStep>
  );
}

export function PhotosScreen({ navigation }: Props<"Photos">) {
  const photos = useProfileState().data?.profile?.photos ?? [];
  return (
    <OnboardingStep
      footer={
        <Button
          disabled={photos.length === 0}
          label="Continuer"
          onPress={() => navigation.navigate("Location")}
          testID="onboarding-photos-next"
        />
      }
      step={4}
      subtitle="Ajoutez au moins une photo. La première sera votre photo principale."
      testID="onboarding-photos"
      title="Vos plus belles photos"
    >
      <PhotoManager photos={photos} />
    </OnboardingStep>
  );
}

export function LocationScreen() {
  const { finish } = useOnboarding();
  const queryClient = useQueryClient();

  const locate = useMutation({
    mutationFn: async () => {
      const outcome = await detectLocation();
      if (outcome.status === "granted") {
        const profile = await profileApi.setLocation(outcome.location);
        queryClient.setQueryData<ProfileState>(queryKeys.profile, (state) => (state ? { ...state, profile } : state));
      }
      return outcome.status;
    },
    onSuccess: (status) => {
      if (status === "granted") {
        finish();
        return;
      }
      Alert.alert(
        status === "denied" ? "Localisation refusée" : "Position introuvable",
        "Vous pourrez l'activer plus tard depuis votre profil. Continuer sans localisation ?",
        [
          { text: "Annuler", style: "cancel" },
          { text: "Continuer", onPress: finish }
        ]
      );
    }
  });

  return (
    <OnboardingStep
      footer={
        <>
          <Button
            label="Activer la localisation"
            loading={locate.isPending}
            onPress={() => locate.mutate()}
            testID="onboarding-location-enable"
          />
          <Button label="Passer pour l'instant" onPress={finish} testID="onboarding-location-skip" variant="ghost" />
        </>
      }
      step={5}
      subtitle="Pour vous montrer des personnes près de chez vous. Seule une distance approximative est affichée, jamais votre position exacte."
      testID="onboarding-location"
      title="Autour de vous"
    >
      <Text color="textMuted">
        Cette étape est facultative : sans localisation, vous voyez des profils sans filtre de distance.
      </Text>
    </OnboardingStep>
  );
}
