import React, { useEffect, useState } from "react";
import { StyleSheet, View } from "react-native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { profileApi } from "../../api/endpoints";
import { queryKeys } from "../../api/queryKeys";
import type { Gender, MyProfile } from "../../api/types";
import { useAuth } from "../../auth/AuthProvider";
import { useInterests, useMyProfile } from "../../hooks";
import { colors, radii, spacing } from "../../theme";
import { Button, LoadingState, Screen, Text, TextField } from "../../ui";
import { ageFromIso, isoToFrenchDate, parseFrenchDate } from "../../utils/format";
import { BirthdateField, GenderPicker, InterestedInPicker, InterestsPicker, LocationPicker, PhotoGrid } from "../profile/fields";

const STEPS = ["identity", "gender", "photos", "about", "location"] as const;
type Step = (typeof STEPS)[number];

function firstStep(profile: MyProfile): number {
  if (profile.missing.includes("firstName") || profile.missing.includes("birthdate")) return 0;
  if (profile.missing.includes("gender")) return 1;
  if (profile.missing.includes("photos")) return 2;
  return 4;
}

export function OnboardingScreen() {
  const profile = useMyProfile();
  const [step, setStep] = useState<number | null>(null);

  useEffect(() => {
    if (step === null && profile.data) setStep(firstStep(profile.data));
  }, [profile.data, step]);

  if (!profile.data || step === null) {
    return (
      <Screen>
        <LoadingState />
      </Screen>
    );
  }
  const next = () => setStep((s) => Math.min((s ?? 0) + 1, STEPS.length - 1));
  const back = step > 0 ? () => setStep(step - 1) : undefined;
  return <StepView key={STEPS[step]} step={STEPS[step]} index={step} profile={profile.data} onNext={next} onBack={back} />;
}

type StepProps = { step: Step; index: number; profile: MyProfile; onNext: () => void; onBack?: () => void };

function StepView({ step, index, profile, onNext, onBack }: StepProps) {
  const queryClient = useQueryClient();
  const { signOut } = useAuth();
  const interests = useInterests();

  const [firstName, setFirstName] = useState(profile.firstName);
  const [birthdate, setBirthdate] = useState(isoToFrenchDate(profile.birthdate));
  const [gender, setGender] = useState<Gender | null>(profile.gender);
  const [interestedIn, setInterestedIn] = useState<Gender[]>(profile.preferences.interestedIn);
  const [bio, setBio] = useState(profile.bio);
  const [jobTitle, setJobTitle] = useState(profile.jobTitle);
  const [interestIds, setInterestIds] = useState<number[]>(profile.interests.map((i) => i.id));
  const [localError, setLocalError] = useState<string | null>(null);

  const save = useMutation({
    mutationFn: async (): Promise<MyProfile> => {
      switch (step) {
        case "identity": {
          const iso = parseFrenchDate(birthdate);
          return profileApi.update({ firstName: firstName.trim(), birthdate: iso ?? "" });
        }
        case "gender": {
          const updated = await profileApi.update({ gender: gender ?? undefined });
          const age = updated.age ?? 30;
          return profileApi.updatePreferences({
            ...updated.preferences,
            interestedIn,
            ageMin: Math.max(18, age - 8),
            ageMax: Math.min(99, age + 8)
          });
        }
        case "about":
          return profileApi.update({ bio, jobTitle, interestIds });
        default:
          return profile;
      }
    },
    onSuccess: (updated) => {
      queryClient.setQueryData(queryKeys.profile, updated);
      onNext();
    }
  });

  function submit() {
    setLocalError(null);
    if (step === "identity") {
      if (!firstName.trim()) return setLocalError("Indique ton prénom.");
      const iso = parseFrenchDate(birthdate);
      if (!iso) return setLocalError("Date de naissance invalide (JJ/MM/AAAA).");
      if (ageFromIso(iso) < 18) return setLocalError("Lueur est réservé aux personnes majeures (18 ans et plus).");
    }
    if (step === "gender" && !gender) return setLocalError("Choisis ton genre.");
    if (step === "photos") {
      if (profile.photos.length === 0) return setLocalError("Ajoute au moins une photo.");
      return onNext();
    }
    save.mutate();
  }

  const titles: Record<Step, [string, string]> = {
    identity: ["Faisons connaissance", "Ton prénom et ton âge apparaîtront sur ton profil."],
    gender: ["Qui es-tu, qui cherches-tu ?", "Tu pourras modifier ces choix à tout moment."],
    photos: ["Ajoute tes photos", "Jusqu'à 6 photos. La première est ta photo principale."],
    about: ["Parle un peu de toi", "Une bio sincère et quelques passions aident à briser la glace."],
    location: ["Où te trouves-tu ?", "Pour te proposer des personnes près de chez toi."]
  };

  const footer =
    step === "location" ? null : (
      <View style={styles.footerRow}>
        {onBack ? (
          <Button label="Retour" variant="ghost" onPress={onBack} compact />
        ) : (
          <Button label="Déconnexion" variant="ghost" onPress={() => void signOut()} compact />
        )}
        <Button label="Continuer" testID="onboarding-next" loading={save.isPending} onPress={submit} style={styles.flex} />
      </View>
    );

  return (
    <Screen scroll edges={["top", "left", "right", "bottom"]} footer={footer} testID={`onboarding-${step}`}>
      <View style={styles.progress} accessibilityLabel={`Étape ${index + 1} sur ${STEPS.length}`}>
        {STEPS.map((s, i) => (
          <View key={s} style={[styles.progressBar, i <= index ? styles.progressActive : null]} />
        ))}
      </View>
      <View style={styles.header}>
        <Text variant="title">{titles[step][0]}</Text>
        <Text tone="muted">{titles[step][1]}</Text>
      </View>

      <View style={styles.body}>
        {step === "identity" ? (
          <>
            <TextField
              label="Prénom"
              testID="field-first-name"
              value={firstName}
              onChangeText={setFirstName}
              autoComplete="given-name"
              maxLength={40}
            />
            <BirthdateField value={birthdate} onChange={setBirthdate} />
          </>
        ) : null}

        {step === "gender" ? (
          <>
            <Text variant="heading">Je suis</Text>
            <GenderPicker value={gender} onChange={setGender} />
            <Text variant="heading" style={styles.spaced}>
              Je souhaite rencontrer
            </Text>
            <InterestedInPicker value={interestedIn} onChange={setInterestedIn} />
          </>
        ) : null}

        {step === "photos" ? <PhotoGrid photos={profile.photos} /> : null}

        {step === "about" ? (
          <>
            <TextField
              label="Bio"
              testID="field-bio"
              value={bio}
              onChangeText={setBio}
              multiline
              maxLength={500}
              counter={{ max: 500 }}
              placeholder="Ce qui te fait sourire, ton dimanche idéal…"
            />
            <TextField label="Métier (facultatif)" value={jobTitle} onChangeText={setJobTitle} maxLength={80} />
            <Text variant="heading">Centres d’intérêt</Text>
            {interests.data ? <InterestsPicker all={interests.data} selected={interestIds} onChange={setInterestIds} /> : <LoadingState />}
          </>
        ) : null}

        {step === "location" ? (
          <>
            <LocationPicker currentCity={profile.city} hasLocation={profile.hasLocation} />
            {onBack ? <Button label="Retour" variant="ghost" onPress={onBack} /> : null}
          </>
        ) : null}

        {localError ? (
          <Text tone="danger" accessibilityLiveRegion="polite" testID="onboarding-error">
            {localError}
          </Text>
        ) : null}
        {save.isError ? (
          <Text tone="danger" accessibilityLiveRegion="polite">
            {errorMessage(save.error)}
          </Text>
        ) : null}
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  progress: {
    flexDirection: "row",
    gap: spacing.xs,
    marginBottom: spacing.xl
  },
  progressBar: {
    flex: 1,
    height: 4,
    borderRadius: radii.pill,
    backgroundColor: colors.border
  },
  progressActive: {
    backgroundColor: colors.accent
  },
  header: {
    gap: spacing.sm,
    marginBottom: spacing.xl
  },
  body: {
    gap: spacing.lg
  },
  spaced: {
    marginTop: spacing.md
  },
  footerRow: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md
  },
  flex: {
    flex: 1
  }
});
