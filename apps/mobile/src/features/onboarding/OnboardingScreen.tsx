import React, { useEffect, useState } from "react";
import { StyleSheet, View } from "react-native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { profileApi } from "../../api/endpoints";
import type { ProfileInput } from "../../api/types";
import { errorMessage } from "../../lib/format";
import { parseBirthdate, splitIsoDate } from "../../lib/validation";
import { useAuth } from "../../providers/AuthProvider";
import { keys } from "../../providers/queryClient";
import { useTheme } from "../../theme";
import { Button, ErrorState, LoadingState, Screen, Text } from "../../ui";
import { PhotoGrid } from "../profile/PhotoGrid";
import {
  AboutFields,
  BasicsFields,
  InterestPicker,
  PreferencesFields,
  type AboutValue,
  type BasicsValue,
  type PreferencesValue
} from "../profile/ProfileFields";
import { useInterests, useMyProfile } from "../profile/queries";
import { useLocationUpdate } from "../profile/useLocationUpdate";

const steps = [
  { title: "Faisons connaissance", subtitle: "Ces informations apparaissent sur ton profil." },
  { title: "Qui aimerais-tu rencontrer ?", subtitle: "Tu pourras changer d’avis à tout moment." },
  { title: "Parle un peu de toi", subtitle: "Une bio sincère et quelques centres d’intérêt font toute la différence." },
  { title: "Ajoute tes photos", subtitle: "Au moins une photo, idéalement trois où l’on voit bien ton visage." },
  { title: "Rencontres à proximité", subtitle: "Ta position sert uniquement à calculer une distance approximative." }
];

export function OnboardingScreen() {
  const { spacing, colors } = useTheme();
  const { refreshUser, signOut } = useAuth();
  const queryClient = useQueryClient();
  const profileQuery = useMyProfile();
  const interestsQuery = useInterests();
  const location = useLocationUpdate();

  const [step, setStep] = useState(0);
  const [initialised, setInitialised] = useState(false);
  const [basics, setBasics] = useState<BasicsValue>({ firstName: "", day: "", month: "", year: "", gender: null });
  const [prefs, setPrefs] = useState<PreferencesValue>({ interestedIn: [], ageMin: 18, ageMax: 45, maxDistanceKm: 50 });
  const [about, setAbout] = useState<AboutValue>({ bio: "", jobTitle: "", city: "", relationshipGoal: null });
  const [interestIds, setInterestIds] = useState<number[]>([]);
  const [showErrors, setShowErrors] = useState(false);

  // Resume an interrupted onboarding with what was already saved.
  useEffect(() => {
    if (initialised || profileQuery.isLoading) {
      return;
    }
    const p = profileQuery.data;
    if (p) {
      setBasics({ firstName: p.firstName, ...splitIsoDate(p.birthdate), gender: p.gender });
      setAbout({ bio: p.bio, jobTitle: p.jobTitle, city: p.city, relationshipGoal: p.relationshipGoal });
      setInterestIds(p.interests.map((i) => i.id));
      if (p.preferences) {
        setPrefs(p.preferences);
        setStep(3);
      }
    }
    setInitialised(true);
  }, [initialised, profileQuery.isLoading, profileQuery.data]);

  const birth = parseBirthdate(basics.day, basics.month, basics.year);
  const basicsErrors = {
    firstName: basics.firstName.trim() ? null : "Indique ton prénom.",
    birthdate: birth.ok ? null : birth.error,
    gender: basics.gender ? null : "Choisis une option."
  };
  const prefsError = prefs.interestedIn.length ? null : "Choisis au moins une option.";

  const saveProfile = useMutation({
    mutationFn: async () => {
      if (!birth.ok || !basics.gender) {
        throw new Error("Informations incomplètes.");
      }
      const input: ProfileInput = {
        firstName: basics.firstName.trim(),
        birthdate: birth.iso,
        gender: basics.gender,
        bio: about.bio.trim(),
        jobTitle: about.jobTitle.trim(),
        city: about.city.trim(),
        heightCm: null,
        relationshipGoal: about.relationshipGoal,
        interestIds
      };
      const saved = await profileApi.save(input);
      await profileApi.savePreferences(prefs);
      return saved;
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: keys.profile });
      setStep(3);
    }
  });

  const finish = useMutation({
    mutationFn: refreshUser
  });

  if (profileQuery.isLoading || !initialised) {
    return (
      <Screen>
        <LoadingState />
      </Screen>
    );
  }
  if (profileQuery.isError) {
    return (
      <Screen>
        <ErrorState message={errorMessage(profileQuery.error)} onRetry={() => void profileQuery.refetch()} />
      </Screen>
    );
  }

  const photos = profileQuery.data?.photos ?? [];
  const next = () => {
    setShowErrors(true);
    if (step === 0 && Object.values(basicsErrors).some(Boolean)) {
      return;
    }
    if (step === 1 && prefsError) {
      return;
    }
    setShowErrors(false);
    if (step === 2) {
      saveProfile.mutate();
      return;
    }
    if (step === 4) {
      finish.mutate();
      return;
    }
    setStep(step + 1);
  };

  const current = steps[step];
  return (
    <Screen edges={["top", "bottom", "left", "right"]} keyboard>
      <View style={[styles.progress, { paddingHorizontal: spacing.lg, paddingTop: spacing.sm }]}>
        {steps.map((s, i) => (
          <View key={s.title} style={[styles.bar, { backgroundColor: i <= step ? colors.primary : colors.border }]} />
        ))}
      </View>
      <View style={styles.flex}>
        <Screen edges={[]} scroll>
          <Text tone="subtle" variant="eyebrow">
            Étape {step + 1} sur {steps.length}
          </Text>
          <Text variant="title">{current.title}</Text>
          <Text tone="muted">{current.subtitle}</Text>

          {step === 0 ? <BasicsFields errors={showErrors ? basicsErrors : {}} onChange={setBasics} value={basics} /> : null}
          {step === 1 ? <PreferencesFields error={showErrors ? prefsError : null} onChange={setPrefs} value={prefs} /> : null}
          {step === 2 ? (
            <View style={{ gap: spacing.xl }}>
              <AboutFields onChange={setAbout} value={about} />
              {interestsQuery.data ? <InterestPicker interests={interestsQuery.data} onChange={setInterestIds} selected={interestIds} /> : <LoadingState />}
              {saveProfile.isError ? <Text tone="danger">{errorMessage(saveProfile.error)}</Text> : null}
            </View>
          ) : null}
          {step === 3 ? <PhotoGrid photos={photos} /> : null}
          {step === 4 ? (
            <View style={{ gap: spacing.md }}>
              <Text tone="muted">
                Nous n’enregistrons jamais ta position exacte : elle est arrondie à environ 1 km, et les autres membres ne voient qu’une distance approximative.
                Tu peux la retirer à tout moment dans les réglages.
              </Text>
              {profileQuery.data?.hasLocation ? (
                <Text tone="success" variant="bodyStrong">
                  Position approximative enregistrée ✓
                </Text>
              ) : (
                <Button
                  icon="navigate-outline"
                  label="Utiliser ma position"
                  loading={location.pending}
                  onPress={() => void location.update()}
                  variant="secondary"
                />
              )}
              {finish.isError ? <Text tone="danger">{errorMessage(finish.error)}</Text> : null}
            </View>
          ) : null}
        </Screen>
      </View>
      <View style={[styles.footer, { padding: spacing.lg, gap: spacing.sm, borderTopColor: colors.border }]}>
        <Button
          disabled={step === 3 && photos.length === 0}
          label={
            step === 4
              ? profileQuery.data?.hasLocation
                ? "Découvrir des profils"
                : "Continuer sans position"
              : step === 3 && photos.length === 0
                ? "Ajoute au moins une photo"
                : "Continuer"
          }
          loading={saveProfile.isPending || finish.isPending}
          onPress={next}
          testID="onboarding-next"
        />
        {step > 0 && step !== 3 ? (
          <Button label="Retour" onPress={() => setStep(step === 4 ? 3 : step - 1)} variant="ghost" />
        ) : step === 0 ? (
          <Button label="Se déconnecter" onPress={() => void signOut()} variant="ghost" />
        ) : (
          <Button label="Modifier mes informations" onPress={() => setStep(2)} variant="ghost" />
        )}
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  progress: { flexDirection: "row", gap: 6 },
  bar: { flex: 1, height: 4, borderRadius: 2 },
  footer: { borderTopWidth: StyleSheet.hairlineWidth }
});
